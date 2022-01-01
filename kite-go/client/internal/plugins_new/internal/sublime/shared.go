package sublime

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

const (
	id            = "sublime3"
	name          = "Sublime Text 3"
	pluginDirName = "KiteSublime"
)

// sublimeManager is used to share code between the different implementations of the Sublime plugin manager
type sublimeManager interface {
	runSublime(editorPath string, args ...string) ([]byte, error)
	cliPath(editorPath string) string
}

func editorConfig(editorPath string, build int) (system.Editor, error) {
	version := fmt.Sprintf("Build %d", build)
	compatibility := ""
	requiredVersion := ""
	if build == 0 {
		// In windows, build == 0 if we fail to parse the version.
		// But we sometimes get the *file version* (1.0.0.1) instead of product version,
		// so consider this case compatible for now.
		// TODO(naman) see #8931
	} else if build < 3000 {
		requiredVersion = "3000"
		compatibility = fmt.Sprintf("build must be 3000 or higher (found %d)", build)
	} else if build >= 4000 {
		requiredVersion = "3000"
		compatibility = fmt.Sprintf("only builds < 4000 are supported (found %d)", build)
	}

	return system.Editor{
		Path:            editorPath,
		Version:         version,
		Compatibility:   compatibility,
		RequiredVersion: requiredVersion,
	}, nil
}

func openFile(mgr sublimeManager, editorPath string, filePath string, line int) (<-chan error, error) {
	if editorPath == "" {
		return nil, errors.New("empty editor path")
	}
	dest := fmt.Sprintf("%s:%d", filePath, line)
	_, err := mgr.runSublime(editorPath, dest)
	return nil, err
}
