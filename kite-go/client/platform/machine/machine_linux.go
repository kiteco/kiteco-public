package machine

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// IDIfSet checks and returns a machineid if its been set. It will not attempt to generate one.
func IDIfSet() (string, bool) {
	// the machine ID is stored separate from settings.json because we want it to
	// persist across installs.
	path := os.ExpandEnv("$HOME/.kite/machine")
	buf, err := ioutil.ReadFile(path)
	if err == nil {
		return strings.TrimSpace(string(buf)), true
	}

	return "", false
}

// ID returns the ID of the current machine
func ID(devmode bool) (string, error) {
	// the machine ID is stored separate from settings.json because we want it to
	// persist across installs.
	path := os.ExpandEnv("$HOME/.kite/machine")

	buf, err := ioutil.ReadFile(path)
	if err == nil {
		return strings.TrimSpace(string(buf)), nil
	}

	if !os.IsNotExist(err) {
		log.Println("error loading machine ID:", err)
	}

	// could not load from file so generate and save
	mid := generateMachineID()

	err = os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		log.Println("error creating dir for machine ID:", err)
	}

	err = ioutil.WriteFile(path, []byte(mid), 0777)
	if err != nil {
		log.Println("error saving machine ID:", err)
	}

	return mid, nil
}
