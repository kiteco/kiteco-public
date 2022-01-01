package neovim

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
)

func TestBasics(t *testing.T) {
	mgr := newTestManager(&process.MockManager{})
	require.EqualValues(t, neovimID, mgr.ID())
	require.EqualValues(t, neovimName, mgr.Name())
}

func TestValidVersion(t *testing.T) {
	versionString := "NVIM v0.2.2"
	require.NoError(t, validate(versionString))
}

func TestInvalidVersion(t *testing.T) {
	versionString := "NVIM v0.1.7"
	require.Error(t, validate(versionString))
}

func TestParseVersion(t *testing.T) {
	versionString := "NVIM v0.2.2"
	info, err := parseVersion(versionString)
	require.NoError(t, err)
	require.EqualValues(t, "0.2.2", info.String())
}

func TestParseVersionErr(t *testing.T) {
	versionString := ""
	_, err := parseVersion(versionString)
	require.Error(t, err, errInvalidBinary)
}
