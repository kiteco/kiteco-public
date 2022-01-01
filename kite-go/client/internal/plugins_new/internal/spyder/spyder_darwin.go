package spyder

import (
	"context"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

type macSpyder struct {
	process         process.MacProcessManager
	commonCondaDirs []string
}

// NewManager returns a new plugin manager for Spyder
func NewManager(process process.MacProcessManager) (editor.Plugin, error) {
	var commonDirs []string
	if home, err := process.Home(); err == nil {
		commonDirs = []string{
			filepath.Join(home, "anaconda3"),
			filepath.Join(home, "miniconda3"),
			filepath.Join(home, "opt", "anaconda3"),
			filepath.Join(home, "opt", "miniconda3"),
		}
	}

	return &macSpyder{
		process:         process,
		commonCondaDirs: commonDirs,
	}, nil
}

func (p *macSpyder) ID() string {
	return ID
}

func (p *macSpyder) Name() string {
	return name
}

func (p *macSpyder) InstallConfig(ctx context.Context) *editor.InstallConfig {
	runningCmds, err := p.DetectRunningEditors(ctx)
	isRunning := err == nil && len(runningCmds) > 0

	return &editor.InstallConfig{
		RequiresRestart:          true,
		MultipleInstallLocations: false,
		Running:                  isRunning,
		InstallWhileRunning:      false,
		UpdateWhileRunning:       false,
		UninstallWhileRunning:    false,
	}
}

func (p *macSpyder) DetectEditors(ctx context.Context) ([]string, error) {
	return p.detectAnacondaInstallationDirs()
}

func (p *macSpyder) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := p.process.AllRunningApplications()
	if err != nil {
		return nil, err
	}

	// search for python running spyder,
	// e.g. "/Users/user/anaconda3/python.app/Contents/MacOS/python /Users/user/anaconda3/bin/spyder"
	// then extract the conda base directory from that
	var runningEditors []string
	for _, proc := range list {
		if len(proc.Arguments) >= 1 && filepath.Base(proc.Arguments[len(proc.Arguments)-1]) == "spyder" {
			// map /Users/user/anaconda3/bin/spyder to
			//     /Users/user/anaconda3/
			dir := filepath.Dir(filepath.Dir(proc.Arguments[len(proc.Arguments)-1]))
			if p.isCondaDir(dir) {
				runningEditors = append(runningEditors, dir)
			}
		}
	}
	return shared.DedupePaths(runningEditors), nil
}

func (p *macSpyder) EditorConfig(ctx context.Context, condaDir string) (system.Editor, error) {
	packages, err := p.loadCondaPackages(condaDir)
	if err != nil {
		return system.Editor{}, err
	}

	// we return the mapping of the first spyder package found in the list
	// we're assuming that it's not possible to install two versions of Spyder
	// in the same anaconda environment

	for _, spyder := range packages {
		var compatibility string
		if spyder.majorVersion() < 4 {
			compatibility = "Incompatible version of Spyder"
		}

		// using configPath as Path, otherwise
		// we'd have to query "conda list" again during install and uninstall
		// to retrieve the version of spyder

		configPath, _ := p.configFilePathVersion(spyder)
		return system.Editor{
			Path:            configPath,
			Version:         spyder.Version,
			Compatibility:   compatibility,
			RequiredVersion: "4.0.0",
		}, nil
	}
	return system.Editor{}, errors.Errorf("unable to provide editor info for conda dir %s", condaDir)
}

func (p *macSpyder) IsInstalled(ctx context.Context, configFilePath string) bool {
	return fs.FileExists(configFilePath) && isKiteEnabled(configFilePath)
}

func (p *macSpyder) Install(ctx context.Context, configFilePath string) error {
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

func (p *macSpyder) Uninstall(ctx context.Context, configPath string) error {
	return setKiteEnabled(configPath, false)
}

func (p *macSpyder) Update(ctx context.Context, configPath string) error {
	// no-op, because Spyder bundles Kite supported
	return nil
}

func (p *macSpyder) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	return nil, errors.New("Not Implemented")
}

// loadCondaPackages takes a conda base directory and returns the list of
// available spyder packages
func (p *macSpyder) loadCondaPackages(condaDir string) ([]condaPackageInfo, error) {
	if !p.isCondaDir(condaDir) {
		return nil, errors.Errorf("conda not found at %s", condaDir)
	}

	condaCmd := p.condaCmd(condaDir)
	output, err := p.process.Run(condaCmd, "list", "-f", "--json", "spyder")
	if err != nil {
		return nil, errors.Errorf("error loading conda packages with %s", condaCmd)
	}
	return parseCondaPackageList(output)
}

// detectAnacondaInstallationDirs detects conda command which are available on the current system
// it supports Anaconda3 and Miniconda3 distributions
func (p *macSpyder) detectAnacondaInstallationDirs() ([]string, error) {
	// lookup in $PATH, then map cmd to dir,
	// e.g. /Users/username/miniconda3/bin/conda -> /Users/username/miniconda3/
	condaDirs := p.process.FindBinary("conda")
	condaDirs = shared.MapStrings(condaDirs, func(item string) string {
		return filepath.Dir(filepath.Dir(item))
	})

	// try common locations, as conda is only in $PATH when `conda activate` was called,
	// e.g. manually or in the shell setup code
	for _, dir := range p.commonCondaDirs {
		if p.isCondaDir(dir) {
			condaDirs = append(condaDirs, dir)
		}
	}

	return shared.DedupePaths(condaDirs), nil
}

func (p *macSpyder) condaCmd(dir string) string {
	return filepath.Join(dir, "bin", "conda")
}

func (p *macSpyder) isCondaDir(dir string) bool {
	return fs.FileExists(p.condaCmd(dir))
}

func (p *macSpyder) configFilePathVersion(info condaPackageInfo) (string, error) {
	return p.configFilePath(info.majorVersion(), info.isDevVersion())
}

// configFilePath returns the path to Spyder's ini file
func (p *macSpyder) configFilePath(majorVersion int, devVersion bool) (string, error) {
	home, err := p.process.Home()
	if err != nil {
		return "", err
	}

	if devVersion {
		return filepath.Join(home, ".spyder-py3-dev", "config", "spyder.ini"), nil
	}
	return filepath.Join(home, ".spyder-py3", "config", "spyder.ini"), nil
}
