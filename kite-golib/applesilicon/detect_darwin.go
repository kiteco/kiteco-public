// +build darwin

package applesilicon

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
)

// Detected is set to true if process is running via Rosetta 2 on Apple Silicon
var Detected bool

// Based on https://steipete.com/posts/apple-silicon-mac-mini-for-ci/#detecting-apple-silicon-via-scripts
// and https://www.yellowduck.be/posts/detecting-apple-silicon-via-go/
func init() {
	r, err := syscall.Sysctl("sysctl.proc_translated")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("running on intel mac, arch:", runtime.GOARCH)
		} else {
			fmt.Println("unknown error when detecting architecture:", err)
		}
		return
	}

	switch r {
	case "\x00\x00\x00":
		// This should not be possible, as we aren't building for arm64
		fmt.Println("running on apple silicon natively, arch:", runtime.GOARCH)
	case "\x01\x00\x00":
		fmt.Println("running on apple silicon under rosetta 2, arch:", runtime.GOARCH)
		Detected = true
	}
}
