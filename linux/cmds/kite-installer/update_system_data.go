package main

import (
	"os"

	"github.com/mitchellh/cli"
)

type updateSystemDataCommand struct {
}

func (i *updateSystemDataCommand) Help() string {
	return ""
}

func (i *updateSystemDataCommand) Synopsis() string {
	return "updates desktop, autostart and systemd files only"
}

func (i *updateSystemDataCommand) Run(args []string) int {
	var ui cli.Ui
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	prefix := "[updater] "
	ui = &cli.PrefixedUi{
		AskPrefix:       prefix,
		AskSecretPrefix: prefix,
		OutputPrefix:    prefix,
		InfoPrefix:      prefix,
		ErrorPrefix:     prefix,
		WarnPrefix:      prefix,
		Ui:              ui,
	}

	// the update-system-data command can be called by the user on the commandline
	// or by the previous updater during the update workflow
	// we assume that the user doesn't disable the kite service before calling the command
	// the previous updater does disable the service before calling this
	// we're stopping as a safeguard but do not return an error because it might be already stopped
	_ = stopAndDisableUpdaterService()
	// same for the autostart service
	_ = stopAndDisableAutostartService()

	if status := installSystemData(ui, "system-data-only"); status != 0 {
		return status
	}

	ui.Info("installed system data")
	return 0
}
