package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTracker(t *testing.T) {
	dir, err := ioutil.TempDir("", "kite-installer")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	d := newDownloadTracker(dir)
	err = d.load()
	require.True(t, os.IsNotExist(err))

	d.addDownloadError("2.20190501.1")
	d.addDownloadError("2.20190501.1")
	d.addDownloadError("2.20190501.1")
	d.addValidationError("2.20190501.1")

	d.addDownloadError("2.20190509.1")
	d.addValidationError("2.20190509.1")

	d.addDownloadError("2.20190509.2")

	assert.EqualValues(t, 3, d.get("2.20190501.1").DownloadErrors)
	assert.EqualValues(t, 1, d.get("2.20190501.1").ValidationErrors)

	assert.EqualValues(t, 1, d.get("2.20190509.1").DownloadErrors)
	assert.EqualValues(t, 1, d.get("2.20190509.1").ValidationErrors)

	assert.EqualValues(t, 1, d.get("2.20190509.2").DownloadErrors)
	assert.EqualValues(t, 0, d.get("2.20190509.2").ValidationErrors)

	assert.EqualValues(t, 0, d.get("not-released").DownloadErrors)
	assert.EqualValues(t, 0, d.get("not-released").ValidationErrors)

	err = d.save()
	require.NoError(t, err)

	// load in another instance
	d2 := newDownloadTracker(dir)
	err = d2.load()
	require.NoError(t, err)
	assert.EqualValues(t, 3, d2.get("2.20190501.1").DownloadErrors)
	assert.EqualValues(t, 1, d2.get("2.20190501.1").ValidationErrors)

	assert.EqualValues(t, 1, d2.get("2.20190509.1").DownloadErrors)
	assert.EqualValues(t, 1, d2.get("2.20190509.1").ValidationErrors)

	assert.EqualValues(t, 1, d2.get("2.20190509.2").DownloadErrors)
	assert.EqualValues(t, 0, d2.get("2.20190509.2").ValidationErrors)

	assert.EqualValues(t, 0, d2.get("not-released").DownloadErrors)
	assert.EqualValues(t, 0, d2.get("not-released").ValidationErrors)
}

func Test_AddError(t *testing.T) {
	dir, err := ioutil.TempDir("", "kite-installer")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	tracker := newDownloadTracker(dir)
	tracker.addDownloadError("1.0.0")
	tracker.addValidationError("1.0.0")
	tracker.addValidationError("1.0.0")
	tracker.addDownloadError("1.0.0")
	require.EqualValues(t, 2, tracker.downloads["1.0.0"].DownloadErrors)
	require.EqualValues(t, 2, tracker.downloads["1.0.0"].ValidationErrors)

	tracker.addValidationError("2.0.0")
	tracker.addDownloadError("2.0.0")
	tracker.addDownloadError("2.0.0")
	tracker.addValidationError("2.0.0")
	require.EqualValues(t, 2, tracker.downloads["2.0.0"].DownloadErrors)
	require.EqualValues(t, 2, tracker.downloads["2.0.0"].ValidationErrors)
}
