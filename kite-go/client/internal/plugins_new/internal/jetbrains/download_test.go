package jetbrains

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/jetbrains/buildnumber"
	"github.com/stretchr/testify/require"
)

func Test_DownloadBetaBuild(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "kite-pycharm*.zip")
	require.NoError(t, err)
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	err = downloadBetaChannelPlugin(tempFile)
	require.NoError(t, err)
}

func Test_Download(t *testing.T) {
	skipMarketplaceDownloads(t)

	tempFile, err := ioutil.TempFile("", "kite-pycharm*.zip")
	require.NoError(t, err)
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	ideBuild, err := buildnumber.FromString("IC-202.6770.20")
	require.NoError(t, err)

	err = downloadKitePluginZip(tempFile, ideBuild)
	require.NoError(t, err)
}

func Test_FetchLatestVersion(t *testing.T) {
	skipMarketplaceDownloads(t)

	ideBuild, err := buildnumber.FromString("IC-202.6770.20")
	require.NoError(t, err)

	version, err := requestLatestAvailableVersion(ideBuild)
	require.NoError(t, err)
	require.NotEmpty(t, version)
}

// this downloads and extracts the beta channel plugin
func Test_UnzipBetaBuild(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "kite-pycharm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ideBuild, err := buildnumber.FromString("IU-202.6770.20")
	require.NoError(t, err)

	// downloads from beta channel
	err = extractPluginData(ideBuild, tempDir, tempDir, true)
	require.NoError(t, err)
}

func Test_FindVersionInXML(t *testing.T) {
	xml, err := ioutil.ReadFile(filepath.Join("test", "plugins-list-response.xml"))
	require.NoError(t, err)

	version, err := findLatestVersion(string(xml))
	require.NoError(t, err)
	require.EqualValues(t, "1.8.9", version)

	// test invalid data
	_, err = findLatestVersion("<tag>no version in here</tag>")
	require.Error(t, err)
}

func Test_URLs(t *testing.T) {
	build, err := buildnumber.FromString("IC-201.1234.5")
	require.NoError(t, err)

	listURL := listPluginURL(build, "unique-machine-id")
	require.EqualValues(t, "https://plugins.jetbrains.com/plugins/list/?build=IC-201.1234.5&pluginId=15148&uuid=unique-machine-id", listURL.String())
	listURL = listPluginURL(build, "")
	require.EqualValues(t, "https://plugins.jetbrains.com/plugins/list/?build=IC-201.1234.5&pluginId=15148", listURL.String())

	downloadURL := downloadPluginURL(build, "unique-machine-id")
	require.EqualValues(t, "https://plugins.jetbrains.com/pluginManager/?action=download&build=IC-201.1234.5&id=com.kite.intellij&uuid=unique-machine-id", downloadURL.String())
	downloadURL = downloadPluginURL(build, "")
	require.EqualValues(t, "https://plugins.jetbrains.com/pluginManager/?action=download&build=IC-201.1234.5&id=com.kite.intellij", downloadURL.String())
}
