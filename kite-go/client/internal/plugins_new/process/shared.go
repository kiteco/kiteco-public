package process

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
)

// ErrProcessRunning is used when a process is running but wasn't expected to
var ErrProcessRunning = errors.New("a process is running")

// Error implements error. It wraps an error message, stdout, and stderr of the process which failed to execute
type Error struct {
	msg    string
	stderr string
	stdout string
}

// Error returns the error message intended for users
func (e Error) Error() string {
	return e.msg
}

// Stdout returns the content which was printed to stdout by the failed process
func (e Error) Stdout() string {
	return e.stdout
}

// Stderr returns the content which was printed to stderr by the failed process
func (e Error) Stderr() string {
	return e.stderr
}

// Run executes a command in a subprocess and returns its output:
//  - command is logged always
//  - stdout and stderr are logged on error
//  - returns stdout only
//  - returns a ProcessError which wraps stdout and stderr when the command failed
func runProcess(name string, additionalEnv []string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = attributes
	if additionalEnv != nil {
		cmd.Env = append(os.Environ(), additionalEnv...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, Error{
			msg:    fmt.Sprintf("error running %s %q: %v", name, strings.Join(arg, " "), err),
			stdout: stdout.String(),
			stderr: stderr.String(),
		}
	}
	return stdout.Bytes(), nil
}

// Finds any executables that lives in commonPaths, e.g. Vim.
func findBinary(name string) []string {
	var paths []string
	execPath, err := exec.LookPath(name)
	if err == nil {
		paths = append(paths, execPath)
	}
	for _, base := range commonPaths {
		p := filepath.Join(base, name)
		// don't duplicate path if LookPath already found it
		if fs.FileExists(p) && p != execPath {
			paths = append(paths, p)
		}
	}
	return paths
}

// Returns the current user's home directory, or an error if it can't be found.
func homeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.HomeDir, nil
}
