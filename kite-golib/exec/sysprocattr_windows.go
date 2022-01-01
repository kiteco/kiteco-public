// +build windows

package exec

import (
	"syscall"

	"github.com/winlabs/gowin32/wrappers"
)

var sysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: wrappers.CREATE_NO_WINDOW}
