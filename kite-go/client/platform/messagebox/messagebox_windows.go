// +build !standalone

package messagebox

import (
	"log"
	"syscall"
	"unsafe"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	procMessageBox = user32.NewProc("MessageBoxW")
)

func showAlert(opts Options) error {
	procMessageBox.Call(
		uintptr(0),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(opts.Text))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(opts.Title))),
		uintptr(0x30))
	return nil
}

func showWarning(opts Options) error {
	log.Println(opts.Text)
	return nil
}
