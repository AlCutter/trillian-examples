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

// Package storage provides common code used by storage implementations.
package storage

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/trillian-examples/serverless/api"
)

// TileSize returns the size of contiguous
func TileSize(t *api.Tile) uint64 {
	for i := uint64(0); i < 256; i++ {
		if t.Nodes[api.TileNodeKey(0, i)] == nil {
			return i
		}
	}
	return 256
}

// TileKey creates a string key for the specified tile address.
func TileKey(level, index uint64) string {
	return fmt.Sprintf("%d/%d", level, index)
}

// SplitTileKey returns the level and index implied by the given tile key.
// This key should have been created with the tileKey function above.
func SplitTileKey(s string) (uint64, uint64) {
	p := strings.Split(s, "/")
	l, err := strconv.ParseUint(p[0], 10, 64)
	if err != nil {
		panic(err)
	}
	i, err := strconv.ParseUint(p[1], 10, 64)
	if err != nil {
		panic(err)
	}
	return l, i
}

// NodeCoordsToTileAddress returns the (TileLevel, TileIndex) in tile-space, and the
// (NodeLevel, NodeIndex) address within that tile of the specified tree node co-ordinates.
func NodeCoordsToTileAddress(treeLevel, treeIndex uint64) (uint64, uint64, uint, uint64) {
	tileRowWidth := uint64(1 << (8 - treeLevel%8))
	tileLevel := treeLevel / 8
	tileIndex := treeIndex / tileRowWidth
	nodeLevel := uint(treeLevel % 8)
	nodeIndex := uint64(treeIndex % tileRowWidth)

	return tileLevel, tileIndex, nodeLevel, nodeIndex
}
