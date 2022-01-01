package spyder

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

type windowsSpyder struct {
	process process.WindowsProcessManager
}

// NewManager returns a new plugin manager for Spyder
func NewManager(process process.WindowsProcessManager) (editor.Plugin, error) {
	return &windowsSpyder{
		process: process,
	}, nil
}

func (p *windowsSpyder) ID() string {
	return ID
}

func (p *windowsSpyder) Name() string {
	return name
}

func (p *windowsSpyder) InstallConfig(ctx context.Context) *editor.InstallConfig {
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

func (p *windowsSpyder) DetectEditors(ctx context.Context) ([]string, error) {
	return p.detectAnacondaInstallationDirs(ctx)
}

func (p *windowsSpyder) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := p.process.List(ctx)
	if err != nil {
		return nil, err
	}

	// search for python running spyder-script.py,
	// e.g. "c:\Users\username\Miniconda3\pythonw.exe c:\Users\username\Miniconda3\Scripts\spyder-script.py"
	// then extract the conda base directory from that
	var runningEditors []string
	for _, i := range list {
		// optimization: detect early if the command is run from a conda dir
		// e.g. if it's matching c:\Users\username\Miniconda3\pythonw.exe
		// we're optimizing here because calling CmdlineSlice below is expensive
		exe, _ := i.ExeWithContext(ctx)
		if !p.isCondaDir(filepath.Dir(exe)) {
			continue
		}

		cmdline, err := i.CmdlineSliceWithContext(ctx)
		if err == nil && len(cmdline) >= 2 && strings.ToLower(filepath.Base(cmdline[len(cmdline)-1])) == "spyder-script.py" {
			// map c:\Users\user\Miniconda3\Scripts\spyder-script.py to
			//     c:\Users\user\Miniconda3\
			runningEditors = append(runningEditors, filepath.Dir(filepath.Dir(cmdline[len(cmdline)-1])))
		}
	}
	return shared.DedupePaths(runningEditors), nil
}

func (p *windowsSpyder) EditorConfig(ctx context.Context, condaDir string) (system.Editor, error) {
	packages, err := p.loadCondaPackages(ctx, condaDir)
	if err != nil {
		return system.Editor{}, err
	}

	// we're only supporting the first install of Spyder in the conda directory
	// as there's only one return value

	for _, spyder := range packages {
		var compatibility string
		if spyder.majorVersion() < 4 {
			compatibility = "Incompatible version of Spyder"
		}

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

func (p *windowsSpyder) IsInstalled(ctx context.Context, configFilePath string) bool {
	return fs.FileExists(configFilePath) && isKiteEnabled(configFilePath)
}

func (p *windowsSpyder) Install(ctx context.Context, editorPath string) error {
	if err := setKiteEnabled(editorPath, true); err == errConfigFileNotFound {
		return errors.Errorf("This error message typically means that you have not run the Spyder IDE once yet. Please start Spyder to enable Kite for Spyder.")
	} else if err != nil {
		return err
	}
	if couldApplyOptimalSettings(editorPath) {
		return applyOptimalSettings(editorPath)
	}
	return nil
}

func (p *windowsSpyder) Uninstall(ctx context.Context, editorPath string) error {
	return setKiteEnabled(editorPath, false)
}

func (p *windowsSpyder) Update(ctx context.Context, editorPath string) error {
	// no-op, because Spyder bundles Kite supported
	return nil
}

func (p *windowsSpyder) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	return nil, errors.New("Not Implemented")
}

// loadCondaPackages takes a conda base directory and returns the list of
// available spyder packages
func (p *windowsSpyder) loadCondaPackages(ctx context.Context, condaDir string) ([]condaPackageInfo, error) {
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
func (p *windowsSpyder) detectAnacondaInstallationDirs(ctx context.Context) ([]string, error) {
	condaDirs := p.process.FilterStartMenuTargets(func(path string) bool {
		// e.g.  C:\Users\username\Anaconda3\python.exe
		// or 	 C:\Users\username\Miniconda3\python.exe
		dir := filepath.Dir(path)
		dirName := strings.ToLower(filepath.Base(dir))
		if dirName == "anaconda3" || dirName == "miniconda3" {
			if conda := filepath.Join(dir, "condabin", "conda.bat"); fs.FileExists(conda) {
				return true
			}
		}
		return false
	})

	// map cmd to dir,
	// e.g. C:\Users\username\Anaconda3\python.exe -> C:\Users\username\Anaconda3\
	condaDirs = shared.MapStrings(condaDirs, func(item string) string {
		return filepath.Dir(item)
	})
	return shared.DedupePaths(condaDirs), nil
}

func (p *windowsSpyder) condaCmd(dir string) string {
	return filepath.Join(dir, "condabin", "conda.bat")
}

func (p *windowsSpyder) isCondaDir(dir string) bool {
	return fs.FileExists(p.condaCmd(dir))
}

func (p *windowsSpyder) configFilePathVersion(info condaPackageInfo) (string, error) {
	return p.configFilePath(info.majorVersion(), info.isDevVersion())
}

// configFilePath returns the path to Spyder's ini file
func (p *windowsSpyder) configFilePath(majorVersion int, devVersion bool) (string, error) {
	home, err := p.process.Home()
	if err != nil {
		return "", err
	}

	if devVersion {
		return filepath.Join(home, ".spyder-py3-dev", "config", "spyder.ini"), nil
	}
	return filepath.Join(home, ".spyder-py3", "config", "spyder.ini"), nil
}
