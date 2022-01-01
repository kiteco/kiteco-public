package vim

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/process"
)

func TestBasics(t *testing.T) {
	mgr := newTestManager(&process.MockManager{})
	require.EqualValues(t, vimID, mgr.ID())
	require.EqualValues(t, vimName, mgr.Name())
}

func TestValidate81(t *testing.T) {
	versionString := "VIM - Vi IMproved 8.1 (2016 Sep 12, compiled Aug 17 2018 17:24:51)"
	require.NoError(t, validate(versionString))
}

func TestValidate80(t *testing.T) {
	versionString := "VIM - Vi IMproved 8.0 (2016 Sep 12, compiled Aug 17 2018 17:24:51)"
	patchString := "Included patches: 1-503, 505-680, 682-1283"
	require.NoError(t, validate(versionString+"\n"+patchString))
}

func TestValidateIncompatible(t *testing.T) {
	versionString := "VIM - Vi IMproved 8.0 (2016 Sep 12, compiled Aug 17 2018 17:24:51)"
	patchString := "Included patches: 1-26, 505-680, 682-1283"
	require.Error(t, validate(versionString+"\n"+patchString))
}

func TestValidateNoPatchString(t *testing.T) {
	versionString := "VIM - Vi IMproved 8.0 (2016 Sep 12, compiled Aug 17 2018 17:24:51)"
	patchString := ""
	require.Error(t, validate(versionString+"\n"+patchString), errUnableToFindPatchVersion)
}

func TestGetPatchVersion(t *testing.T) {
	output := "Included patches: 1-503, 505-680, 682-1283"
	patchVersion := getPatchVersion(output)
	require.Equal(t, patchVersion, 503)
}

func TestGetPatchVersionNoMatch(t *testing.T) {
	output := ""
	patchVersion := getPatchVersion(output)
	require.Equal(t, patchVersion, -1)
}

func TestParseVersion(t *testing.T) {
	versionString := "VIM - Vi IMproved 8.0 (2016 Sep 12, compiled Aug 17 2018 17:24:51)"
	info, err := parseVersion(versionString)
	require.NoError(t, err)
	require.EqualValues(t, info.String(), "8.0")
}

func TestParseVersionErr(t *testing.T) {
	versionString := ""
	_, err := parseVersion(versionString)
	require.Error(t, err, errInvalidBinary)
}
