package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	texttemplate "text/template"

	"github.com/kiteco/kiteco/kite-go/client/platform/installid"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/mitchellh/cli"
)

type installCommand struct{
	localManager *localManager
}

func (i *installCommand) Help() string {
	return `install [--download] [--no-launch]
	--download: only download the data needed for an installation. Don't install and launch Kite after the download. 
				The data is stored at $HOME/.local/share/kite/ .
	--no-launch: don't launch Kite after the installation was completed`
}

func (i *installCommand) Synopsis() string {
	return "installs kite"
}

func (i *installCommand) Run(args []string) int {
	var ui cli.Ui
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	prefix := "[installer] "
	ui = &cli.PrefixedUi{
		AskPrefix:       prefix,
		AskSecretPrefix: prefix,
		OutputPrefix:    prefix,
		InfoPrefix:      prefix,
		ErrorPrefix:     prefix,
		WarnPrefix:      prefix,
		Ui:              ui,
	}

	err := os.MkdirAll(i.localManager.basePath, 0755)
	if err != nil {
		ui.Error(fmt.Sprintf("unable to create base path at %s: %s", i.localManager.basePath, err.Error()))
		rollbarError("unable to create base path", "install", err)
		rollback(ui)
		return 1
	}

	if localVersion, err := i.localManager.currentVersion(); err != nil {
		ui.Error(fmt.Sprintf("unable to determine local version: %s", err.Error()))
		rollbarError("unable to determine local version", "install", err)
		rollback(ui)
		return 1
	} else if localVersion != "" {
		// shouldn't happen, because main already check and redirected for this case
		ui.Error("kite seems to be installed already. Terminating.")
		return 1
	}

	lock := newFileLock(i.localManager.lockFilePath())
	err = lock.Lock()
	if err != nil {
		ui.Error(fmt.Sprintf("failed to create lock file %s", i.localManager.lockFilePath()))
		rollbarError("failed to create kite-update lock file", "install", err)
		return statusLockFailed
	}
	defer lock.Unlock()

	ui.Info("no previous kite installation found")

	// update local version only when a new remote version is available
	updateManager := newUpdateManager()
	installID, ok := installid.IDIfSet()
	if !ok {
		installID = ""
	}
	remoteVersion, err := updateManager.remoteVersion("", installID)
	if err != nil {
		// if errNoUpdateAvailable, that's still a bug, since there is no local version installed.
		ui.Error("unable to retrieve version information for kite. please make sure that linux.kite.com is reachable")
		ui.Error(fmt.Sprintf("error: %s", err.Error()))
		rollbarError("unable to retrieve version information", "install", err)
		rollback(ui)
		return 1
	}

	ui.Info(fmt.Sprintf("latest version is %s, downloading now...", remoteVersion.Version))

	tracker := newDownloadTracker(i.localManager.basePath)
	defer tracker.save()

	err = ensureDownloaded(ui, i.localManager, updateManager, remoteVersion, publicKey, tracker, false)
	if err != nil {
		ui.Error(fmt.Sprintf("failed to download kite: %s", err.Error()))
		rollbarError("failed to download kite", "install", err)
		rollback(ui)
		return 1
	}

	if !contains(args, "--download") {
		ui.Info(fmt.Sprintf("installing version %s", remoteVersion.Version))

		err = install(i.localManager, remoteVersion)
		if err != nil {
			ui.Error(fmt.Sprintf("failed to install kite: %s", err.Error()))
			rollbarError("failed to install kite", "install", err)
			rollback(ui)
			return 1
		}

		if status := installSystemData(ui, "install"); status != 0 {
			rollback(ui)
			return status
		}
	}

	if !contains(args, "--download") && !contains(args, "--no-launch") {
		ui.Info("kite is installed! launching now! happy coding! :)")
		ui.Info("with systemd, run systemctl --user start kite-autostart")
		ui.Info(fmt.Sprintf("without systemd, run %s", filepath.Join(i.localManager.basePath, "kited")))
		ui.Info("\tor launch it using the Applications Menu")

		err = launchKite(i.localManager)
		if err != nil {
			ui.Error(fmt.Sprintf("error launching kite: %s", err.Error()))
			rollbarError("error launching kite", "install", err)
			// no rollback as this isn't a critical error
			return 1
		}
	}

	return 0
}

func rollback(ui cli.Ui) {
	ui.Info("Rolling back installed data...")
	_ = uninstall(ui, true)
}

func renderText(w io.Writer, fn string, payload interface{}) error {
	data, err := Asset(fn)
	if err != nil {
		return errors.Errorf("error getting asset data: %s", err)
	}

	template, err := texttemplate.New("filetemplate").Parse(string(data))
	if err != nil {
		return errors.Errorf("error parsing template: %s", err)
	}

	err = template.Execute(w, payload)
	if err != nil {
		return errors.Errorf("error executing template: %s", err)
	}

	return nil
}
