// +build windows

package filesystem

import (
	"syscall"

	"github.com/winlabs/gowin32/wrappers"
)

var attributes = &syscall.SysProcAttr{HideWindow: true, CreationFlags: wrappers.CREATE_NO_WINDOW}
