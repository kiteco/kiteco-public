package permissions

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"golang.org/x/text/unicode/norm"
)

// sanitizePaths calls sanitizePaths on each element of paths.
//
// returns error if after the above operations a path element is not absolute
// or if there was a problem sanitizing the paths
// NOTE: if no error is returned then the return slice has len == len(paths)
func sanitizePaths(paths ...string) ([]string, error) {
	var result []string
	for _, path := range paths {
		sanitized, err := sanitizePath(path)
		if err != nil {
			return nil, fmt.Errorf("error sanitizing path %s: %v", path, err)
		}

		result = append(result, sanitized)
	}
	return result, nil
}

// sanitizePath performs the following operations on path:
//
// - convert "colonated" paths to native paths
//
// - expand prefix tilde
//
// - convert slashed paths `/` to native PathSeparator paths
//
// - windows paths are lowercased
//
// returns error if after the above operations the path is not absolute or if there was a problem sanitizing the paths
//
// NOTE: The original path will be returned when an error ocurred
func sanitizePath(path string) (string, error) {
	// normalize unicode
	var f norm.Form
	sanitized := f.String(path)

	if strings.HasPrefix(path, ":") {
		var err error
		sanitized, err = localpath.ColonToNative(path)
		if err != nil {
			return path, fmt.Errorf("error sanitizing path %s: %v", path, err)
		}
	}

	if runtime.GOOS == "windows" {
		sanitized = strings.ToLower(sanitized)
	}
	sanitized = localpath.ExpandTilde(sanitized)
	sanitized = filepath.FromSlash(sanitized)
	if !filepath.IsAbs(sanitized) {
		return path, fmt.Errorf("non absolute path %s", path)
	}

	return sanitized, nil
}
