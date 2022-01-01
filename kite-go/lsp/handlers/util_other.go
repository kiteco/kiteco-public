// +build !windows

package handlers

import (
	"errors"
	"path/filepath"
)

// https://golang.org/src/cmd/go/internal/web/url_other.go
func convertFileURLPath(host, path string) (string, error) {
	switch host {
	case "", "localhost":
	default:
		return "", errors.New("file URL specifies non-local host")
	}
	return filepath.FromSlash(path), nil
}
