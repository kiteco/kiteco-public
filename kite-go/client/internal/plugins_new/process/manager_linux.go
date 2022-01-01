package process

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/process"
)

var (
	commonPaths = []string{
		// don't include /sbin/ and others because that would be a weird place to put an editor
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
	}
	attributes = &syscall.SysProcAttr{}
)

// LinuxProcessManager handles process information on Linux
type LinuxProcessManager interface {
	IsProcessRunning(ctx context.Context, name string) (bool, error)
	Run(name string, arg ...string) ([]byte, error)
	FindBinary(name string) []string
	Home() (string, error)

	List(ctx context.Context) (List, error)
}

// NewManager returns a new LinuxProcessManager
func NewManager() LinuxProcessManager {
	return &processManager{}
}

type processManager struct{}

// IsProcessRunning implements LinuxProcessManager
// It supports simple names (e.g. idea.sh) and absolute paths to the executable file (e.g. /opt/intellij/bin/idea.sh)
// If a relative or absolute path is passed then only the name of the executable will be checked (i.e. idea.sh)
// If an absolute path is passed then the commandline must also contain this path (i.e. /opt/intellij/bin/idea.sh)
// Testing the prefix is not an option because a commandline may be prefixed with /bin/sh
// The value returned by process.Name() is always the name of the executable without the path, e.g. "idea.sh"
// If exePath is a path (e.g. /opt/intellij/bin/idea.sh) then the command line must contain the path we're looking for
func (m *processManager) IsProcessRunning(ctx context.Context, exePath string) (bool, error) {
	list, err := process.Processes()
	if err != nil {
		log.Printf("error retrieving process list: %s", err.Error())
		return false, err
	}

	isAbsPath := filepath.IsAbs(exePath)
	baseName := filepath.Base(exePath)

	for _, p := range list {
		curName, err := p.NameWithContext(ctx)
		if err != nil {
			log.Printf("error retrieving process name: %s", err.Error())
			continue
		}

		if isAbsPath {
			if cmdline, err := p.CmdlineWithContext(ctx); err != nil {
				log.Printf("error retrieving cmdline: %s", err.Error())
				continue
			} else if !strings.Contains(cmdline, exePath) {
				continue
			}
		}

		if curName == baseName {
			return true, nil
		}
	}
	return false, nil
}

// Run executes a command in a subprocess and returns its output:
//  - command is logged always
//  - stdout and stderr are logged on error
//  - returns stdout only
//  - returns a ProcessError which wraps stdout and stderr when the command failed
func (m *processManager) Run(name string, arg ...string) ([]byte, error) {
	return runProcess(name, nil, arg...)
}

// Finds any executables that lives in commonPaths, e.g. Vim, Neovim.
func (m *processManager) FindBinary(name string) []string {
	return findBinary(name)
}

// Home returns the current user's home directory, or an error if it can't be found.
func (m *processManager) Home() (string, error) {
	return homeDir()
}

// List returns a list of currently running processes
func (m *processManager) List(ctx context.Context) (List, error) {
	return list(ctx)
}
