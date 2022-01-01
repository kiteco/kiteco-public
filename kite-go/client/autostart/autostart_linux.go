// +build !standalone

package autostart

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

const autostartFile = ".config/autostart/kite-autostart.desktop"

var (
	disabledKey    = "Hidden="
	disabledRegexp = regexp.MustCompile("Hidden=(true|false)")
)

func setEnabled(enabled bool) error {
	disabled := "true"
	if enabled {
		disabled = "false"
	}

	homeDir := os.ExpandEnv("$HOME")
	autostartPath := filepath.Join(homeDir, autostartFile)

	config, err := ioutil.ReadFile(autostartPath)
	if err != nil {
		return err
	}

	config = setDisabled(config, disabled)

	return ioutil.WriteFile(autostartPath, config, 0600)
}

func setDisabled(input []byte, disabled string) []byte {
	match := disabledRegexp.Find(input)
	if match == nil {
		input = append(input, []byte("\n"+disabledKey+disabled)...)
	} else {
		input = disabledRegexp.ReplaceAll(input, []byte(disabledKey+disabled))
	}
	return input
}
