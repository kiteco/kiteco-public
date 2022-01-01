package localpath

import (
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// ToUnix converts an absolute filesystem path to a unix-like path. On unix-
// like systems this function returns its input. It panics if the path is non-
// absolute.
func ToUnix(path string) (string, error) {
	if path == "" { // pass through empty paths because unfortunately these are widely used
		return "", nil
	}
	// Check if this is already in Windows->Posix format, and return early
	if strings.HasPrefix(path, "/windows/") {
		return path, nil
	}
	if !filepath.IsAbs(path) {
		return "", errors.Errorf("localpath.ToUnix received non-absolute path: %q", path)
	}
	return toUnix(path)
}

// FromUnix converts a unix-like path to a path that can be used on the local
// filesystem.
func FromUnix(path string) (string, error) {
	return fromUnix(path)
}

// ExpandTilde expands paths starting with ~ using the value of the $HOME variable,
// for windows paths this returns path unchanged.
func ExpandTilde(path string) string {
	return expandTilde(path)
}

// ColonToNative converts a colon separated path to a unix-like path.
// e.g :windows:C:Users:account1:Documents:foo.py becomes C:\Users\account1\Documents\foo.py
// and :Users:alex:test.py becomes /Users/alex/test.py
func ColonToNative(path string) (string, error) {
	path = strings.Replace(path, ":", "/", -1)
	return fromUnix(path)
}

// IsRootDir returns whether or not an input path is a root directory
func IsRootDir(path string) bool {
	return isRootDir(path)
}
