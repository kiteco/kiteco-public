package sandbox

import (
	"fmt"
	"os/exec"
	"time"
)

// UncleanExit is generated when a subprocess returns non-zero exit code.
type UncleanExit struct {
	// ExecError is the underlying exiterror received from exec.Wait
	ExitError *exec.ExitError
	// Stderr is the contents of the standard error stream at exit
	Stderr string
}

// Error returns a description of what went wrong.
func (e *UncleanExit) Error() string {
	return fmt.Sprintf("%v. output was: %s", e.ExitError, e.Stderr)
}

// TimeLimitExceeded is generated when a subprocess returns non-zero exit code.
type TimeLimitExceeded struct {
	// Limit is the timeout (which was exceeded)
	Limit time.Duration
}

// Error returns a description of what went wrong.
func (e *TimeLimitExceeded) Error() string {
	return fmt.Sprintf("time limit exceeded")
}

// OutputLimitExceeded is generated when a subprocess writes too much output to a stream
type OutputLimitExceeded struct {
	// Stream indicates which stream exceeded the output limit, "stdout" or "stderr"
	Stream string
	// Limit is the maximum output length (which was exceeded)
	Limit int
}

// Error returns a description of what went wrong.
func (e *OutputLimitExceeded) Error() string {
	return fmt.Sprintf("%s limit exceeded after %d lines", e.Stream, e.Limit)
}
