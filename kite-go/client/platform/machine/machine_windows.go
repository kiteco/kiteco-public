package machine

import (
	"log"

	"github.com/kiteco/kiteco/kite-go/client/internal/reg"
)

// IDIfSet checks and returns a machineid if its been set. It will not attempt to generate one.
func IDIfSet() (string, bool) {
	mid, err := reg.MachineID()
	if err == nil {
		log.Println("got machine ID from registry")
		return mid, true
	}
	return "", false
}

// ID retrieves the current machine's ID from the Windows registry
// An error is returned if that failed
func ID(devmode bool) (string, error) {
	mid, err := reg.MachineID()
	if err == nil {
		log.Println("got machine ID from registry")
		return mid, nil
	}
	if devmode {
		return generateMachineID(), nil
	}
	return "", ErrNoMachineID
}
