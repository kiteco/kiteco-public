package localpath

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

var (
	driveRegexp = regexp.MustCompile(`[a-zA-Z]\:\\`)
)

func toUnix(path string) (string, error) {
	if strings.HasPrefix(path, `\\`) { // UNC path like \\Bob\Share\Documents
		return "/windows/unc/" + strings.Replace(path[2:], `\`, `/`, -1), nil
	}
	// Repeat this check, since this is called by isRootDir below
	if strings.HasPrefix(path, "/windows/") {
		return path, nil
	}
	volname := filepath.VolumeName(path)
	if len(volname) != 2 {
		return "", errors.Errorf("not a UNC path but volume was %q: %s", volname, path)
	}
	return fmt.Sprintf("/windows/%s%s", volname[:1], filepath.ToSlash(path[2:])), nil
}

func fromUnix(unix string) (string, error) {
	if !strings.HasPrefix(unix, "/windows/") {
		// Check if this is already in windows native format, and return early
		matches := driveRegexp.FindAllStringIndex(unix, -1)
		if matches != nil && matches[0][0] == 0 {
			return unix, nil
		}
		return "", errors.Errorf("expected path to begin '/windows/' but got %s", unix)
	}
	sub := strings.TrimPrefix(unix, "/windows/")
	pos := strings.Index(sub, "/")
	if pos == -1 {
		return "", errors.Errorf("path is missing volume: %s", unix)
	}

	vol := sub[:pos]
	winpath := strings.Replace(sub[pos:], `/`, `\`, -1)

	if vol == "unc" {
		// will result in a path like \\machine\share\path\to\src.py
		return `\` + winpath, nil
	}
	if len(vol) == 0 {
		return "", errors.Errorf("no volume")
	}
	// will result in a path like c:\path\to\src.py
	return fmt.Sprintf(`%s:%s`, vol, winpath), nil
}

func expandTilde(path string) string {
	return path
}

func isRootDir(path string) bool {
	unix, err := toUnix(path)
	if err != nil {
		return false
	}
	if match, _ := regexp.Match("^/windows/[^/]/?$", []byte(unix)); match {
		return match
	}
	match, _ := regexp.Match("^/windows/unc/[^/]+/[^/]+/?$", []byte(unix))
	return match
}
