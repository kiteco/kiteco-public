package sidebar

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/mitchellh/go-ps"
)

const (
	keyVisible  = "copilotWasVisible"
	processName = "kite"
)

func newController(settings component.SettingsManager) linuxController {
	return linuxController{
		settings: settings,
	}
}

type linuxController struct {
	settings component.SettingsManager
}

// Start implements Controller
func (d linuxController) Start() error {
	return exec.Command(d.cmdPath(), "--no-sandbox").Start()
}

// Focus implements Controller
func (d linuxController) Focus() error {
	// TODO(tarak): This is a simple workaround for focus that actually works well I think...
	running, err := d.Running()
	if err != nil {
		return err
	}

	if running {
		return d.Start()
	}

	return nil
}

// Stop implements Controller
func (d linuxController) Stop() error {
	process, err := d.copilotProcess()
	if err != nil {
		return err
	}

	return exec.Command("kill", strconv.Itoa(process.Pid())).Start()
}

// Running implements Controller
func (d linuxController) Running() (bool, error) {
	process, err := d.copilotProcess()
	return process != nil, err
}

// SetWasVisible implements Controller
func (d linuxController) SetWasVisible(val bool) error {
	return d.settings.SetObj(keyVisible, val)
}

// WasVisible implements Controller
func (d linuxController) WasVisible() (bool, error) {
	visible, _ := d.settings.GetBool(keyVisible)
	return visible, nil
}

// Notify implements Controller
func (d linuxController) Notify(id string) error {
	cmd := exec.Command(d.cmdPath(), "--no-sandbox", "--notification="+id)
	err := cmd.Start()
	go cmd.Wait() // cleanup resources when the process exits
	return err
}

func (d linuxController) cmdPath() string {
	// first, we check for linux-unpacked/kite as a sibling of the current process's binary
	if exePath, err := os.Executable(); err == nil {
		cmdPath := filepath.Join(filepath.Dir(exePath), "linux-unpacked", processName)
		if _, err := os.Stat(cmdPath); err == nil {
			return cmdPath
		}
	}

	// fallback to ~/.local/share/kite/current/linux-unpacked/kite
	homeDir := os.ExpandEnv("$HOME")
	cmdPath := filepath.Join(homeDir, ".local", "share", "kite", "current", "linux-unpacked", processName)
	if _, err := os.Stat(cmdPath); err == nil {
		return cmdPath
	}

	// try to launch from $PATH as last resort
	return processName
}

func (d linuxController) copilotProcess() (ps.Process, error) {
	list, err := ps.Processes()
	if err != nil {
		return nil, err
	}

	for _, process := range list {
		if process.Executable() == processName {
			return process, nil
		}
	}
	return nil, fmt.Errorf("pid not found")
}
