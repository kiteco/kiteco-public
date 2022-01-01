package osutils

import (
	"log"
	"os"
)

var (
	// Hostname is the fully-qualified external hostname or IP for the current
	// server, suitable for use in absolute URLs for links.
	Hostname string
)

// init depends on some knowledge of our server/dev configuration, namely:
//  1. our Vagrant VM for development is named "trusty64" and is exposed
//     to the host machine as 192.168.30.10
//  2. our servers use fully-qualified hostnames (so, 'alpha.kite.com' rather
//     than 'alpha')
func init() {
	var err error
	Hostname, err = os.Hostname()
	if err != nil {
		log.Fatal("error getting hostname:", err)
	}
	if Hostname == "trusty64" {
		Hostname = "192.168.30.10"
	}
}
