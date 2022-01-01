package vscode

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
)

func TestBasics(t *testing.T) {
	dir, cleanup := shared.SetupTempDir(t, "kite-vscode")
	defer cleanup()

	mgr := newTestManager(dir, &process.MockManager{})
	require.EqualValues(t, vscodeID, mgr.ID())
	require.EqualValues(t, vscodeName, mgr.Name())
}

func TestReadVersion(t *testing.T) {
	version, err := readBinaryVersion([]byte("invalid"))
	require.Error(t, err)
	require.Empty(t, version)
}

func testBasicInstallFlow(t *testing.T, mgr editor.Plugin, vscodePath string) {
	require.False(t, mgr.IsInstalled(context.Background(), vscodePath), "expected that the plugin isn't installed initially")

	paths, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)
	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Len(t, editors, 1, "expected that %s is detected as valid ide installation", vscodePath)
	require.EqualValues(t, vscodePath, editors[0].Path, "expected that %s is detected as valid ide installation", vscodePath)

	err = mgr.Install(context.Background(), vscodePath)
	require.NoError(t, err)
	require.True(t, mgr.IsInstalled(context.Background(), vscodePath))

	err = mgr.Update(context.Background(), vscodePath)
	require.NoError(t, err)
	require.True(t, mgr.IsInstalled(context.Background(), vscodePath))

	err = mgr.Uninstall(context.Background(), vscodePath)
	require.NoError(t, err)
	require.False(t, mgr.IsInstalled(context.Background(), vscodePath))
}
