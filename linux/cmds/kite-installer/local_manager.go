package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

var (
	errKiteNotReady = errors.New("kite not ready for restart")
)

// localManager handles the locally installed versions of kite
// and interacts with kited on the local machine
type localManager struct {
	basePath   string
	restartURL string
	isReadyURL string
}

// newLocalManager returns a new localManager with default settings for production environments,
// the default installation dir is $HOME/.local/share/kite
func newLocalManager() *localManager {
	return &localManager{
		basePath:   filepath.Join(os.ExpandEnv("$HOME"), ".local", "share", "kite"),
		isReadyURL: "http://127.0.0.1:46624/clientapi/update/readyToRestart",
		restartURL: "http://127.0.0.1:46624/clientapi/update/restart",
	}
}

// currentVersion returns the latest version which is installed locally
func (m *localManager) currentVersion() (string, error) {
	link := filepath.Join(m.basePath, "current")
	if _, err := os.Stat(link); os.IsNotExist(err) {
		return "", nil
	}

	target, err := filepath.EvalSymlinks(link)
	if err != nil {
		return "", err
	}

	targetFile := filepath.Base(target)
	if !strings.HasPrefix(targetFile, "kite-v") {
		return "", errors.Errorf("current version not found")
	}
	return targetFile[6:], nil
}

// installedVersions returns the versions which are installed locally
func (m *localManager) installedVersions() ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(m.basePath, "kite-v*"))
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return []string{}, nil
	}

	var versions []string
	for _, match := range matches {
		name := filepath.Base(match)
		versions = append(versions, name[6:])
	}

	return versions, nil
}

// lockFilePath returns where we download the update package for a given version
func (m *localManager) lockFilePath() string {
	return filepath.Join(m.basePath, "kite-update.lock")
}

// downloadTargetPath returns where we download the update package for a given version
func (m *localManager) downloadTargetPath(version Version) string {
	return filepath.Join(m.basePath, fmt.Sprintf("kite-updater-%s.sh", version.Version))
}

// installDirPath returns where a particular version will be installed
func (m *localManager) installDirPath(version string) string {
	return filepath.Join(m.basePath, fmt.Sprintf("kite-v%s", version))
}

// uninstallVersion removes the directory which contains the data of "version" (e.g installDirPath(version))
func (m *localManager) uninstallVersion(version string) error {
	dir := m.installDirPath(version)
	return os.RemoveAll(dir)
}

// updateCurrentLink updates the symbolic link at "<baseDir>/current" to point to the install directory for the provided version
func (m *localManager) updateCurrentLink(version string) error {
	linkPath := filepath.Join(m.basePath, "current")
	err := os.Remove(linkPath)
	if err != nil && !os.IsNotExist(err) {
		log.Printf("error removing symlink to current release: %v", err)
	}

	targetPath := m.installDirPath(version)
	return os.Symlink(targetPath, linkPath)
}

// removeCurrentLink removes the symbolic link at "<baseDir>/current"
func (m *localManager) removeCurrentLink() error {
	linkPath := filepath.Join(m.basePath, "current")
	return os.Remove(linkPath)
}

// isReadyForUpdate returns if kited is currently ready for a restart related to an update
func (m *localManager) isReadyForUpdate() bool {
	resp, err := http.Get(m.isReadyURL)
	if err != nil {
		// Assuming kited isn't reachable, meaning it's not running
		// Should distinguish between other cases by explicitly checking for `kited` running via ps aux
		return true
	}

	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)

	// status 200 is returned when kited is ready for a restart related to an update
	return resp.StatusCode == http.StatusOK
}

// filePathCurrent returns the path pointing to "relativePath" inside of the "current" directory path
func (m *localManager) filePathCurrent(relativePath string) string {
	return filepath.Join(m.basePath, "current", relativePath)
}

// RestartKited triggers kited to restart if it's running at the moment
func (m *localManager) RestartKited() error {
	resp, err := http.Post(m.restartURL, "text/plain", nil)
	// don't report this when kited wasn't running
	if err != nil && !strings.Contains(err.Error(), "connect: connection refused") {
		return err
	}
	if resp != nil {
		// only when no error occurred
		defer resp.Body.Close()
	}
	return nil
}
