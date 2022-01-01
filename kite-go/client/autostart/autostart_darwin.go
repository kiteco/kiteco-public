// +build !standalone

package autostart

/*
#cgo CFLAGS: -x objective-c
#cgo CFLAGS: -mmacosx-version-min=10.11
#cgo LDFLAGS: -framework Cocoa
#cgo LDFLAGS: -framework Foundation
#cgo LDFLAGS: -framework ServiceManagement
#cgo LDFLAGS: -mmacosx-version-min=10.11

#include <stdlib.h>
#include "autostart_darwin.h"
*/
import "C"

func setEnabled(enabled bool) error {
	C.setEnabled(C._Bool(enabled))
	return nil
}
