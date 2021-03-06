// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package audit contains all the components needed to clone the SumDB into a
// local database and verify that the data downloaded matches that guaranteed
// by the Checkpoint advertised by the SumDB service. It also provides support
// for parsing the downloaded data and verifying that no module+version is
// every provided with different checksums.
package audit

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"

	backoff "github.com/cenkalti/backoff/v4"
	"github.com/golang/glog"
	"github.com/google/trillian/merkle/compact"
	"golang.org/x/mod/sumdb/tlog"
	"golang.org/x/sync/errgroup"
)

// Service has all the operations required for an auditor to verifiably clone
// the remote SumDB.
type Service struct {
	localDB *Database
	sumDB   *SumDBClient
	rf      *compact.RangeFactory
	height  int
}

// NewService constructs a new Service which is ready to go.
func NewService(localDB *Database, sumDB *SumDBClient, height int) *Service {
	rf := &compact.RangeFactory{
		Hash: func(left, right []byte) []byte {
			var lHash, rHash tlog.Hash
			copy(lHash[:], left)
			copy(rHash[:], right)
			thash := tlog.NodeHash(lHash, rHash)
			return thash[:]
		},
	}
	return &Service{
		localDB: localDB,
		sumDB:   sumDB,
		rf:      rf,
		height:  height,
	}
}

// CloneLeafTiles copies the leaf data from the SumDB into the local database.
// It only copies whole tiles, which means that some stragglers may not be
// copied locally.
func (s *Service) CloneLeafTiles(ctx context.Context, checkpoint *tlog.Tree) error {
	head, err := s.localDB.Head()
	if err != nil {
		glog.Infof("failed to find head of database, assuming empty and starting from scratch: %v", err)
		head = -1
	}
	if checkpoint.N < head {
		return fmt.Errorf("illegal state; more leaves locally (%d) than in SumDB (%d)", head, checkpoint.N)
	}
	localLeaves := head + 1

	tileWidth := int64(1 << s.height)
	remainingLeaves := checkpoint.N - localLeaves
	remainingChunks := int(remainingLeaves / tileWidth)
	startOffset := int(localLeaves / tileWidth)

	if remainingChunks > 0 {
		leafChan := make(chan tileLeaves)
		errChan := make(chan error)
		go func() {
			for i := 0; i < remainingChunks; i++ {
				var c tileLeaves
				operation := func() error {
					offset := startOffset + i
					leaves, err := s.sumDB.FullLeavesAtOffset(offset)
					if err != nil {
						return err
					}
					c = tileLeaves{int64(offset) * tileWidth, leaves}
					return nil
				}
				err := backoff.Retry(operation, backoff.NewExponentialBackOff())
				if err != nil {
					errChan <- err
				} else {
					leafChan <- c
				}
			}
		}()

		for i := 0; i < remainingChunks; i++ {
			select {
			case err := <-errChan:
				return err
			case chunk := <-leafChan:
				start, leaves := chunk.start, chunk.data
				err = s.localDB.WriteLeaves(ctx, start, leaves)
				if err != nil {
					return fmt.Errorf("WriteLeaves: %w", err)
				}
			}
		}
	}
	return nil
}

// HashTiles performs a full recalculation of all the tiles using the data from
// the leaves table. Any hashes that no longer match what was previously stored
// will cause an error. Any new hashes will be filled in.
// This could be replaced by something more incremental if the performance is
// unnacceptable. While the SumDB is still reasonably small, this is fine as is.
func (s *Service) HashTiles(ctx context.Context, checkpoint *tlog.Tree) error {
	tileWidth := 1 << s.height
	tileCount := int(checkpoint.N / int64(tileWidth))

	g := new(errgroup.Group)
	roots := make(chan *compact.Range, tileWidth)

	leafTileCount := tileCount
	leafRoots := roots
	g.Go(func() error { return s.hashLeafLevel(leafTileCount, leafRoots) })

	for i := 1; i <= s.getLevelsForLeafCount(checkpoint.N); i++ {
		tileCount /= tileWidth

		thisLevel := i
		thisTileCount := tileCount
		in := roots

		outRoots := make(chan *compact.Range, tileWidth)
		g.Go(func() error { return s.hashUpperLevel(thisLevel, thisTileCount, in, outRoots) })

		roots = outRoots
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to hash: %v", err)
	}
	return nil
}

// CheckRootHash calculates the root hash from the locally generated tiles, and then
// appends any stragglers from the SumDB, returning an error if this calculation
// fails or the result does not match that in the checkpoint provided.
func (s *Service) CheckRootHash(ctx context.Context, checkpoint *tlog.Tree) error {
	logRange := s.rf.NewEmptyRange(0)

	for level := s.getLevelsForLeafCount(checkpoint.N); level >= 0; level-- {
		// how many real leaves a tile at this level covers.
		tileLeafCount := uint64(1) << ((level + 1) * s.height)
		levelTileCount := int(uint64(checkpoint.N) / tileLeafCount)
		firstTileOffset := int(logRange.End() / tileLeafCount)

		for offset := firstTileOffset; offset < levelTileCount; offset++ {
			tHashes, err := s.localDB.Tile(s.height, level, offset)
			if err != nil {
				return fmt.Errorf("failed to get tile L=%d, O=%d: %v", level, offset, err)
			}
			// Calculate this tile as a standalone subtree
			tcr := s.rf.NewEmptyRange(0)
			for _, t := range tHashes {
				tcr.Append(t, nil)
			}
			// Now use the range as what it really is; a commitment to a larger number of leaves
			treeRange, err := s.rf.NewRange(uint64(offset)*tileLeafCount, uint64(offset+1)*tileLeafCount, tcr.Hashes())
			if err != nil {
				return fmt.Errorf("failed to create range for tile L=%d, O=%d: %v", level, offset, err)
			}
			// Append this into the running log range.
			logRange.AppendRange(treeRange, nil)
		}
	}

	stragglersCount := int(uint64(checkpoint.N) - logRange.End())
	stragglerTileOffset := int(checkpoint.N / (1 << s.height))
	stragglers, err := s.sumDB.PartialLeavesAtOffset(stragglerTileOffset, stragglersCount)
	if err != nil {
		return fmt.Errorf("failed to get stragglers: %v", err)
	}
	for _, s := range stragglers {
		sHash := tlog.RecordHash(s)
		logRange.Append(sHash[:], nil)
	}

	if logRange.End() != uint64(checkpoint.N) {
		return fmt.Errorf("calculation error, found %d leaves but expected %d", logRange.End(), checkpoint.N)
	}

	root, err := logRange.GetRootHash(nil)
	if err != nil {
		return fmt.Errorf("failed to get root hash: %v", err)
	}
	var rootHash tlog.Hash
	copy(rootHash[:], root)
	if rootHash != checkpoint.Hash {
		return fmt.Errorf("log root mismatch at tree size %d; calculated %x, SumDB says %x", checkpoint.N, root, checkpoint.Hash[:])
	}
	return nil
}

// VerifyTiles checks that every tile calculated locally matches the result returned
// by SumDB. This shouldn't be necessary if CheckRootHash is working, but this may be
// useful to determine where any corruption has happened in the tree.
func (s *Service) VerifyTiles(ctx context.Context, checkpoint *tlog.Tree) error {
	for level := 0; level <= s.getLevelsForLeafCount(checkpoint.N); level++ {
		finishedLevel := false
		offset := 0
		for !finishedLevel {
			localHashes, err := s.localDB.Tile(s.height, level, offset)
			if err != nil {
				if err == sql.ErrNoRows {
					finishedLevel = true
					continue
				}
				return fmt.Errorf("failed to get tile hashes: %v", err)
			}
			sumDBHashes, err := s.sumDB.TileHashes(level, offset)
			if err != nil {
				return fmt.Errorf("failed to get tile hashes: %v", err)
			}

			for i := 0; i < 1<<s.height; i++ {
				var lHash tlog.Hash
				copy(lHash[:], localHashes[i])
				if sumDBHashes[i] != lHash {
					return fmt.Errorf("found mismatched hash at L=%d, O=%d, leaf=%d\n\tlocal : %x\n\tremote: %x", level, offset, i, sumDBHashes[i][:], localHashes[i])
				}
			}
			offset++
		}
	}
	return nil
}

// ProcessMetadata parses the leaf data and writes the semantic data into the DB.
func (s *Service) ProcessMetadata(ctx context.Context, checkpoint *tlog.Tree) error {
	tileWidth := 1 << s.height
	metadata := make([]Metadata, tileWidth)
	// TODO: skip to head of metadata
	for offset := 0; offset < int(checkpoint.N/int64(tileWidth)); offset++ {
		leafOffset := int64(offset) * int64(tileWidth)
		hashes, err := s.localDB.Leaves(leafOffset, tileWidth)
		if err != nil {
			return err
		}
		for i, h := range hashes {
			leafID := leafOffset + int64(i)

			lines := strings.Split(string(h), "\n")
			tokens := strings.Split(lines[0], " ")
			module, version, repoHash := tokens[0], tokens[1], tokens[2]
			tokens = strings.Split(lines[1], " ")
			if got, want := tokens[0], module; got != want {
				return fmt.Errorf("mismatched module names at %d: (%s, %s)", leafID, got, want)
			}
			if got, want := tokens[1][:len(version)], version; got != want {
				return fmt.Errorf("mismatched version names at %d: (%s, %s)", leafID, got, want)
			}
			modHash := tokens[2]

			metadata[i] = Metadata{module, version, repoHash, modHash}
		}
		if err := s.localDB.SetLeafMetadata(ctx, leafOffset, metadata); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) hashLeafLevel(tileCount int, roots chan<- *compact.Range) error {
	for offset := 0; offset < tileCount; offset++ {
		hashes, err := s.localDB.Tile(s.height, 0, offset)
		if err == sql.ErrNoRows {
			hashes, err = s.hashLeafTile(offset)
		}
		if err != nil {
			return err
		}
		cr := s.rf.NewEmptyRange(uint64(offset) * 1 << s.height)
		for _, h := range hashes {
			cr.Append(h, nil)
		}
		if got, want := len(cr.Hashes()), 1; got != want {
			return fmt.Errorf("expected single root hash but got %d", got)
		}
		roots <- cr
	}
	return nil
}

func (s *Service) hashLeafTile(offset int) ([][]byte, error) {
	tileWidth := 1 << s.height

	leaves, err := s.localDB.Leaves(int64(tileWidth)*int64(offset), tileWidth)
	if err != nil {
		return nil, fmt.Errorf("failed to get leaves from DB: %v", err)
	}
	res := make([][]byte, tileWidth)
	leafHashes := make([]byte, tileWidth*HashLenBytes)
	for i, l := range leaves {
		recordHash := tlog.RecordHash(l)
		res[i] = recordHash[:]
		copy(leafHashes[i*HashLenBytes:], res[i])
	}
	return res, s.localDB.SetTile(s.height, 0, offset, leafHashes)
}

func (s *Service) hashUpperLevel(level, tileCount int, in <-chan *compact.Range, out chan<- *compact.Range) error {
	tileWidth := 1 << s.height

	inHashes := make([][]byte, tileWidth)
	tileHashBlob := make([]byte, tileWidth*HashLenBytes)
	for offset := 0; offset < tileCount; offset++ {
		dbTileHashes, err := s.localDB.Tile(s.height, level, offset)
		found := true
		if err == sql.ErrNoRows {
			found = false
			err = nil
		}
		if err != nil {
			return err
		}
		for i := 0; i < tileWidth; i++ {
			cr := <-in
			inHashes[i] = cr.Hashes()[0]
			copy(tileHashBlob[i*HashLenBytes:], inHashes[i])

			if found && !bytes.Equal(dbTileHashes[i], inHashes[i]) {
				return fmt.Errorf("got diffence in hash at L=%d, O=%d, leaf=%d", level, offset, i)
			}
		}

		if !found {
			if err := s.localDB.SetTile(s.height, level, offset, tileHashBlob); err != nil {
				return fmt.Errorf("failed to set tile at L=%d, O=%d: %v", level, offset, err)
			}
		}
		cr := s.rf.NewEmptyRange(uint64(offset * tileWidth))
		for _, h := range inHashes {
			cr.Append(h, nil)
		}
		if got, want := len(cr.Hashes()), 1; got != want {
			return fmt.Errorf("expected single root hash but got %d", got)
		}
		out <- cr
	}
	return nil
}

// getLevelsForLeafCount determines how many strata of tiles of the configured
// height are needed to contain the largest perfect subtree that can be made of
// the leaves.
func (s *Service) getLevelsForLeafCount(leaves int64) int {
	topLevel := -1
	coveredIdx := leaves >> s.height
	for coveredIdx > 0 {
		coveredIdx = coveredIdx >> s.height
		topLevel++
	}
	return topLevel
}

// tileLeaves is a contiguous block of leaves within a tile.
type tileLeaves struct {
	start int64    // The leaf index of the first leaf
	data  [][]byte // The leaf data
}
