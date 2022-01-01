package jupyterlab

import (
	"context"
	"errors"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
)

// NewManager returns a new JupyterLab plugin manager for all OS's.
func NewManager() (editor.Plugin, error) {
	return &jupyterLab{}, nil
}

type jupyterLab struct{}

func (j *jupyterLab) ID() string {
	return "jupyterlab"
}

func (j *jupyterLab) Name() string {
	return "JupyterLab"
}

func (j *jupyterLab) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		ManualInstallOnly: true,
	}
}

func (j *jupyterLab) DetectEditors(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (j *jupyterLab) DetectRunningEditors(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (j *jupyterLab) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	return system.Editor{}, nil
}

func (j *jupyterLab) IsInstalled(ctx context.Context, editorPath string) bool {
	return false
}

func (j *jupyterLab) Install(ctx context.Context, editorPath string) error {
	return nil
}

func (j *jupyterLab) Uninstall(ctx context.Context, editorPath string) error {
	return nil
}

func (j *jupyterLab) Update(ctx context.Context, editorPath string) error {
	return nil
}

func (j *jupyterLab) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	return nil, errors.New("Not Implemented")
}
