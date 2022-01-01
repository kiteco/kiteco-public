// +build !windows

package installid

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var kiteRoot = os.ExpandEnv("$HOME/.kite")

// IDIfSet checks and returns a machineid if its been set. It will not attempt to generate one.
func IDIfSet() (string, bool) {
	// the machine ID is stored separate from settings.json because we want it to
	// persist across installs.
	path := filepath.Join(kiteRoot, "installid")
	buf, err := ioutil.ReadFile(path)
	if err == nil {
		return strings.TrimSpace(string(buf)), true
	}

	return "", false
}
