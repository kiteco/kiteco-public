package client

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/auth"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Upload(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	err = auth.SetupWithAuthDefaults(s)
	assert.NoError(t, err)

	logDir, err := ioutil.TempDir("", "kite-logUpload")
	assert.NoError(t, err)
	defer os.RemoveAll(logDir)

	//setup logfile, only files *.bak are uploaded
	logFile1 := filepath.Join(logDir, fmt.Sprintf("first.log.%s", PreviousLogsSuffix))
	err = ioutil.WriteFile(logFile1, []byte("content"), 0600)
	assert.NoError(t, err)

	//mock handler for /clientlogs which expects that logFile1 is uploaded
	s.Backend.AddPrefixRequestHandler("/clientlogs", []string{"POST"}, func(w http.ResponseWriter, r *http.Request) {
		machineid := r.URL.Query().Get("machineid")
		if machineid != "dummy-machine" {
			http.Error(w, "Upload received invalid machine id", http.StatusBadRequest)
			return
		}

		installid := r.URL.Query().Get("installid")
		if installid != "dummy-install" {
			http.Error(w, "Upload received invalid install id", http.StatusBadRequest)
			return
		}

		filename := r.URL.Query().Get("filename")
		if filename != filepath.Base(logFile1) {
			http.Error(w, "Upload received invalid filename", http.StatusBadRequest)
			return
		}

		gzipReader, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "Error reading gzip request body", http.StatusBadRequest)
			return
		}

		body, err := ioutil.ReadAll(gzipReader)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}
		r.Body.Close()

		if string(body) != "content" {
			http.Error(w, "Upload received invalid content", http.StatusBadRequest)
			return
		}

		s.Backend.IncrementRequestCount("/clientlogs")
		w.WriteHeader(http.StatusOK)
	})

	err = uploadLogs(s.AuthClient, logDir, "dummy-machine", "dummy-install")
	assert.NoError(t, err)
	assert.EqualValues(t, 1, s.Backend.GetRequestCount("/clientlogs"))

	//uploadLogs must remove the uploaded file
	if _, err := os.Stat(logFile1); err == nil {
		assert.Fail(t, "uploaded file still exists: ", logFile1)
	}
}

func Test_NoUploadInEmptyDir(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	err = auth.SetupWithAuthDefaults(s)
	assert.NoError(t, err)

	s.Backend.AddPrefixRequestHandler("/clientlogs", []string{"POST"}, func(w http.ResponseWriter, r *http.Request) {
		s.Backend.IncrementRequestCount("/clientlogs")
		w.WriteHeader(http.StatusBadRequest)
	})

	tmpDir, err := ioutil.TempDir("", "kite-logUpload")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = uploadLogs(s.AuthClient, tmpDir, "dummy-machine", "dummy-install")
	assert.NoError(t, err)
	assert.EqualValues(t, 0, s.Backend.GetRequestCount("/clientlogs"))
}
