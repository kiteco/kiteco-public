package vscode

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"regexp"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

const (
	// https://marketplace.visualstudio.com/items?itemName=kiteco.kite
	vscodeMarketplaceID = "kiteco.kite"
	vscodeID            = "vscode"
	vscodeName          = "Visual Studio Code"
	installExtensionArg = "--install-extension"
	forceArg            = "--force"
	gotoArg             = "-g"
)

var (
	versionMatcher = regexp.MustCompile(`^(\d+.\d+)`)
)

// vscodeManager is used to share code between the different implementations of the VSCode plugin manager
type vscodeManager interface {
	runVSCode(editorPath string, args ...string) ([]byte, error)
	cliPath(editorPath string) string
	userExtensionsDir(editorPath string) string
}

func isInstalled(mgr vscodeManager, editorPath string) bool {
	cliResult, err := isInstalledCLI(mgr, editorPath)
	if err != nil {
		cliErr := fmt.Errorf("vscode plugin install detection via CLI failed")
		log.Printf("%s - %s", cliErr, err)
	}
	return cliResult
}

func isInstalledCLI(mgr vscodeManager, editorPath string) (bool, error) {
	stdout, err := mgr.runVSCode(editorPath, "--list-extensions")
	if err != nil {
		return false, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	for scanner.Scan() {
		if scanner.Text() == vscodeMarketplaceID {
			return true, nil
		}
	}
	return false, nil
}

func isInstalledExtensionsDir(mgr vscodeManager, editorPath string) (bool, error) {
	extDir := mgr.userExtensionsDir(editorPath)
	fis, err := ioutil.ReadDir(extDir)
	if err != nil {
		return false, fmt.Errorf("error reading vscode extensions directory %s: %s", extDir, err)
	}

	for _, fi := range fis {
		if strings.Contains(fi.Name(), "kiteco.kite") {
			return true, nil
		}
	}
	return false, nil
}

func install(mgr vscodeManager, editorPath string) error {
	_, err := mgr.runVSCode(editorPath, "--install-extension", vscodeMarketplaceID, forceArg)
	return err
}

func uninstall(mgr vscodeManager, editorPath string) error {
	// fixme is this still needed?
	_ = uninstallOldPlugin(mgr, editorPath)

	// NOTE: we don't use the CLI to uninstall the plugin b/c its whack. The CLI
	// erroneously reports that our plugin is not installed when the extension exists in the
	// extension directory and seems to be operational within vscode. to get around this, we
	// check the extensions directory to see if its actually installed. A side-effect
	// of this is that when the CLI erroneously thinks the plugin is not installed, it will
	// return an error during uninstalled (e.g plugin does not exist). So, we resort to manually
	// removing the plugin. The CLI *does* seem to do the right thing when manually removing the plugin

	path := mgr.userExtensionsDir(editorPath)
	fis, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("error reading vscode extensions directory %s: %s", path, err)
	}

	for _, fi := range fis {
		if strings.Contains(fi.Name(), "kiteco.kite") {
			err = os.RemoveAll(filepath.Join(path, fi.Name()))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// To update an extension via the CLI, append the --force flag to the standard install command.
func update(mgr vscodeManager, editorPath string) error {
	_, err := mgr.runVSCode(editorPath, installExtensionArg, vscodeMarketplaceID, forceArg)
	return err
}

func openFile(mgr vscodeManager, editorPath string, filePath string, line int) (<-chan error, error) {
	if editorPath == "" {
		return nil, errors.New("empty editor path")
	}
	dest := fmt.Sprintf("%s:%d", filePath, line)
	_, err := mgr.runVSCode(editorPath, gotoArg, dest)
	return nil, err
}

// uninstallOldPlugin removes the data of the old plugin which was shipped with kited
func uninstallOldPlugin(mgr vscodeManager, editorPath string) error {
	dest := filepath.Join(mgr.userExtensionsDir(editorPath), "kite.vscode")
	if !fs.DirExists(dest) {
		return nil
	}
	return os.RemoveAll(dest)
}

func readBinaryVersion(out []byte) (string, error) {
	groups := versionMatcher.FindSubmatch(out)
	if len(groups) < 2 {
		return "", fmt.Errorf("could not determine build from %q", out)
	}
	return string(groups[1]), nil
}
