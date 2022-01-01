package nsbundle

/*
#cgo CFLAGS: -mmacosx-version-min=10.11
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#cgo LDFLAGS: -framework Foundation
#cgo LDFLAGS: -mmacosx-version-min=10.11
#include <stdlib.h>
#include "nsbundle_darwin.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// AppVersion returns the version string of a bundle.
// Expects appPath to be the absolute path to the `.app` folder.
// This returns the empty string if the path is invalid or the version could not be deduced.
func AppVersion(appPath string) (string, error) {
	var err *C.char
	defer func() {
		cleanup(err)
	}()
	cString := C.CString(appPath)
	v := C.GoString(C.getVersion(cString, &err))
	return v, toError(err)
}

// AppRunning returns whether the app with the bundle ID is running.
func AppRunning(bundleID string) (bool, error) {
	var err *C.char
	defer func() {
		cleanup(err)
	}()
	arg := C.CString(bundleID)
	b := int(C.appRunning(arg, &err))
	return b != 0, toError(err)
}

// cleanup frees memory for an error if it is non-nil
func cleanup(err *C.char) {
	if err != nil {
		C.free(unsafe.Pointer(err))
	}
}

func toError(err *C.char) error {
	if err != nil {
		return fmt.Errorf("caught Objective-C exception in NSBundle: %s", C.GoString(err))
	}
	return nil
}
