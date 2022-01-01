package visibility

// #cgo LDFLAGS: -lPsapi
// #include "visibility_windows.h"
import "C"
import "unsafe"

var sidebarExe = "Kite.exe"

// windowVisible returns whether the center of any window belonging to
// the given executable is currently visible.
func windowVisible() bool {
	str := C.CString(sidebarExe)
	defer C.free(unsafe.Pointer(str))
	return bool(C.windowVisible(str))
}
