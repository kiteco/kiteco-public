package shared

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-golib/rollbar"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

// InstallOrUpdatePluginAssets tries to atomically update the plugin if it's already installed.
// If it's not yet there, it installs the plugin.
// This requires that extractPluginData extracts the assets into a directory whose name
// is matching the value of 'pluginDirName'
// extractPluginData should return "AlreadyInstalledError" if an update was cancelled
// because the same or a newer version was already installed
func InstallOrUpdatePluginAssets(pluginParentDir, pluginDirName string, extractPluginData func(parentDir string) error) error {
	pluginDir := filepath.Join(pluginParentDir, pluginDirName)

	var err error
	if fs.DirExists(pluginDir) {
		err = UpdatePluginAtomically(pluginDir, extractPluginData)
	} else {
		info, statErr := os.Stat(pluginDir)
		if statErr == nil && !info.IsDir() {
			if info.Mode().IsRegular() {
				rmErr := os.Remove(pluginDir)
				if rmErr != nil {
					log.Printf("failed to remove file: %s %s\n", pluginDir, rmErr)
					rollbar.Error(errors.Errorf("failed to remove file"), pluginDir, rmErr)
				}
			} else {
				log.Printf("pluginDir FileMode is unexpected: %s %s\n", pluginDir, info.Mode())
				rollbar.Error(errors.Errorf("pluginDir FileMode is unexpected"), pluginDir, info.Mode())
			}
		} else if statErr != nil && !os.IsNotExist(statErr) {
			log.Printf("Error getting info on pluginDir: %s %s\n", pluginDir, err)
			rollbar.Error(errors.Errorf("Error getting info on pluginDir"), pluginDir, err)
		}
		err = extractPluginData(pluginParentDir)
	}

	// don't report an error if the plugin was already installed with the same or a newer version
	if err == AlreadyInstalledError {
		return nil
	}
	return err
}

// UpdatePluginAtomically installs new version of the plugins
// it tries to always restore the original data when the update fails
func UpdatePluginAtomically(pluginDir string, extractPluginData func(parentDir string) error) error {
	// the atomic update follows these steps:
	// 1. unpack to location into temp dir, which has to be outside of pluginParentDir
	// 2. rename existing plugin dir
	// 3. move temp dir to plugin dir and rename
	// 4. remove old plugin dir
	// If any of these steps fail, the original plugin must be restored
	pluginDirName := filepath.Base(pluginDir)

	// 1. unpack new plugin version to a temp dir
	tempNewPluginParent, err := ioutil.TempDir("", "kite-plugin-new")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempNewPluginParent)
	if err = extractPluginData(tempNewPluginParent); err != nil {
		// storing plugin in temp dir failed, return
		return err
	}
	tempNewPluginDir := filepath.Join(tempNewPluginParent, pluginDirName)
	if !fs.DirExists(tempNewPluginDir) {
		// the contract of extractPluginData requires that it created
		// a subdir named like the plugin dir
		return errors.Errorf("new plugin wasn't properly unpacked at %s", tempNewPluginDir)
	}

	// 2. move existing plugin dir to temp location
	tempOldPluginParent, err := ioutil.TempDir("", "kite-plugin-old")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempOldPluginParent)
	tempOldPluginDir := filepath.Join(tempOldPluginParent, pluginDirName)
	err = fs.MoveOrCopyDir(pluginDir, tempOldPluginDir)
	if err != nil {
		// MoveOrCopy already restores when needed
		return errors.Errorf("error moving existing plugin dir %s: %v", pluginDir, err)
	}

	// 3. move temp dir to plugin dir and rename
	err = fs.MoveOrCopyDir(tempNewPluginDir, pluginDir)
	if err != nil {
		// failed to install the new plugin version, try to restore
		// we already could have a partially installed plugin, so we need to remove first
		// don't use restore, as it doesn't overwrite files already in target.
		// These could be files from the new plugin version
		_ = os.RemoveAll(pluginDir)
		_ = fs.MoveOrCopyDir(tempOldPluginDir, pluginDir)
		return err
	}

	// 4. cleanup:
	// the deferred calls take care of this

	return nil
}

// UninstallPlugin tries to atomically uninstall the plugin installed at pluginDir.
// It tries to remove directory pluginDir and all files contained in this dir and its subdirs.
// If the removal of the dir or any child element failed, then the original data is restored.
func UninstallPlugin(pluginDir string) error {
	if _, err := os.Stat(pluginDir); err != nil && os.IsNotExist(err) {
		// same behavior as os.RemoveAll, a missing path isn't an error
		return nil
	}

	tempParent, err := ioutil.TempDir("", "kite-uninstall")
	if err != nil {
		return errors.Errorf("unable to create temp directory: %v", err)
	}

	tempPath := filepath.Join(tempParent, "kite-plugin")
	// fixme MoveOrCopyDir falls back to non-atomic calls of copy, followed by a os.RemoveAll
	err = fs.MoveOrCopyDir(pluginDir, tempPath)
	if err != nil {
		return err
	}

	// don't return an error when the removal of the tempParent failed
	// pluginDir was already removed and we assume that the tempDir
	// will be cleaned up from time to time
	_ = os.RemoveAll(tempParent)
	return nil
}
