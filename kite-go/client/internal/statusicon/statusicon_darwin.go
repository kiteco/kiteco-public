// +build !standalone

package statusicon

/*
#cgo CFLAGS: -x objective-c
#cgo CFLAGS: -mmacosx-version-min=10.11
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#cgo LDFLAGS: -framework Foundation
#cgo LDFLAGS: -mmacosx-version-min=10.11

#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

void terminate() {
	[NSApp terminate:nil];
}
*/
import "C"

import (
	"github.com/kiteco/kiteco/kite-golib/systray"
	"github.com/skratchdot/open-golang/open"
)

func (ui *UI) onBeforeRun() {
}

func (ui *UI) onHandleReceived(h systray.Handle) {
}

func (ui *UI) onSettingsClicked() {
	open.Run("kite://settings")
}

func (ui *UI) onSignedInAsClicked() {
	if _, err := ui.auth.GetUser(); err == nil {
		open.Run("kite://settings")
	} else {
		open.Run("kite://login")
	}
}

func terminate() {
	// Use cgo to terminate via NSApp so proper shutdown occurs. This ensures that
	// applicationWillTerminate is called on the application delegate. This contains
	// logic for applying pending updates.
	C.terminate()
}
