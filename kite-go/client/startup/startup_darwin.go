// +build darwin

package startup

/*
#cgo CFLAGS: -x objective-c
#cgo CFLAGS: -mmacosx-version-min=10.11
#cgo LDFLAGS: -framework Cocoa
#cgo LDFLAGS: -framework Foundation
#cgo LDFLAGS: -mmacosx-version-min=10.11

#include <stdlib.h>
#include "startup_darwin.h"
*/
import "C"

func init() {
	C.init()
}

func mode() Mode {
	if wasManuallyLaunched() {
		return ManualLaunch
	}

	// Backwards compatibility: Check if shouldReopenSidebar has been set to NO (defaults to Yes, see startup_darwin.m).
	// If so, we are in plugin launch mode (plugins use this variable to disable sidebar launching)
	if !bool(C.shouldReopenSidebar()) {
		return PluginLaunch
	}

	// NOTE: KiteHelper.app uses the --system-boot flag, and a restart via the Sidebar always uses
	// the --sidebar-restart flag, so if those did not already get caught
	// via sidebar.go, and the application was not manually launched, then this must be the result
	// of relaunch after update (right?). Unfortunately I don't know of a way to make sparkle pass in
	// an argument when relaunching the application
	return RelaunchAfterUpdate
}

func reset() {
	// We need to set this to true immediately after detecting mode. The goal is to detect
	// when this is set to false, so we want to avoid leaving it at false after reading from it.
	C.setShouldReopenSidebar(true)
}

// --

func wasManuallyLaunched() bool {
	return bool(C.wasManuallyLaunched())
}
