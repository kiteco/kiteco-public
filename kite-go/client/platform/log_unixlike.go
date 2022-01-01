// +build !windows

package platform

import (
	"io"
	"os"
)

// logWriter returns a writer that recieves log output.
func logWriter(logfile string, enableStdout bool) (io.Writer, error) {
	f, err := os.Create(logfile)
	if err != nil {
		return nil, err
	}

	if enableStdout {
		return io.MultiWriter(f, os.Stdout), nil
	}

	return f, nil
}
