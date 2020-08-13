// This package is the entrypoint for the Firmware Transparency personality server.
package main

import (
	"flag"
	"net/http"

	"github.com/golang/glog"
	"github.com/google/trillian-examples/binary_transparency/firmware/cmd/ft_personality/internal"
)

var listenAddr = flag.String("listen", ":8000", "Address:port to listen for requests on")

func main() {
	flag.Parse()
	glog.Infof("Starting FT personality server...")
	srv := &internal.Server{}

	srv.RegisterHandlers()

	glog.Fatal(http.ListenAndServe(*listenAddr, nil))
}
