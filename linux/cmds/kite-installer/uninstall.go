package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/shirou/gopsutil/process"
)

type uninstallCommand struct{}

func (i *uninstallCommand) Help() string {
	return ""
}

func (i *uninstallCommand) Synopsis() string {
	return "uninstalls kite"
}

func (i *uninstallCommand) Run(args []string) int {
	var ui cli.Ui
	ui = &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	prefix := "[uninstaller] "
	ui = &cli.PrefixedUi{
		AskPrefix:       prefix,
		AskSecretPrefix: prefix,
		OutputPrefix:    prefix,
		InfoPrefix:      prefix,
		ErrorPrefix:     prefix,
		WarnPrefix:      prefix,
		Ui:              ui,
	}

	if err := uninstall(ui, false); err != nil {
		return 1
	}
	return 0
}

func uninstall(ui cli.Ui, rollback bool) error {
	localManager := newLocalManager()

	// first, try to shutdown kited via systemd service
	ui.Info("removing kite-autostart systemd service")
	_ = stopAndDisableAutostartService()

	// terminate kited and copilot before uninstalling
	// an error isn't critical here, continue with uninstall
	ui.Info("terminating kite processes")
	_ = terminateProcesses(localManager.basePath, "kite", "kited")

	// this isn't critical, continue with uninstall when an error occurred
	ui.Info("removing kite-updater systemd service")
	_ = stopAndDisableUpdaterService()

	err := removeBindataFiles(ui)
	if err != nil {
		ui.Error(fmt.Sprintf("failed to remove system files: %s", err.Error()))
		rollbarError("failed to remove system files", "uninstall", err)
		return err
	}

	if !exists(localManager.basePath) {
		if !rollback {
			ui.Info("kite is uninstalled. we'd love to hear your thoughts! give us some feedback@kite.com")
		}
		return nil
	}

	err = os.RemoveAll(localManager.basePath)
	if err != nil {
		if !rollback {
			ui.Error(fmt.Sprintf("error removing %s: %s", tildify(localManager.basePath), err.Error()))
		}
		rollbarError("failed to remove kited base path", "uninstall", err)
	}

	ui.Info(fmt.Sprintf("removed %s", tildify(localManager.basePath)))
	if !rollback {
		ui.Info("kite is uninstalled. we'd love to hear your thoughts! give us some feedback@kite.com")
	}

	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

func terminateProcesses(baseDir string, processNames ...string) error {
	executable, _ := os.Executable()
	executableName := filepath.Base(executable)

	processes, err := process.Processes()
	if err != nil {
		return err
	}

	// kill processes in the given order
	// we don't want to let Copilot spawn another instance of kited, for example
	for _, query := range processNames {
		for _, p := range processes {
			// e.g. "kite" or "kited"
			name, _ := p.Name()
			// e.g. /home/user/.local/share/kite/kite-v2.20190516.0/linux-unpacked/kite
			// or /home/user/.local/share/kite/kite-v2.20190516.0/kited
			exe, _ := p.Exe()

			// only terminate processes which match the given name
			// and have an executable stored in the base dir
			// take care not to terminate the current process, which is also stored in baseDir
			matchingName := name == query || filepath.Base(exe) == query
			isCurrent := strings.Contains(name, executableName)
			inBaseDir := strings.Index(exe, baseDir) == 0
			if matchingName && !isCurrent && inBaseDir {
				_ = p.Terminate()
			}
		}
	}
	return nil
}
