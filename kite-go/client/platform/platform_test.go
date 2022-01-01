package platform

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/platform/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DebugFile(t *testing.T) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "kite")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	file, err := os.Create(filepath.Join(tempDir, "DEBUG"))
	require.NoError(t, err)
	file.Close()

	platform, err := NewTestPlatform(tempDir)
	require.NoError(t, err)
	assert.True(t, platform.DevMode, "DEBUG file must enable dev mode")

	//this test only makes sense on systems where the debug flag is not set by default
	if !version.IsDebugBuild() {
		//remove and test disabled dev mode
		err = os.Remove(file.Name())
		require.NoError(t, err)
		platform, err = NewTestPlatform(tempDir)
		require.NoError(t, err)
		assert.False(t, platform.DevMode, "Removed DEBUG file must disable dev mode")
	}
}

func Test_Version(t *testing.T) {
	p, err := NewTestPlatform("")
	require.NoError(t, err)
	defer os.RemoveAll(p.KiteRoot)

	assert.NotEmpty(t, p.ClientVersion)
}

func Test_MachineID(t *testing.T) {
	p, err := NewTestPlatform("")
	require.NoError(t, err)
	defer os.RemoveAll(p.KiteRoot)

	assert.NotEmpty(t, p.MachineID)
}

func Test_UnitTestModeDev(t *testing.T) {
	p, err := NewTestPlatform("")
	require.NoError(t, err)
	defer os.RemoveAll(p.KiteRoot)

	assert.True(t, p.IsUnitTestMode, "Unit test mode must be enabled if NewTestPlatform was used")
}

func Test_UnitTestModeProd(t *testing.T) {
	p, err := NewPlatform()
	require.NoError(t, err)
	assert.False(t, p.IsUnitTestMode, "Unit test mode must not be set in a platform used in a production environment")
}

func Test_InstallID(t *testing.T) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "kite")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	p1, err := NewTestPlatform(tempDir)
	require.NoError(t, err)
	assert.NotEmpty(t, p1.InstallID, "A new install id must be generated for a new install")

	_, err = os.Stat(filepath.Join(tempDir, "installid"))
	require.NoError(t, err, "The install id must be saved into $KITEROOT/installid when it's generated")

	p2, err := NewTestPlatform(tempDir)
	require.NoError(t, err)
	assert.EqualValues(t, p1.InstallID, p2.InstallID, "InstallID must be persisted when it's created for the first time. Restarts have to load it from disk.")

	buf, err := ioutil.ReadFile(filepath.Join(tempDir, "installid"))
	require.NoError(t, err)
	alphaNumericDashes := regexp.MustCompile("^[a-zA-Z0-9-]+$")
	assert.True(t, alphaNumericDashes.Match(buf), string(buf))
}
