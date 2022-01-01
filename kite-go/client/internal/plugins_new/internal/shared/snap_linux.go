package shared

import (
	"path/filepath"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

var snapDir = "/snap"

// SnapPath checks if path is a snap path.
// If so, it converts it to /snap/{name}/current/{loc}
// so the plugin manager can use the actual binary.
func SnapPath(path string, name string, loc string) (string, error) {
	// Check if path starts with /{snapDir}/bin
	if filepath.Dir(path) != filepath.Join(snapDir, "bin") {
		return path, nil
	}
	sp := filepath.Join(snapDir, name, "current", loc)
	if !fs.FileExists(sp) {
		return "", errors.Errorf("Converted Snap path {%s} does not exist", sp)
	}
	return sp, nil
}
