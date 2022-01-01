// +build !windows

package filesystem

import "syscall"

var attributes = &syscall.SysProcAttr{}
