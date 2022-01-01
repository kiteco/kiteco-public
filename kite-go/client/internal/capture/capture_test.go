package capture

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/auth"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testURL = "https://s3.console.aws.amazon.com/s3/object/test"

func handleUpload(w http.ResponseWriter, r *http.Request) {
	data := UploadResponse{URL: testURL}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if err := json.NewEncoder(w).Encode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func Test_UploadCapture(t *testing.T) {
	if !allowUploads {
		return
	}
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	s.SetupWithCustomAuthClient(auth.NewTestClient(35 * time.Second))
	s.AuthClient.SetTarget(s.Backend.URL)
	s.Backend.AddPrefixRequestHandler("/capture", []string{"POST"}, handleUpload)

	server := newTestServer(s.AuthClient, "")
	captureURL := makeTestURL(server.URL, "clientapi/capture")
	resp, err := http.Get(captureURL.String())
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	var ur UploadResponse
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&ur)
	require.NoError(t, err)
	require.Equal(t, testURL, ur.URL)
}

func Test_UploadLogs(t *testing.T) {
	if !allowUploads {
		return
	}
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	s.SetupWithCustomAuthClient(auth.NewTestClient(5 * time.Second))
	s.AuthClient.SetTarget(s.Backend.URL)
	s.Backend.AddPrefixRequestHandler("/logupload", []string{"POST"}, handleUpload)

	logDir, err := ioutil.TempDir("", "kite-logUpload")
	assert.NoError(t, err)
	defer os.RemoveAll(logDir)

	logFile := filepath.Join(logDir, "client.log")
	err = ioutil.WriteFile(logFile, []byte("content"), 0600)
	assert.NoError(t, err)

	server := newTestServer(s.AuthClient, logFile)
	logsURL := makeTestURL(server.URL, "clientapi/logupload")
	resp, err := http.Get(logsURL.String())
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	var ur UploadResponse
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&ur)
	require.NoError(t, err)
	require.Equal(t, testURL, ur.URL)
}

//--

func newTestServer(auth component.AuthClient, logFile string) *httptest.Server {
	m := NewManager()
	uids := userids.NewUserIDs("01", "00")
	uids.SetUser(1, "", true)
	m.Initialize(component.InitializerOptions{
		AuthClient: auth,
		UserIDs:    uids,
		Platform:   &platform.Platform{LogFile: logFile},
	})
	mux := mux.NewRouter()
	m.RegisterHandlers(mux)
	server := httptest.NewServer(mux)
	return server
}

func makeTestURL(base, endpoint string) *url.URL {
	baseURL, err := url.Parse(base)
	if err != nil {
		log.Fatal(err)
	}

	endpointURL, err := baseURL.Parse(endpoint)
	if err != nil {
		log.Fatal(err)
	}

	return endpointURL
}
