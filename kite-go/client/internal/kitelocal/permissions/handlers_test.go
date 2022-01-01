package permissions

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/stretchr/testify/assert"
)

func Test_HandleLanguages(t *testing.T) {
	server := requireServer(t)
	defer server.close(t)

	resp, err := doRequest(server, "GET", "/clientapi/languages", nil)
	assert.NoError(t, err)

	var langs []string
	requireJSON(t, resp, http.StatusOK, &langs)
	assert.Equal(t, []string{lang.Python.Name()}, langs)
}

func Test_HandleSupportStatus(t *testing.T) {
	server := requireServer(t)
	defer server.close(t)

	resp, err := doRequest(server, "GET", "/clientapi/support-status", nil)
	assert.NoError(t, err)
	requireStatus(t, resp, http.StatusBadRequest)

	status := &component.SupportStatus{}
	resp, err = doRequest(server, "GET", "/clientapi/support-status?filename=foo.py", nil)
	assert.NoError(t, err)
	requireJSON(t, resp, http.StatusOK, status)
	assert.Equal(t, supportMap[".py"], *status)

	resp, err = doRequest(server, "GET", "/clientapi/support-status?filename=bar.go", nil)
	assert.NoError(t, err)
	requireJSON(t, resp, http.StatusOK, status)
	assert.Equal(t, supportMap[".go"], *status)

	resp, err = doRequest(server, "GET", "/clientapi/support-status?filename=foo.abc", nil)
	assert.NoError(t, err)
	requireJSON(t, resp, http.StatusOK, status)
	assert.Equal(t, component.SupportStatus{}, *status)
}

func Test_Authorized(t *testing.T) {
	server := requireServer(t)
	defer server.close(t)

	// whitelisted file
	resp, err := doGetWithFilenameRequest(server, "/clientapi/permissions/authorized", platformPath(server, "whitelisted", "file.py"))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// an empty file returns a bad request status code
	resp, err = doGetWithFilenameRequest(server, "/clientapi/permissions/authorized", "")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// bad filename
	resp, err = doGetWithFilenameRequest(server, "/clientapi/permissions/authorized", "c::")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// file outside of our root Dir
	resp, err = doGetWithFilenameRequest(server, "/clientapi/permissions/authorized", platformPath(server, "file.py"))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// unsupported file type returns a forbidden status code
	resp, err = doGetWithFilenameRequest(server, "/clientapi/permissions/authorized", platformPath(server, "test.txt"))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// --

// getPaths retrieves a list of paths from the given url using a GET request
// It returns the list of paths and an error

func doGetWithFilenameRequest(server *testServer, urlPath, filename string) (*http.Response, error) {
	base, err := url.Parse(server.http.URL)
	if err != nil {
		return nil, err
	}

	url, err := base.Parse(urlPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("filename", filename)
	req.URL.RawQuery = q.Encode()

	return http.DefaultClient.Do(req)
}

func doRequest(server *testServer, method, path string, obj interface{}) (*http.Response, error) {
	base, err := url.Parse(server.http.URL)
	if err != nil {
		return nil, err
	}

	url, err := base.Parse(path)
	if err != nil {
		return nil, err
	}

	if obj != nil {
		buf := &bytes.Buffer{}
		err = json.NewEncoder(buf).Encode(obj)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest(method, url.String(), buf)
		if err != nil {
			return nil, err
		}
		return http.DefaultClient.Do(req)
	}

	req, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func requireJSON(t *testing.T, resp *http.Response, expectedStatus int, obj interface{}) {
	requireStatus(t, resp, expectedStatus)
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(obj)
	assert.NoError(t, err)
}

func requireStatus(t *testing.T, resp *http.Response, expectedStatus int) {
	assert.Equal(t, expectedStatus, resp.StatusCode)
}

func requireServer(t *testing.T) *testServer {
	mgr := requireManager(t)

	router := mux.NewRouter()
	mgr.RegisterHandlers(router)

	return &testServer{
		mgr:  mgr,
		http: httptest.NewServer(router),
	}
}

type testServer struct {
	mgr    *Manager
	router *mux.Router
	http   *httptest.Server
}

func (s *testServer) close(t *testing.T) {
	s.http.Close()
}

func platformPath(s *testServer, paths ...string) string {
	var all []string
	if runtime.GOOS == "windows" {
		all = append(all, "c:\\directory")
	} else {
		all = append(all, "/directory")
	}

	all = append(all, paths...)
	return filepath.Join(all...)
}
