package spyder

import (
	"context"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

// linuxSpyder implements plugin.Plugin
type linuxSpyder struct {
	process process.LinuxProcessManager
}

// NewManager returns a new plugin manager for Spyder
func NewManager(process process.LinuxProcessManager) (editor.Plugin, error) {
	return &linuxSpyder{
		process: process,
	}, nil
}

func (p *linuxSpyder) ID() string {
	return ID
}

func (p *linuxSpyder) Name() string {
	return name
}

func (p *linuxSpyder) InstallConfig(ctx context.Context) *editor.InstallConfig {
	runningEditors, err := p.DetectRunningEditors(ctx)
	isRunning := err == nil && len(runningEditors) > 0

	return &editor.InstallConfig{
		RequiresRestart:          true,
		MultipleInstallLocations: false,
		Running:                  isRunning,
		InstallWhileRunning:      false,
		UpdateWhileRunning:       false,
		UninstallWhileRunning:    false,
	}
}

func (p *linuxSpyder) DetectEditors(ctx context.Context) ([]string, error) {
	packages, err := p.loadConda()
	if err != nil {
		return nil, err
	}

	var configFiles []string
	for _, spyder := range packages {
		if config, err := p.configFilePath(spyder); err == nil {
			configFiles = append(configFiles, config)
		}
	}
	return configFiles, nil
}

func (p *linuxSpyder) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := p.process.List(ctx)
	if err != nil {
		return nil, err
	}

	return list.Matching(ctx, func(ctx context.Context, process process.Process) string {
		cmdline, err := process.CmdlineSliceWithContext(ctx)
		if err == nil && len(cmdline) >= 2 {
			// e.g. /opt/miniconda3/bin/python /opt/miniconda3/bin/spyder
			if filepath.Base(cmdline[1]) == "spyder" {
				// fixme not correct, at other places we're returning config locations as editor path
				return filepath.Dir(cmdline[1])
			}
		}
		return ""
	})
}

func (p *linuxSpyder) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	packages, err := p.loadConda()
	if err != nil {
		return system.Editor{}, err
	}

	for _, spyder := range packages {
		if path, err := p.configFilePath(spyder); err == nil && path == editorPath {
			var compatibility string
			if spyder.majorVersion() < 4 {
				compatibility = "Incompatible version of Spyder"
			}

			return system.Editor{
				Path:            editorPath,
				Version:         spyder.Version,
				Compatibility:   compatibility,
				RequiredVersion: "4.0.0",
			}, nil
		}
	}
	return system.Editor{}, errors.Errorf("unable to provide editor info for %s", editorPath)
}

func (p *linuxSpyder) IsInstalled(ctx context.Context, configFilePath string) bool {
	return fs.FileExists(configFilePath) && isKiteEnabled(configFilePath)
}

func (p *linuxSpyder) Install(ctx context.Context, configFilePath string) error {
	if err := setKiteEnabled(configFilePath, true); err == errConfigFileNotFound {
		return errors.Errorf("This error message typically means that you have not run the Spyder IDE once yet. Please start Spyder to enable Kite for Spyder.")
	} else if err != nil {
		return err
	}
	if couldApplyOptimalSettings(configFilePath) {
		return applyOptimalSettings(configFilePath)
	}
	return nil
}

func (p *linuxSpyder) Uninstall(ctx context.Context, editorPath string) error {
	return setKiteEnabled(editorPath, false)
}

func (p *linuxSpyder) Update(ctx context.Context, editorPath string) error {
	// do nothing
	return nil
}

func (p *linuxSpyder) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	return nil, errors.New("Not Implemented")
}

func (p *linuxSpyder) loadConda() ([]condaPackageInfo, error) {
	// there's no known way to find out where spyder was installed by conda
	// therefore we can only return the location of the config file

	output, err := p.process.Run("conda", "list", "-f", "--json", "spyder")
	if err != nil {
		return nil, err
	}

	return parseCondaPackageList(output)
}

// configFilePath returns the path to Spyder's ini file
func (p *linuxSpyder) configFilePath(info condaPackageInfo) (string, error) {
	home, err := p.process.Home()
	if err != nil {
		return "", err
	}

	if info.isDevVersion() {
		return filepath.Join(home, ".config", "spyder-py3-dev", "config", "spyder.ini"), nil
	}
	return filepath.Join(home, ".config", "spyder-py3", "config", "spyder.ini"), nil
}
