package sandbox

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

const (
	// MaxTries is the number of ports to try before giving up
	MaxTries = 100
)

// UnusedPort finds a port that is not currently being used. Unfortunately
// there is no way to guarantee that this port will not be in use by the
// time the caller attempts to use it.
func UnusedPort() (int, error) {
	return unusedPortOn("")
}

// unusedPortOn finds a port that is not currently being used on the given
// interface. Unfortunately there is no way to guarantee that this port
// will not be in use by the time the caller attempts to use it.
func unusedPortOn(iface string) (int, error) {
	// Pick a random integer between 1025 and 65535
	const minport = 2<<10 + 1
	const maxport = 2<<16 - 1
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < MaxTries; i++ {
		port := rng.Intn(maxport-minport) + 1025
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", iface, port))
		if err == nil {
			ln.Close()
			return port, nil
		}
		// Otherwise err != nil meaning the port is in use, so try another
	}
	return 0, fmt.Errorf("did not find an unused port after %d tries", MaxTries)
}
