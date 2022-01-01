package process

import (
	"log"
	"path/filepath"
	"syscall"

	"github.com/winlabs/gowin32"
	"github.com/winlabs/gowin32/wrappers"
)

var attributes = &syscall.SysProcAttr{HideWindow: true, CreationFlags: wrappers.CREATE_NO_WINDOW}

// Name of Kite process
var Name = "kited.exe"

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
	programFiles, err := gowin32.GetKnownFolderPath(gowin32.KnownFolderProgramFiles)
	if err != nil {
		log.Println("error retrieving programFiles path", err)
		return "", err
	}
	// C:\Program Files\Kite
	p := filepath.Join(programFiles, "Kite", "kited")
	return p, nil
}
