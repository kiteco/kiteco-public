package process

import (
	"path/filepath"
	"syscall"
)

var attributes = &syscall.SysProcAttr{}

// Name of Kite process
var Name = "kited"

// Start attempts to start Kite.
func Start() error {
	path, err := installPath()
	if err != nil {
		return err
	}
	_, err = startProcess(path, nil, "--plugin-launch")
	if err != nil {
		return err
	}
	return nil
}

// Use default location defined in https://help.kite.com/article/136-how-to-restart-kite
func installPath() (string, error) {
	hd, err := homeDir()
	if err != nil {
		return "", err
	}
	// ~/.local/share/kite/kited
	p := filepath.Join(hd, ".local", "share", "kite", "kited")
	return p, nil
}
