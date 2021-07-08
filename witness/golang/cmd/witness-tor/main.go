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

// Package witness-tor is a TOR-ified witness server.
package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/google/trillian-examples/witness/golang/cmd/witness/impl"
	"golang.org/x/mod/sumdb/note"

	"github.com/cretz/bine/tor"
	"github.com/cretz/bine/torutil/ed25519"
	"github.com/ipsn/go-libtor"
)

var (
	port              = flag.Int("port", 80, "Port to listen for requests on via TOR")
	dbFile            = flag.String("db_file", ":memory:", "path to a file to be used as sqlite3 storage for checkpoints, e.g. /tmp/chkpts.db")
	configFile        = flag.String("config_file", "example.conf", "path to a JSON config file that specifies the logs followed by this witness")
	witnessSK         = flag.String("private_key", "", "private signing key for the witness")
	publishTimeout    = flag.Duration("publish_timeout", 5*time.Minute, "Maximum time to wait for service to be published on TOR network")
	torPrivateKeyFile = flag.String("tor_private_key_file", "", "Path to file to load TOR ed25519 private key from, if no file exists one will be created with a new key")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	if *witnessSK == "" {
		glog.Exitf("--private_key must not be empty")
	}
	signer, err := note.NewSigner(*witnessSK)
	if err != nil {
		glog.Exitf("Error forming a signer: %v", err)
	}

	if len(*configFile) == 0 {
		glog.Exit("--config_file must not be empty")
	}
	fileData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		glog.Exitf("Failed to read from config file: %v", err)
	}
	var js impl.LogConfig
	if err := json.Unmarshal(fileData, &js); err != nil {
		glog.Exitf("Failed to parse config file as proper JSON: %v", err)
	}

	// Bring up TOR service:
	// Ensure we've got a stable key so we have a stable onion address:
	torKey, err := getTORKey(*torPrivateKeyFile)
	if err != nil {
		glog.Exitf("Failed to get TOR key: %v", err)
	}

	// Start the TOR service
	glog.Info("Starting and registering onion service, please wait a couple of minutes...")
	t, err := tor.Start(ctx, &tor.StartConf{ProcessCreator: libtor.Creator, DebugWriter: os.Stderr})
	if err != nil {
		glog.Exitf("Unable to start Tor: %v", err)
	}
	defer t.Close()

	// Create a v3 onion service to listen on any port but show as *port, waiting at
	// most a few minutes to publish the service
	listenCtx, listenCancel := context.WithTimeout(context.Background(), *publishTimeout)
	defer listenCancel()
	onion, err := t.Listen(listenCtx, &tor.ListenConf{Version3: true, Key: torKey, RemotePorts: []int{*port}})
	if err != nil {
		glog.Exitf("Failed to create onion service: %v", err)
	}
	defer onion.Close()

	// We're on the air!

	glog.Infof("Witness published at http://%v.onion", onion.ID)

	if err := impl.Main(ctx, impl.ServerOpts{
		Listener: onion,
		DBFile:   *dbFile,
		Signer:   signer,
		Config:   js,
	}); err != nil {
		glog.Exitf("Error running witness: %v", err)
	}
}

// getTORKey returns a TOR service private key, or nil if f is empty.
//
// If a filename is specified, attempts to load an ed25519 key from the file,
// if no file exists at that location, generate a new key and store it in the file.
//
// If no filename is provided, returns nil.
func getTORKey(f string) (crypto.PrivateKey, error) {
	var torKey ed25519.PrivateKey
	if f != "" {
		var err error
		torKey, err = ioutil.ReadFile(f)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("failed to read tor private key from %q: %v", f, err)
			}
			kp, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				return nil, fmt.Errorf("failed to generate new TOR key: %v", err)
			}
			torKey = kp.PrivateKey()
			glog.Infof("Saving newly created TOR service key to %q", f)
			if err := ioutil.WriteFile(f, torKey, 0600); err != nil {
				return nil, fmt.Errorf("failed to write TOR key file %q: %v", f, err)
			}
		}
		return torKey.KeyPair(), nil
	}
	return nil, nil
}
