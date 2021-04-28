// Copyright 2021 Google LLC. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package api contains the "public" API/artifacts of the serverless log.
package api

import (
	"bytes"
	"fmt"
	"io"
)

// LogState represents the state of a serverless log
type LogState struct {
	// Size is the number of leaves in the log
	Size uint64

	// SHA256 log root, RFC6962 flavour.
	RootHash []byte
}

// Tile represents a subtree tile, containing inner nodes of a log tree.
type Tile struct {
	// NumLeaves is the number of entries at level 0 of this tile.
	NumLeaves uint

	// Nodes stores the log tree nodes.
	// Nodes are stored linearised using in-order traversal - this isn't completely optimal
	// in terms of storage for partial tiles, but index calculation is relatively
	// straight-forward.
	// Note that only non-ephemeral nodes are stored.
	Nodes [][]byte
}

// MarshalBinary implements encoding/BinaryMarshaller and writes out a Tile
// instance in the following format:
//
// <unsigned byte - hash size in bytes>
// <unsigned byte - number of leaves in tile>
// <node data - hashsize * numleaves * 2 bytes>
func (t Tile) MarshalBinary() ([]byte, error) {
	b := bytes.NewBuffer(make([]byte, 0, len(t.Nodes)*32+2))
	b.WriteByte(32) // SHA256
	b.WriteByte(byte(t.NumLeaves))
	for _, n := range t.Nodes {
		b.Write(n)
	}
	return b.Bytes(), nil
}

// UnmarshalBinary implements encoding/BinaryUnmarshaler and reads tiles
// which were written by the MarshalBinary method above.
func (t *Tile) UnmarshalBinary(raw []byte) error {
	b := bytes.NewBuffer(raw)
	hs, err := b.ReadByte()
	if err != nil {
		return fmt.Errorf("unable to read hashsize: %w", err)
	}
	if hs != 32 {
		return fmt.Errorf("invalid hash size %d", hs)
	}
	numLeaves, err := b.ReadByte()
	if err != nil {
		return fmt.Errorf("unable to read numLeaves: %w", err)
	}
	nodes := make([][]byte, 0, numLeaves*2)
	for {
		h := make([]byte, hs)
		n, err := b.Read(h)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("unable to read node: %w", err)
		}
		if n != int(hs) {
			return fmt.Errorf("short read (%d bytes)", n)
		}
		nodes = append(nodes, h)
	}
	t.NumLeaves, t.Nodes = uint(numLeaves), nodes
	return nil
}

// TileNodeKey generates keys used in Tile.Nodes array.
func TileNodeKey(level uint, index uint64) uint {
	return uint(1<<(level+1)*index + 1<<level - 1)
}
