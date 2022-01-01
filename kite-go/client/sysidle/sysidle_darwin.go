package sysidle

/*
#cgo CFLAGS: -mmacosx-version-min=10.11
#cgo LDFLAGS: -framework Cocoa
#cgo LDFLAGS: -framework Foundation
#cgo LDFLAGS: -mmacosx-version-min=10.11

#include <stdlib.h>
#include "sysidle_darwin.h"
*/
import "C"

// sysIdle returns if the system is idle.
func sysIdle() bool {
	return bool(C.systemIdle())
}
