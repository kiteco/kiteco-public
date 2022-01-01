package jetbrains

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/internal/jetbrains/buildnumber"
	"github.com/kiteco/kiteco/kite-go/client/platform/machine"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

const marketplacePluginID = 15148
const marketplacePluginXMLID = "com.kite.intellij"

var versionTagRegex = regexp.MustCompile("<version>([0-9.]+)</version>")

func downloadBetaChannelPlugin(targetFile io.WriteCloser) error {
	log.Printf("Downloading JetBrains plugin from beta channel....")

	// this URL always points to the latest released plugin version
	resp, err := http.DefaultClient.Get("https://kite-plugin-binaries.s3-us-west-1.amazonaws.com/latest/kite-pycharm.zip")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("Unexpected status downloading the staged JetBrains plugin build. %s: %d", resp.Status, resp.StatusCode)
	}

	_, err = io.Copy(targetFile, resp.Body)
	return err
}

// downloadKitePluginZip downloads the latest compatible release from the JetBrains marketplace.
// The zip file is stored at targetFile. If the file already exists, then an error is returned.
// targetFile is always closed when this method returns.
// See https://plugins.jetbrains.com/docs/marketplace/plugin-update-download.html for the HTTP request.
func downloadKitePluginZip(targetFile io.WriteCloser, ideBuild buildnumber.BuildNumber) error {
	defer targetFile.Close()

	// using the same URL as the IDEs do, this is slightly different from the official JetBrains documentation.
	// The URL redirects to a CDN.
	// The kited bootstrapping is already setting up the default client's proxy.
	machineID, _ := machine.IDIfSet()
	downloadURL := downloadPluginURL(ideBuild, machineID)
	resp, err := http.DefaultClient.Get(downloadURL.String())
	if err != nil {
		return errors.Errorf("error downloading kite plugin zip: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected HTTP status for url %s: %d / %s", downloadURL.String(), resp.StatusCode, resp.Status)
	}

	if resp.Header.Get("Content-Type") != "application/zip" {
		return errors.Errorf("unexpected response content type for url %s: %s", downloadURL.String(), resp.Header.Get("Content-Type"))
	}

	if _, err := io.Copy(targetFile, resp.Body); err != nil {
		return err
	}
	return nil
}

// requestLatestAvailableVersion fetches the version of the latest, compatible version from the JetBrains marketplace
// The version is compatible with the given ideBuild.
func requestLatestAvailableVersion(ideBuild buildnumber.BuildNumber) (string, error) {
	machineID, _ := machine.IDIfSet()
	listURL := listPluginURL(ideBuild, machineID)

	resp, err := http.DefaultClient.Get(listURL.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("unexpected HTTP status %d / %s for plugin list", resp.StatusCode, resp.Status)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/xml") {
		return "", errors.Errorf("unexpected response content type %s for plugin list", resp.Header.Get("Content-Type"))
	}

	xmlBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	version, err := findLatestVersion(string(xmlBytes))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(version), nil
}

func listPluginURL(ideBuild buildnumber.BuildNumber, machineID string) url.URL {
	query := url.Values{}
	query.Add("pluginId", strconv.Itoa(marketplacePluginID))
	query.Add("build", ideBuild.String())
	if machineID != "" {
		query.Add("uuid", machineID)
	}

	listURL := url.URL{
		Host:     "plugins.jetbrains.com",
		Scheme:   "https",
		Path:     "/plugins/list/",
		RawQuery: query.Encode(),
	}
	return listURL
}

func downloadPluginURL(ideBuild buildnumber.BuildNumber, machineID string) url.URL {
	query := url.Values{}
	query.Add("action", "download")
	query.Add("id", marketplacePluginXMLID)
	query.Add("build", ideBuild.String())
	if machineID != "" {
		query.Add("uuid", machineID)
	}

	downloadURL := url.URL{
		Host:     "plugins.jetbrains.com",
		Scheme:   "https",
		Path:     "/pluginManager/",
		RawQuery: query.Encode(),
	}
	return downloadURL
}

// findLatestVersion extracts the latest compatible version from the given XML (following JetBrains XML format)
// It's a separate method to simplify testing.
func findLatestVersion(xml string) (string, error) {
	// the xml contains all compatible versions,
	// the latest releases come first
	// instead of parsing all of the XML, we just locate the first <version>...</version tag and return the text content

	matches := versionTagRegex.FindStringSubmatch(xml)
	if len(matches) != 2 {
		return "", errors.Errorf("unable to find version tag")
	}
	return matches[1], nil
}

// extractZipEntry extracts a single zip entry into the given parent directory
// The relative path and filename of the entry is retained.
func extractZipEntry(f *zip.File, pluginParentDir string) error {
	entryReader, err := f.Open()
	if err != nil {
		return err
	}
	defer entryReader.Close()

	var targetFilePath = filepath.Join(pluginParentDir, f.Name)

	if f.FileInfo().IsDir() {
		return os.MkdirAll(targetFilePath, 0700)
	}

	// safe-guard to create parent hierarchy of a file-entry
	if err := os.MkdirAll(filepath.Dir(targetFilePath), 0700); err != nil && !os.IsExist(err) {
		return err
	}

	out, err := os.Create(targetFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, entryReader); err != nil {
		return err
	}
	return nil
}
