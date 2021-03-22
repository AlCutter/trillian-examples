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

// Package integration provides an integration test for the serverless example.
package integration

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/golang/glog"
	"github.com/google/trillian-examples/serverless/internal/log"
	"github.com/google/trillian-examples/serverless/internal/storage/fs"
	"github.com/google/trillian/merkle/hashers"
	"github.com/google/trillian/merkle/logverifier"
	"github.com/google/trillian/merkle/rfc6962/hasher"
)

func RunIntegration(t *testing.T, s log.Storage) {
	lh := hasher.DefaultHasher
	lv := logverifier.New(lh)

	// Do a few interations around the sequence/integrate loop;
	const (
		loops         = 200
		leavesPerLoop = 200
	)
	for i := 0; i < loops; i++ {
		glog.Infof("----------------%d--------------", i)
		state := s.LogState()

		// Sequence some leaves:
		sequenceNLeaves(t, s, lh, leavesPerLoop)

		// Integrate those leaves
		if err := log.Integrate(s, lh); err != nil {
			t.Fatalf("Integrate = %v", err)
		}

		newState := s.LogState()
		if got, want := newState.Size-state.Size, uint64(leavesPerLoop); got != want {
			t.Errorf("Integrate missed some entries, got %d want %d", got, want)
		}
	}

	// TODO(al): Add inclusion and consistency proof checking here too
}

func TestLogless(t *testing.T) {
	root := filepath.Join(t.TempDir(), "log")

	fs, err := fs.Create(root, []byte("empty"))
	if err != nil {
		t.Fatalf("Create = %v", err)
	}

	RunIntegration(t, fs)
}

func sequenceNLeaves(t *testing.T, s log.Storage, lh hashers.LogHasher, n int) {
	c := make(chan []byte, n)
	for i := 0; i < n; i++ {
		c <- []byte(fmt.Sprintf("Leaf %d", i))
	}
	close(c)
	if err := log.Sequence(s, lh, c); err != nil {
		t.Fatalf("Sequence = %v", err)
	}
}
