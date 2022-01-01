package sidebar

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"unsafe"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/reg"
)

// #cgo LDFLAGS: -lPsapi
// #include "sidebar_windows.h"
import "C"

const sidebarExe = "Kite.exe"

var copilotDevDir string

func newController(settings component.SettingsManager) windowsController {
	return windowsController{}
}

type windowsController struct{}

// Start implements Controller
func (d windowsController) Start() error {
	running, runningErr := d.Running()
	if runningErr != nil {
		return fmt.Errorf("unable to check running state: %v", runningErr)
	}
	if running {
		return d.Focus()
	}

	return exec.Command(d.cmdPath()).Start()
}

// Focus implements Controller
func (d windowsController) Focus() error {
	cstr := C.CString(sidebarExe)
	defer C.free(unsafe.Pointer(cstr))
	C.focus(cstr)
	return nil
}

// Stop implements Controller
func (d windowsController) Stop() error {
	cstr := C.CString(sidebarExe)
	defer C.free(unsafe.Pointer(cstr))
	C.killIfRunning(cstr)
	return nil
}

// Running implements Controller
func (d windowsController) Running() (bool, error) {
	cstr := C.CString(sidebarExe)
	defer C.free(unsafe.Pointer(cstr))
	ret := C.isRunning(cstr)
	return bool(ret), nil
}

// SetWasVisible implements Controller
func (d windowsController) SetWasVisible(val bool) error {
	return reg.SetWasVisible(val)
}

// WasVisible implements Controller
func (d windowsController) WasVisible() (bool, error) {
	return reg.WasVisible()
}

// Notify implements Controller
func (d windowsController) Notify(id string) error {
	cmd := exec.Command(d.cmdPath(), "--notification="+id)
	err := cmd.Start()
	go cmd.Wait() // cleanup resources when the process exits
	return err
}

func (d windowsController) cmdPath() string {
	installdir, err := reg.InstallPath()
	if err != nil {
		installdir = `C:\Program Files\Kite`
		log.Println("unable to get installdir from registry, err:", err)
	}
	if copilotDevDir != "" {
		// Set when running kited.exe on windows for development
		installdir = copilotDevDir
	}
	log.Println("installdir:", installdir)

	return filepath.Join(installdir, "win-unpacked", sidebarExe)
}
