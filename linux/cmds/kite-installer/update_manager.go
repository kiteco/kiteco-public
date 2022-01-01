package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/klauspost/cpuid"
)

var (
	errNoUpdateAvailable = errors.New("no update available")
)

// Version ...
type Version struct {
	// Version number, e.g. 1.2019.0317
	Version string `json:"version,required"`
	// UpdaterURL is the complete URL pointing where the update package is downloaded
	UpdaterURL string `json:"updater_url,required"`
	// Sha256Checksum is the base64 encoded sha256 checksum of the data stored at UpdaterURL
	Sha256Checksum string `json:"sha256,required"`
	// Signature is the base64 encoded RSA signature of the data stored at UpdaterURL
	Signature string `json:"signature,required"`
}

// SignatureBytes returns the signature as bytes suitable for the crypt package
func (v Version) SignatureBytes() ([]byte, error) {
	return base64.StdEncoding.DecodeString(v.Signature)
}

type updateManager struct {
	metadataURL string
	httpClient  *http.Client
}

func newUpdateManager() *updateManager {
	return &updateManager{
		metadataURL: "https://linux.kite.com/linux/kite-app/update-check",
		httpClient: &http.Client{
			Timeout: 0,
		},
	}
}

// remoteVersion retrieves meta-data about the given version.
// It expects the remote content to be valid JSON of the Version struct.
func (m *updateManager) remoteVersion(localVersion, installID string) (Version, error) {
	metadataURL, err := url.Parse(m.metadataURL)
	if err != nil {
		return Version{}, err
	}

	// fallback to a dummy version to be able to use the version comparison below
	if localVersion == "" {
		localVersion = "0.0.0"
	}

	query := url.Values{}
	query.Add("will-attempt-update-if-provided", "true")
	query.Add("install-id", installID)
	query.Add("version", localVersion)

	if !cpuid.CPU.AVX() {
		query.Add("cpu_features_no_avx", "true")
	}

	metadataURL.RawQuery = query.Encode()

	resp, err := m.httpClient.Get(metadataURL.String())
	if err != nil {
		return Version{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusSeeOther {
		return Version{}, errors.Errorf("unexpected http status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Version{}, err
	}

	if len(body) == 0 {
		return Version{}, errNoUpdateAvailable
	}

	var version Version
	if err = json.Unmarshal(body, &version); err != nil {
		return Version{}, err
	}

	return version, nil
}

// downloadUpdate downloads the data of the given version of Kite and stores the file at targetFilePath
// onProgress may be nil, it's called when a chunk of data was downloaded
func (m *updateManager) downloadUpdate(targetFilePath string, version Version, onProgress func(received, total int64)) error {
	target, err := os.Create(targetFilePath)
	if err != nil {
		return err
	}

	defer target.Close()

	sourceURL, err := url.Parse(version.UpdaterURL)
	if err != nil {
		return err
	}

	resp, err := m.httpClient.Get(sourceURL.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected status code %d for URL %s", resp.StatusCode, sourceURL)
	}

	var source io.Reader = resp.Body
	if onProgress != nil && resp.ContentLength > 0 {
		source = io.TeeReader(source, &httpProgress{
			total:      resp.ContentLength,
			onProgress: onProgress,
		})
	}
	_, err = io.Copy(target, source)
	return err
}
