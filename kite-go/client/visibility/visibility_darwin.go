package visibility

/*
#cgo CFLAGS: -mmacosx-version-min=10.11
#cgo LDFLAGS: -framework Cocoa
#cgo LDFLAGS: -framework Foundation
#cgo LDFLAGS: -mmacosx-version-min=10.11
#include <stdlib.h>
#include "visibility_darwin.h"
*/
import "C"
import "unsafe"

var sidebarApp = "Kite"

// windowVisible returns whether the center of the first window belonging to
// the given app is currently visible.
func windowVisible() bool {
	str := C.CString(sidebarApp)
	defer C.free(unsafe.Pointer(str))
	return bool(C.windowVisible(str))
}
