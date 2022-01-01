package exec

import "os/exec"

// Cmd is os/exec.Cmd
type Cmd = exec.Cmd

// LookPath is os/exec.LookPath
var LookPath = exec.LookPath

// Command is os/exec.Command, but prevents Windows from opening a Window
func Command(name string, arg ...string) *Cmd {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = sysProcAttr
	return cmd
}
