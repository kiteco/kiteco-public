package atom

import (
	"fmt"
	"log"
	"regexp"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

const (
	atomID       = "atom"
	atomName     = "Atom"
	updateArg    = "upgrade"
	apmPluginID  = "kite"
	noConfirmArg = "--no-confirm"
)

var (
	atomVersionMatcher        = regexp.MustCompile(`(?m)^Atom\s*:\s*(\S+)`)
	apmIsKiteInstalledMatcher = regexp.MustCompile(`(?m)^kite@`)

	errApmNotFound = errors.New("'apm' was not found on the system path")
)

// atomManager is used to share code between the different implementations of the Atom plugin manager
type atomManager interface {
	run(path string, args ...string) ([]byte, error)
	apmPath(editorPath string) string
	atomPath(editorPath string) string
}

func isInstalled(mgr atomManager, editorPath string) bool {
	out, err := mgr.run(mgr.apmPath(editorPath), "list", "--installed", "--bare", "--packages")
	if err != nil {
		log.Println(err)
		return false
	}
	return apmIsKiteInstalledMatcher.Match(out)
}

func install(mgr atomManager, editorPath string) error {
	_, err := mgr.run(mgr.apmPath(editorPath), "install", apmPluginID)
	return err
}

func uninstall(mgr atomManager, editorPath string) error {
	_, err := mgr.run(mgr.apmPath(editorPath), "uninstall", apmPluginID)
	return err
}

func update(mgr atomManager, editorPath string) error {
	_, err := mgr.run(mgr.apmPath(editorPath), updateArg, apmPluginID, noConfirmArg)
	return err
}

func openFile(mgr atomManager, editorPath string, filePath string, line int) (<-chan error, error) {
	if editorPath == "" {
		return nil, errors.New("empty editor path")
	}
	dest := fmt.Sprintf("%s:%d", filePath, line)
	_, err := mgr.run(mgr.atomPath(editorPath), dest)
	return nil, err
}

func readAtomVersion(stdout []byte) (string, error) {
	matches := atomVersionMatcher.FindSubmatch(stdout)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not read version from %q", stdout)
	}
	return string(matches[1]), nil
}
