// Package internal contains private implementation details for the FirmwareTransparency personality server.
package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/google/trillian-examples/binary_transparency/firmware/api"
)

// Server is the core state & handler implementation of the FT personality.
type Server struct {
}

// addFirmware handles requests to log new firmware images.
// It expects a mime/multipart POST consisting of FirmwareMetadata json,
// followed by a signature over those bytes.
// TODO: store the actual firmware image in a CAS too.
//
// curl -i -X POST -H "Content-Type: multipart/mixed" -F 'json=@testdata/firmware_metadata.json;type=application/json' -F 'sig=@testdata/firmware_metadata.json.sig' localhost:8000/ft/v0/add_firmware
func (s *Server) addFirmware(w http.ResponseWriter, r *http.Request) {

	h := r.Header["Content-Type"]
	if len(h) == 0 {
		http.Error(w, "no content-type header", http.StatusBadRequest)
		return
	}

	mediaType, mediaParams, err := mime.ParseMediaType(h[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(mediaType, "multipart/") {
		http.Error(w, "expecting mime multipart body", http.StatusBadRequest)
		return
	}
	boundary := mediaParams["boundary"]
	if len(boundary) == 0 {
		http.Error(w, "invalid mime multipart header - no boundary specified", http.StatusBadRequest)
		return
	}
	mr := multipart.NewReader(r.Body, boundary)

	// Get raw JSON
	p, err := mr.NextPart() // JSON body section
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rawJSON, err := ioutil.ReadAll(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Get signature
	p, err = mr.NextPart() // Sig section
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rawSig, err := ioutil.ReadAll(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// "Verify" the signature:
	if sigStr := string(rawSig); sigStr != "LOL!" {
		http.Error(w, fmt.Sprintf("invalid LOL! sig %q", sigStr), http.StatusBadRequest)
		return
	}

	// Parse the JSON:
	var meta api.FirmwareMetadata
	if err := json.Unmarshal(rawJSON, &meta); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	glog.V(1).Infof("Got firmware %+v", meta)

	meta.Raw = rawJSON
	meta.Signature = rawSig

}

// getConsitency returns consistency proofs between publised tree sizes.
func (s *Server) getConsistency(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

// getFirmwareEntries returns the leaves in the tree.
func (s *Server) getFirmwareEntries(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

// getRoot returns a recent tree root.
func (s *Server) getRoot(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

// RegisterHandlers registers HTTP handlers for firmware transparency endpoints.
func (s *Server) RegisterHandlers() {
	http.HandleFunc("/ft/v0/add-firmware", s.addFirmware)
	http.HandleFunc("/ft/v0/get-consistency", s.getConsistency)
	http.HandleFunc("/ft/v0/get-firmware-entries", s.getFirmwareEntries)
	http.HandleFunc("/ft/v0/get-root", s.getRoot)
}
