package platform

import (
	"io"
	"log"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procOutputDebugString = kernel32.NewProc("OutputDebugStringW")
	procSetStdHandle      = kernel32.NewProc("SetStdHandle")
)

// debugStringOutputter is an io.Writer that writes to OutputDebugString
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa363362(v=vs.85).aspx
//
// this seems not to work when kited.exe is build with "-H windowsgui"
type debugStringOutputter struct{}

// Write implements io.Writer.Write
func (debugStringOutputter) Write(buf []byte) (int, error) {
	p := syscall.StringToUTF16Ptr(string(buf))
	procOutputDebugString.Call(uintptr(unsafe.Pointer(p)))
	return len(buf), nil
}

// logWriter returns a writer that recieves log output
func logWriter(logfile string, devmode bool) (io.Writer, error) {
	f, err := os.Create(logfile)
	if err != nil {
		return nil, err
	}

	if devmode {
		return io.MultiWriter(f, debugStringOutputter{}), nil
	}

	redirectStderr(f)

	return f, nil
}

// redirectStderr to the file passed in
//
// Copied from SO link: /questions/34772012/capturing-panic-in-golang
func redirectStderr(f *os.File) {
	err := setStdHandle(syscall.STD_ERROR_HANDLE, syscall.Handle(f.Fd()))
	if err != nil {
		log.Printf("Failed to redirect stderr to file: %v", err)
	}
	// Also assign os.Stderr since setStdHandle does not affect prior references to stderr
	os.Stderr = f
}

func setStdHandle(stdhandle int32, handle syscall.Handle) error {
	r0, _, e1 := syscall.Syscall(procSetStdHandle.Addr(), 2, uintptr(stdhandle), uintptr(handle), 0)
	if r0 == 0 {
		if e1 != 0 {
			return error(e1)
		}
		return syscall.EINVAL
	}
	return nil
}
