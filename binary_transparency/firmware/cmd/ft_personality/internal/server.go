// Package internal contains private implementation details for the FirmwareTransparency personality server.
package internal

import (
	"net/http"
)

// Server is the core state & handler implementation of the FT personality.
type Server struct {
}

func (s *Server) addFirmware(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// RegisterHandlers registers HTTP handlers for firmware transparency endpoints.
func (s *Server) RegisterHandlers() {
	http.HandleFunstec("/ft/v0/add_firmware", s.addFirmware)
}
