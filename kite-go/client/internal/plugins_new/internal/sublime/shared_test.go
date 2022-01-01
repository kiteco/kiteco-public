package sublime

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testBasics(t *testing.T, mgr editor.Plugin) {
	assert.EqualValues(t, id, mgr.ID())
	assert.EqualValues(t, name, mgr.Name())
}

// test that install, update and uninstall succeed
func testInstallUninstallUpdate(t *testing.T, mgr editor.Plugin) {
	err := mgr.Install(context.Background(), "")
	require.NoError(t, err, "installing must succeed")
	assert.True(t, mgr.IsInstalled(context.Background(), ""), "plugin must be installed after a successful call of Install")

	err = mgr.Update(context.Background(), "")
	require.NoErrorf(t, err, "updating must succeed")
	assert.True(t, mgr.IsInstalled(context.Background(), ""), "plugin must still be installed after Update")

	err = mgr.Uninstall(context.Background(), "")
	require.NoErrorf(t, err, "uninstalling must succeed")
	assert.False(t, mgr.IsInstalled(context.Background(), ""), "plugin must be uninstalled after Uninstall")
}

func TestEditorConfig(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "kite-sublime")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	sublimePath := filepath.Join(tmpDir, "sublime")
	err = ioutil.WriteFile(sublimePath, nil, 0700)
	require.NoError(t, err)

	// compatible builds
	config, err := editorConfig(sublimePath, 3000)
	require.NoError(t, err)
	require.Empty(t, config.Compatibility)
	require.Empty(t, config.RequiredVersion)

	config, err = editorConfig(sublimePath, 3123)
	require.NoError(t, err)
	require.Empty(t, config.Compatibility)
	require.Empty(t, config.RequiredVersion)

	// incompatible builds
	config, err = editorConfig(sublimePath, 2999)
	require.NoError(t, err)
	require.EqualValues(t, "build must be 3000 or higher (found 2999)", config.Compatibility)
	require.EqualValues(t, "3000", config.RequiredVersion)

	config, err = editorConfig(sublimePath, 4000)
	require.NoError(t, err)
	require.EqualValues(t, "only builds < 4000 are supported (found 4000)", config.Compatibility)
	require.EqualValues(t, "3000", config.RequiredVersion)
}
