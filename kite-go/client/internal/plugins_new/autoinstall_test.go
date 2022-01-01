package plugins

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Autoinstall(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	assert.NoError(t, err)
	defer s.Close()

	dummyPath := s.GetFilePath("dummy-editor")
	err = os.Mkdir(dummyPath, 0700)
	require.NoError(t, err)

	dummyPathRunning := s.GetFilePath("dummy-editor-running")
	err = os.Mkdir(dummyPathRunning, 0700)
	require.NoError(t, err)

	dummy := MockEditor{
		id:             "dummy",
		name:           "Dummy Editor",
		isRunning:      false,
		isInstalled:    false,
		installedPaths: []string{dummyPath},
		runningPaths:   []string{dummyPathRunning},
	}

	pluginMgr := NewTestManager(system.Options{DevMode: false}, &dummy)
	err = s.SetupComponents(nil, settings.NewTestManager(), nil, nil, pluginMgr)
	assert.NoError(t, err)

	// no installation with the setting == false
	err = s.Settings.SetObj(settings.AutoInstallPluginsEnabledKey, false)
	require.NoError(t, err)
	installed := pluginMgr.AutoInstallPlugins(context.Background())
	require.EqualValues(t, 0, installed, "plugins must not be installed")
	require.EqualValues(t, 0, dummy.installedCount, "plugins must not be installed")

	// revert to allow installs
	err = s.Settings.SetObj(settings.AutoInstallPluginsEnabledKey, true)
	require.NoError(t, err)

	// no installation when the editor was encountered
	pluginMgr.encountered[dummy.id] = true
	installed = pluginMgr.AutoInstallPlugins(context.Background())
	require.EqualValues(t, 0, installed, "plugins must not be installed")
	require.EqualValues(t, 0, dummy.installedCount, "plugins must not be installed")

	// install
	pluginMgr.encountered[dummy.id] = false
	installed = pluginMgr.AutoInstallPlugins(context.Background())
	require.EqualValues(t, 1, installed)
	require.EqualValues(t, 1, dummy.installedCount)

	// resetting the auto-installed plugins returns the currently stored values before deleting them
	var ids []string
	err = s.KitedClient.DeleteJSON("/clientapi/plugins/auto_installed", nil, &ids)
	require.NoError(t, err)
	require.EqualValues(t, []string{"dummy"}, ids)

	// get after reset returns an empty set of ids
	resp, _ := s.KitedClient.Get("/clientapi/plugins/auto_installed")
	require.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

type MockEditor struct {
	id             string
	name           string
	isRunning      bool
	isInstalled    bool
	installedPaths []string
	runningPaths   []string

	installedCount   int
	updatedCount     int
	uninstalledCount int
}

func (m *MockEditor) ID() string {
	return m.id
}

func (m *MockEditor) Name() string {
	return m.name
}

func (m *MockEditor) InstallConfig(ctx context.Context) *editor.InstallConfig {
	return &editor.InstallConfig{
		RequiresRestart:          false,
		MultipleInstallLocations: false,
		Running:                  m.isRunning,
		InstallWhileRunning:      true,
		UpdateWhileRunning:       true,
		UninstallWhileRunning:    true,
	}
}

func (m *MockEditor) DetectEditors(ctx context.Context) ([]string, error) {
	return m.installedPaths, nil
}

func (m *MockEditor) DetectRunningEditors(ctx context.Context) ([]string, error) {
	return m.runningPaths, nil
}

func (m *MockEditor) EditorConfig(ctx context.Context, editorPath string) (system.Editor, error) {
	return system.Editor{
		Path:          "",
		Version:       "",
		Compatibility: "",
	}, nil
}

func (m *MockEditor) IsInstalled(ctx context.Context, editorPath string) bool {
	return m.isInstalled
}

func (m *MockEditor) Install(ctx context.Context, editorPath string) error {
	m.installedCount++
	return nil
}

func (m *MockEditor) Uninstall(ctx context.Context, editorPath string) error {
	m.uninstalledCount++
	return nil
}

func (m *MockEditor) Update(ctx context.Context, editorPath string) error {
	m.updatedCount++
	return nil
}

func (m *MockEditor) OpenFile(ctx context.Context, id string, editorPath string, filePath string, line int) (<-chan error, error) {
	return nil, nil
}
