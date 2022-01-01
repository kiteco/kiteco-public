package main

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
)

// errorInfo counts download and validation errors
type errorInfo struct {
	DownloadErrors   int32 `json:"download_errors"`
	ValidationErrors int32 `json:"validation_errors"`
	RollbarSent      bool  `json:"rollbar_sent"`
}

// total returns the sum of download and validaton errors
func (d errorInfo) total() int32 {
	return d.DownloadErrors + d.ValidationErrors
}

// errorTracker tracks download errors and persists this in configFile on disk
type errorTracker struct {
	configFile string
	downloads  map[string]*errorInfo
}

func newDownloadTracker(baseDir string) *errorTracker {
	tracker := &errorTracker{
		configFile: filepath.Join(baseDir, "kite-installer.json"),
		downloads:  make(map[string]*errorInfo),
	}
	_ = tracker.load()
	return tracker
}

func (t *errorTracker) load() error {
	data, err := ioutil.ReadFile(t.configFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &t.downloads)
}

func (t *errorTracker) save() error {
	data, err := json.Marshal(t.downloads)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(t.configFile, data, 0600)
}

func (t *errorTracker) get(version string) *errorInfo {
	inf := t.downloads[version]
	if inf == nil {
		inf = &errorInfo{}
		t.downloads[version] = inf
	}
	return inf
}

func (t *errorTracker) addDownloadError(version string) {
	v, ok := t.downloads[version]
	if !ok {
		v = &errorInfo{}
	}
	v.DownloadErrors++
	t.downloads[version] = v
}

func (t *errorTracker) addValidationError(version string) {
	v, ok := t.downloads[version]
	if !ok {
		v = &errorInfo{}
	}
	v.ValidationErrors++
	t.downloads[version] = v
}
