// +build !standalone

package liveupdates

/*
#cgo CFLAGS: -framework Sparkle
#cgo CFLAGS: -F${SRCDIR}/../../../../../osx
#cgo CFLAGS: -mmacosx-version-min=10.11
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#cgo LDFLAGS: -framework Foundation
#cgo LDFLAGS: -framework Sparkle
#cgo LDFLAGS: -framework QuartzCore
#cgo LDFLAGS: -F${SRCDIR}/../../../../../osx
#cgo LDFLAGS: -Wl,-rpath,@executable_path/../Frameworks
#cgo LDFLAGS: -Wl,-rpath,@loader_path/../Frameworks
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/../../../../../osx
#cgo LDFLAGS: -mmacosx-version-min=10.11

#include <stdlib.h>
#include "updater_darwin.h"
*/
import "C"
import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	"unsafe"

	"github.com/kiteco/kiteco/kite-go/client/platform/machine"
)

// Listener is the callback for when Sparkle has fetched an update and will
// install it on quit. It corresponds to SUUpdaterDelegate willInstallUpdateOnQuit
type Listener func()

var listener Listener

// cleanup frees memory for an error if it is non-nil
func cleanup(err *C.char) {
	if err != nil {
		C.free(unsafe.Pointer(err))
	}
}

//export cgoWillInstallUpdateOnQuit
func cgoWillInstallUpdateOnQuit() {
	if listener == nil {
		log.Println("cgoWillInstallUpdateOnQuit called but listener is nil")
		return
	}
	listener()
}

//export cgoGetMachineID
func cgoGetMachineID() *C.char {
	mID, err := machine.ID(false)
	if err != nil {
		mID = ""
	}
	return C.CString(mID)
}

func checkForUpdates(showModal bool) error {
	var err *C.char
	defer func() {
		cleanup(err)
	}()
	C.checkForUpdates(C.bool(showModal), &err)
	if err != nil {
		return fmt.Errorf("caught objective-c exception in checkForUpdates: %s", C.GoString(err))
	}
	return nil
}

func restartAndUpdate() error {
	var err *C.char
	defer func() {
		cleanup(err)
	}()
	C.restartAndUpdate(&err)
	if err != nil {
		return fmt.Errorf("caught objective-c exception in restartAndUpdate: %s", C.GoString(err))
	}
	return nil
}

func restart() error {
	// not supported on macOS
	return nil
}

func updateReady() bool {
	return bool(C.updateReady())
}

func secondsSinceUpdateReady() int {
	return int(C.secondsSinceUpdateReady())
}

// start starts an updater for the bundle at the given path. The update settings
// will be read from that bundle Info.plist (including the remove address from
// which to fetch updates). The listener will be called when an update is ready.
func start(ctx context.Context, bundle string, f Listener, lastEvent func() time.Time) error {
	var err *C.char
	defer func() {
		cleanup(err)
	}()
	listener = f
	bundlestr := C.CString(bundle)
	defer C.free(unsafe.Pointer(bundlestr))
	C.start(bundlestr, &err)
	if err != nil {
		return fmt.Errorf("caught objective-c exception in start update loop: %s", C.GoString(err))
	}
	return nil
}

// UpdateTarget returns the update target to use for the live updater
func UpdateTarget() (string, error) {
	target := os.Getenv("KITE_UPDATE_TARGET")
	if target == "" {
		return "", fmt.Errorf("kiteInitialize: need to set KITE_UPDATE_TARGET")
	}
	return target, nil
}
