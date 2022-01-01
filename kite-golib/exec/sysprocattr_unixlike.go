// +build !windows

package exec

import "syscall"

var sysProcAttr = &syscall.SysProcAttr{}
