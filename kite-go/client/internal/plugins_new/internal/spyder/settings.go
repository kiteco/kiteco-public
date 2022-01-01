package spyder

import (
	"context"
	"strconv"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

// SettingsStatus returns if the optimal settings are applied to all detected installation of spyder
// and if there's at least one running editor
func SettingsStatus(ctx context.Context, spyder editor.Plugin) (optimalSettings bool, runningEditor bool, err error) {
	if spyder == nil || spyder.ID() != ID {
		return true, false, errors.Errorf("invalid spyder plugin")
	}

	// default to true to avoid unnecessary notifications in Copilot
	optimalSettings = true
	if editors, err := spyder.DetectEditors(ctx); err == nil && len(editors) > 0 {
		for _, editorPath := range editors {
			info, err := spyder.EditorConfig(ctx, editorPath)
			if err == nil && info.Compatibility == "" && couldApplyOptimalSettings(info.Path) {
				optimalSettings = false
				break
			}
		}
	}

	running, err := spyder.DetectRunningEditors(ctx)
	runningEditor = err == nil && len(running) > 0

	return optimalSettings, runningEditor, err
}

// ApplyOptimalSettings applies optimal spyder settings to all installations detected by the given Spyder manager
func ApplyOptimalSettings(ctx context.Context, spyder editor.Plugin) error {
	if spyder == nil || spyder.ID() != ID {
		return errors.Errorf("invalid spyder plugin")
	}

	editors, err := spyder.DetectEditors(ctx)
	if err != nil {
		return err
	}

	for _, editorPath := range editors {
		info, err := spyder.EditorConfig(ctx, editorPath)
		if err == nil && info.Compatibility == "" && couldApplyOptimalSettings(info.Path) {
			if err = applyOptimalSettings(info.Path); err != nil {
				return err
			}
		}
	}
	return nil
}

// couldApplyOptimalSettings return true if the optimized settings may be applied
func couldApplyOptimalSettings(configFile string) bool {
	if !isKiteEnabled(configFile) {
		return false
	}

	if v, err := getSpyderConfigValue(configFile, "editor", "automatic_completions"); err != nil || v != "True" {
		return false
	}

	var completionChars int
	if v, err := getSpyderConfigValue(configFile, "editor", "automatic_completions_after_chars"); err != nil {
		return false
	} else if completionChars, err = strconv.Atoi(v); err != nil {
		return false
	}

	var completionDelay int
	if v, err := getSpyderConfigValue(configFile, "editor", "automatic_completions_after_ms"); err != nil {
		return false
	} else if completionDelay, err = strconv.Atoi(v); err != nil {
		return false
	}

	return completionChars > 1 || completionDelay > 100
}

// ApplyOptimalSettings updates Spyder's config file with the optimized settings
func applyOptimalSettings(configFile string) error {
	if err := setSpyderConfigValue(configFile, "editor", "automatic_completions_after_chars", "1"); err != nil {
		return err
	}
	return setSpyderConfigValue(configFile, "editor", "automatic_completions_after_ms", "100")
}
