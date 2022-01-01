package machine

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"

	"github.com/kiteco/kiteco/kite-golib/macaddr"
)

// ErrNoMachineID is the error returned when the machine ID cannot be found.
var ErrNoMachineID = errors.New("unable to find machine ID")

// generateMachineID is called when there is no machine ID in dev mode
func generateMachineID() string {
	// default to the primary network interface for backwards compatibility
	// with old clients
	var buf []byte
	addr, err := macaddr.Primary()

	// if we could not find the primary network interface then generate a
	// random string
	if err != nil {
		log.Println("could not get machine ID from MAC address, randomizing...")
		buf = make([]byte, 32)
		rand.Read(buf)
	}

	hash := md5.Sum([]byte(addr))
	return hex.EncodeToString(hash[:])
}
