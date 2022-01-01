package vscode

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/fs"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

var (
	// common binary names.
	// Arch linux has code-oss and the corresponding extensions dir $HOME/.vscode-oss/extensions
	binaryNames = []string{"code-oss", "code-insiders", "code"}
	// Binary locations within /snap/{editor}/current/
	snapInstalls = map[string]string{
		"code":          "usr/share/code/bin/code",
		"code-insiders": "usr/share/code-insiders/bin/code-insiders",
	}
)

// NewManager returns a new windows plugin manager suitable for Linux
func NewManager(process process.LinuxProcessManager) (editor.Plugin, error) {
	home, err := process.Home()
	if err != nil {
		return nil, err
	}

	return &linuxVSCode{
		process:  process,
		userHome: home,
	}, nil
}

func newTestManager(baseDir string, process *process.MockManager) *linuxVSCode {
	return &linuxVSCode{
		process:  process,
		userHome: baseDir,
	}
}

type linuxVSCode struct {
	process  process.LinuxProcessManager
	userHome string
}

// ID implements editor.Plugin
func (m *linuxVSCode) ID() string {
	return vscodeID
}

// Name implements editor.Plugin
func (m *linuxVSCode) Name() string {
	return vscodeName
}

// InstallConfig implements editor.Plugin
func (m *linuxVSCode) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:          true,
		MultipleInstallLocations: true,
		Running:                  m.isRunning(ctx),
		InstallWhileRunning:      true,
		UpdateWhileRunning:       true,
		UninstallWhileRunning:    true,
	}
}

// DetectEditors implements editor.Plugin
func (m *linuxVSCode) DetectEditors(ctx context.Context) ([]string, error) {
	var vscodePaths []string
	for _, name := range binaryNames {
		path, err := exec.LookPath(name)
		if err != nil {
			log.Println(err)
			continue
		}
		path, err = shared.SnapPath(path, filepath.Base(path), snapInstalls[name])
		if err != nil {
			log.Println(err)
			continue
		}
		vscodePaths = append(vscodePaths, path)
	}
	return vscodePaths, nil
}

// DetectRunningEditors implements editor.Plugin
func (m *linuxVSCode) DetectRunningEditors(ctx context.Context) ([]string, error) {
	list, err := m.process.List(ctx)
	if err != nil {
		return nil, err
	}

	return list.Matching(ctx, func(ctx context.Context, process process.Process) string {
		// the tar.gz and .deb downloads of VSCode contains a single binary "code"
		// the snap package also contains a binary called "code"
		// there's code a install-dir/code and install-dir/bin/code, the latter is the one we want
		// only bin/code is accepting the --version arg
		if exe, err := process.ExeWithContext(ctx); err == nil {
			dir := filepath.Dir(exe)
			name := filepath.Base(exe)
			if filepath.Base(dir) == "bin" && (name == "code" || name == "code-insiders") {
				return exe
			}

			// /bin/code of the official tar.gz package is a wrapper which terminates after launching "/code"
			// we have to check for /bin/code when /code was found
			codePath := filepath.Join(dir, "bin", "code")
			if fs.FileExists(codePath) {
				return codePath
			}

			insidersPath := filepath.Join(dir, "bin", "code-insiders")
			if fs.FileExists(insidersPath) {
				return insidersPath
			}
		}
		return ""
	})
}

// EditorConfig implements editor.Plugin
func (m *linuxVSCode) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	// MapUniqueEditors skips empty path values
	v, err := m.readVSCodeBinaryVersion(editorPath)
	if err != nil {
		return system.Editor{}, err
	}
	return system.Editor{
		Path:    editorPath,
		Version: v,
	}, nil
}

// IsInstalled implements editor.Plugin
func (m *linuxVSCode) IsInstalled(ctx context.Context, editorPath string) bool {
	return isInstalled(m, editorPath)
}

// Install implements editor.Plugin
func (m *linuxVSCode) Install(ctx context.Context, editorPath string) error {
	return install(m, editorPath)
}

// Uninstall implements editor.Plugin
func (m *linuxVSCode) Uninstall(ctx context.Context, editorPath string) error {
	// fixme remove this? calling the cli is slow
	if !m.IsInstalled(ctx, editorPath) {
		return nil
	}
	return uninstall(m, editorPath)
}

// Update implements editor.Plugin
func (m *linuxVSCode) Update(ctx context.Context, editorPath string) error {
	return update(m, editorPath)
}

func (m *linuxVSCode) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	// If editorPath is sent by the plugin, the binary location needs to be found
	if strings.HasSuffix(editorPath, "/resources/app") {
		root := strings.Replace(editorPath, "/resources/app", "/bin", -1)
		// Get binary inside editorPath/bin
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			for _, binary := range binaryNames {
				if filepath.Base(path) == binary {
					editorPath = path
				}
			}
			return nil
		})
	}
	return openFile(m, editorPath, filePath, line)
}

func (m *linuxVSCode) runVSCode(editorPath string, args ...string) ([]byte, error) {
	return m.process.Run(editorPath, args...)
}

func (m *linuxVSCode) readVSCodeBinaryVersion(path string) (string, error) {
	out, err := m.process.Run(path, "--version")
	if err != nil {
		return "", err
	}
	return readBinaryVersion(out)
}

func (m *linuxVSCode) cliPath(editorPath string) string {
	return editorPath
}

func (m *linuxVSCode) userExtensionsDir(editorPath string) string {
	var extensionsDir string
	switch filepath.Base(editorPath) {
	case "code-oss":
		extensionsDir = ".vscode-oss"
	case "code-insiders":
		extensionsDir = ".vscode-insiders"
	default:
		extensionsDir = ".vscode"
	}
	return filepath.Join(m.userHome, extensionsDir, "extensions")
}

func (m *linuxVSCode) isRunning(ctx context.Context) bool {
	for _, name := range binaryNames {
		result, err := m.process.IsProcessRunning(ctx, name)
		if err == nil && result {
			return true
		}
	}
	return false
}
