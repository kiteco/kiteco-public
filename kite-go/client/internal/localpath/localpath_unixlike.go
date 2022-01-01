// +build !windows

package localpath

import (
	"os"
	"strings"
)

func toUnix(path string) (string, error) {
	return path, nil
}

func fromUnix(path string) (string, error) {
	return path, nil
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~") {
		return os.ExpandEnv("$HOME") + strings.TrimPrefix(path, "~")
	}
	return path
}

func isRootDir(path string) bool {
	return path == "/"
}
