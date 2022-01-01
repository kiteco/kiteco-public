package process

import (
	"errors"
	"os/exec"
	"strings"
	"syscall"
)

var attributes = &syscall.SysProcAttr{}
var bundleID = "com.kite.Kite"

// Name of Kite process
var Name = "Kite"

// Start attempts to start Kite.
func Start() error {
	loc, err := bundleLocation()
	if err != nil {
		return err
	}

	_, err = startProcess("open", nil, "-a", loc, "--args", "--plugin-launch")
	return err
}

func bundleLocation() (string, error) {
	out, err := exec.Command("mdfind", "kMDItemCFBundleIdentifier", "=", bundleID).Output()
	if err != nil {
		return "", err
	}

	var valid []string
	for _, x := range strings.Split(string(out), "\n") {
		if x != "" {
			valid = append(valid, x)
		}
	}

	if len(valid) < 1 {
		return "", errors.New("Couldn't find bundle location")
	}
	return valid[0], nil
}
