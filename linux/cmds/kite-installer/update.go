package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/platform/installid"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/mitchellh/cli"
)

// exit codes to communicate status back to kited
const (
	statusSuccess         = 0
	statusUpToDate        = 1
	statusLocalNotFound   = 10
	statusRemoteNotFound  = 11
	statusTooManyAttempts = 12
	statusDownloadFailed  = 13
	statusDiskFull        = 14
	statusInstallFailed   = 15
	statusLockFailed      = 16
)

type updateCommand struct {
	localManager *localManager
	selfUpdate   bool
}

func (i *updateCommand) Help() string {
	return ""
}

func (i *updateCommand) Synopsis() string {
	if i.selfUpdate {
		return "updates kite to the latest version, intended to be called by kited itself to perform a self-update"
	}
	return "updates kite to the latest version (use 'update force' to force an update, use 'update silent' to suppress progress information)"
}

func (i *updateCommand) Run(args []string) int {
	var name string
	if i.selfUpdate {
		name = "self-update"
	} else {
		name = "update"
	}

	var forceUpdate bool
	var silentUpdate bool
	for _, arg := range args {
		if arg == "force" {
			forceUpdate = true
		}
		if arg == "silent" {
			silentUpdate = true
		}
	}

	var ui cli.Ui
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	prefix := fmt.Sprintf("[%s] ", name)
	ui = &cli.PrefixedUi{
		AskPrefix:       prefix,
		AskSecretPrefix: prefix,
		OutputPrefix:    prefix,
		InfoPrefix:      prefix,
		ErrorPrefix:     prefix,
		WarnPrefix:      prefix,
		Ui:              ui,
	}

	lock := newFileLock(i.localManager.lockFilePath())
	err := lock.Lock()
	if err != nil {
		ui.Error(fmt.Sprintf("failed to create lock file %s", i.localManager.lockFilePath()))
		rollbarError("failed to create kite-update lock file", name, err)
		return statusLockFailed
	}
	defer lock.Unlock()

	localVersion, err := i.localManager.currentVersion()
	if err != nil {
		ui.Error(fmt.Sprintf("unable to determine local version: %s", err.Error()))
		rollbarError("unable to determine local version", name, err)
		return statusLocalNotFound
	} else if localVersion == "" {
		// shouldn't happen, because main already checked and redirected for this case
		ui.Error("no previous kite installation found. Terminating.")
		return statusLocalNotFound
	}

	ui.Info(fmt.Sprintf("found version %s installed", localVersion))

	// update local version only when a new remote version is available
	updateManager := newUpdateManager()
	installID, ok := installid.IDIfSet()
	if !ok {
		installID = "unknown"
	}
	remoteVersion, err := updateManager.remoteVersion(localVersion, installID)
	switch {
	case err == errNoUpdateAvailable:
		ui.Info("already up to date!")
		if i.selfUpdate {
			// don't let kited restart without an update
			return statusUpToDate
		} else {
			return statusSuccess
		}
	case err != nil:
		ui.Error("unable to retrieve version information for kite. please make sure that linux.kite.com is reachable")
		ui.Error(fmt.Sprintf("error: %s", err.Error()))
		rollbarError("unable to retrieve version information", name, err)
		return statusRemoteNotFound
	}

	tracker := newDownloadTracker(i.localManager.basePath)
	defer tracker.save()

	// skip update after too many failed attempts
	if errorInfo := tracker.get(remoteVersion.Version); errorInfo.total() > 5 {
		ui.Error(fmt.Sprintf("download or validation of %s failed too many times. Exiting now.", remoteVersion.Version))
		err = errors.Errorf("version: %s, failed downloads: %d, failed validations: %d",
			remoteVersion.Version, errorInfo.DownloadErrors, errorInfo.ValidationErrors)
		if !errorInfo.RollbarSent {
			rollbarError("skipping update after too many download or validation errors", name, err)
			errorInfo.RollbarSent = true
		}
		return statusTooManyAttempts
	}

	ui.Info(fmt.Sprintf("latest version is %s, downloading now...", remoteVersion.Version))

	err = ensureDownloaded(ui, i.localManager, updateManager, remoteVersion, publicKey, tracker, silentUpdate)
	if err != nil {
		ui.Error(fmt.Sprintf("failed to download kite: %s", err.Error()))
		rollbarError("failed to download kite", name, err)
		return statusDownloadFailed
	}

	if !i.selfUpdate {
		// don't install if kited isn't ready yet.
		// the next activation of the kite-updater.timer service will perform this check again
		// and will install the already downloaded update when kited is ready
		if !i.localManager.isReadyForUpdate() || forceUpdate {
			ui.Info("not updating because kited isn't ready to restart, terminating")
			return statusSuccess
		}
	}

	ui.Info(fmt.Sprintf("installing version %s", remoteVersion.Version))
	err = install(i.localManager, remoteVersion)
	if err != nil {
		ui.Error(fmt.Sprintf("failed to update kite: %s", err.Error()))
		if eStr := err.Error(); strings.Contains(eStr, "Not enough space") || strings.Contains(eStr, "Disk quota exceeded") {
			return statusDiskFull
		}
		rollbarError("failed to update kite", name, err)
		return statusInstallFailed
	}

	if !i.selfUpdate {
		// stop the autostart and updater services before installing and updating the new service files
		// the service was installed by this updater, therefore we're also disabling it
		// a new set of service files will be installed by "new-updater update-system-data"

		err = stopAndDisableAutostartService()
		if err != nil {
			ui.Error(err.Error())
			rollbarError("failed to disable autostart service", name, err)
		}

		err = stopAndDisableUpdaterService()
		if err != nil {
			ui.Error(err.Error())
			rollbarError("failed to disable updater service", name, err)
			// recover, the service must not remain stopped.
			// but proceed even if the systemd command(s) failed, systemd is optional now
			err = enableAndStartUpdaterService()
			if err != nil {
				rollbarError("failed to restart updater service after failure", name, err)
			}
		}
	}

	// call the new updater to install the new set of system data
	// this installs the new system data files,
	// updates the service configuration
	// It (re)starts the update service, unless kited is calling this command
	updaterPath := i.localManager.filePathCurrent("kite-update")
	ui.Info(fmt.Sprintf("running %s update-system-data...", updaterPath))
	if _, err := os.Stat(updaterPath); err == nil {
		err := updateSystemData(updaterPath)
		if err != nil {
			ui.Error(fmt.Sprintf("error updating system data: %s", err.Error()))
			rollbarError("error updating system data", name, err)

			if !i.selfUpdate {
				ui.Info("reverting to existing system data")
				err = inflateBindataFiles(ui)
				if err != nil {
					ui.Error(fmt.Sprintf("failed to revert system files: %s", err.Error()))
					rollbarError("failed to revert system files", name, err)
				}

				// recover, the service must not remain stopped
				// proceed even if the systemd commands failed, it's optional now
				err = enableAndStartUpdaterService()
				if err != nil {
					rollbarError("failed to restart updater service after revert", name, err)
				}

				err = enableAutostartService()
				if err != nil {
					rollbarError("failed to enable autostart service after revert", name, err)
				}
			}
		}
	} else {
		ui.Error(fmt.Sprintf("unable to locate kite-update binary: %s", err))
		rollbarError("enable to locate kite-update binary", name, err)
	}

	// if kited has called this updater, skip the cleanup. The kited wrapper script is doing this
	// otherwise cleanup old versions
	if !i.selfUpdate {
		if err = i.localManager.RestartKited(); err != nil {
			ui.Error(fmt.Sprintf("error restarting kited after update: %s", err.Error()))
			rollbarError("failed to restart kited after update", name, err)
		}

		removeOldVersions(i.localManager, remoteVersion)
	}

	return statusSuccess
}
