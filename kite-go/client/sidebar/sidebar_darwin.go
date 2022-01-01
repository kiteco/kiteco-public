// +build !standalone

package sidebar

/*
#cgo CFLAGS: -x objective-c
#cgo CFLAGS: -mmacosx-version-min=10.11
#cgo LDFLAGS: -framework Cocoa
#cgo LDFLAGS: -framework Foundation
#cgo LDFLAGS: -mmacosx-version-min=10.11

#include <stdlib.h>
#include "sidebar_darwin.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"unsafe"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

func newController(settings component.SettingsManager) darwinController {
	// start observer of the activation notification. this notification lets us know
	// when Kite.app has been activated (e.g via spotlight, or Finder) via a re-launch.
	// see cgoOnAppActivated for logic that is invoked when the app is activated
	if err := startObserver(); err != nil {
		errMsg := errors.New("unable to start sidebar observer: sidebar activation may not work")
		log.Println(errMsg, err)
		rollbar.Error(errMsg, err)
	}

	return darwinController{}
}

type darwinController struct{}

// Start implements Controller
func (d darwinController) Start() error {
	running, runningErr := d.Running()
	if runningErr != nil {
		return fmt.Errorf("unable to check running state: %v", runningErr)
	}
	if running {
		return d.Focus()
	}

	var err *C.char
	defer func() { cleanup(err) }()
	C.launch(&err)
	if err != nil {
		return fmt.Errorf("caught objective-c exception in launch: %s", C.GoString(err))
	}
	return nil
}

// Stop implements Controller
func (d darwinController) Stop() error {
	var err *C.char
	defer func() { cleanup(err) }()
	C.quitSidebar(&err)
	if err != nil {
		return fmt.Errorf("caught objective-c exception in quitSidebar: %s", C.GoString(err))
	}
	return nil
}

// Running implements Controller
func (d darwinController) Running() (bool, error) {
	var err *C.char
	defer func() { cleanup(err) }()
	v := bool(C.isRunning(&err))
	if err != nil {
		return v, fmt.Errorf("caught objective-c exception in isRunning: %s", C.GoString(err))
	}
	return v, nil
}

// Focus shows the sidebar window if it is hidden and brings it to the front.
func (d darwinController) Focus() error {
	var err *C.char
	defer func() { cleanup(err) }()
	C.focus(&err)
	if err != nil {
		return fmt.Errorf("caught objective-c exception in focus: %s", C.GoString(err))
	}
	return nil
}

// SetWasVisible implements Controller
func (d darwinController) SetWasVisible(val bool) error {
	C.setWasVisible(C._Bool(val))
	return nil
}

// WasVisible implements Controller
func (d darwinController) WasVisible() (bool, error) {
	return bool(C.wasVisible()), nil
}

// Notify implements Controller
func (d darwinController) Notify(id string) error {
	var path *C.char = C.appPath()
	defer C.free(unsafe.Pointer(path))

	exe := filepath.Join(C.GoString(path), "Contents/MacOS/Kite")
	cmd := exec.Command(exe, "--notification="+id)

	err := cmd.Start()
	go cmd.Wait() // cleanup resources when the process exits
	return err
}

// --

// cleanup frees memory for an error if it is non-nil
func cleanup(err *C.char) {
	if err != nil {
		C.free(unsafe.Pointer(err))
	}
}

func startObserver() error {
	var err *C.char
	defer func() {
		cleanup(err)
	}()
	C.startObserver(&err)
	if err != nil {
		return fmt.Errorf("caught objective-c exception in startObserver: %s", C.GoString(err))
	}
	return nil
}

//export cgoOnAppActivated
func cgoOnAppActivated() {
	log.Println("sidebar: application activated")
	// Start the sidebar when app is activated. On OS X, the app can become "activated"
	// when launched via Spotlight/Finder while already running.
	Start()
}
