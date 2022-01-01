package atom

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/shared"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
)

func TestBasics(t *testing.T) {
	mgr := newTestManager(&process.MockManager{})
	require.EqualValues(t, atomID, mgr.ID())
	require.EqualValues(t, atomName, mgr.Name())
}

func TestReadVersion(t *testing.T) {
	version, err := readAtomVersion([]byte("invalid"))
	require.Error(t, err)
	require.Empty(t, version)
}

func testBasicInstallFlow(t *testing.T, mgr editor.Plugin, atomPath string) {
	if resolved, err := filepath.EvalSymlinks(atomPath); err == nil {
		atomPath = resolved
	}

	require.False(t, mgr.IsInstalled(context.Background(), atomPath), "expected that the plugin isn't installed initially")

	paths, err := mgr.DetectEditors(context.Background())
	require.NoError(t, err)

	editors := shared.MapEditors(context.Background(), paths, mgr)
	require.Len(t, editors, 1, "expected that %s is detected as valid ide installation", atomPath)
	require.EqualValues(t, atomPath, editors[0].Path, "expected that %s is detected as valid ide installation", atomPath)

	err = mgr.Install(context.Background(), atomPath)
	require.NoError(t, err)
	require.True(t, mgr.IsInstalled(context.Background(), atomPath))

	err = mgr.Update(context.Background(), atomPath)
	require.NoError(t, err)
	require.True(t, mgr.IsInstalled(context.Background(), atomPath))

	err = mgr.Uninstall(context.Background(), atomPath)
	require.NoError(t, err)
	require.False(t, mgr.IsInstalled(context.Background(), atomPath))
}
