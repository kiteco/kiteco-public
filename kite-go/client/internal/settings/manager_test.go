package settings

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Basics(t *testing.T) {
	ts := requireTestServer(t)
	defer requireCleanupTestServer(t, ts)

	assert.Equal(t, "settings", ts.mgr.Name())

	type testCase struct {
		method string
		key    string
		value  string
		status int
	}

	cases := []testCase{
		//since the key is part of the path it will result in a 404 instead of triggering the result for an empty key
		{"GET", "", "", http.StatusNotFound},
		{"GET", "somekey", "", http.StatusNotFound},
		{"PUT", "somekey", "somevalue", http.StatusOK},
		{"GET", "somekey", "somevalue", http.StatusOK},
		{"DELETE", "somekey", "", http.StatusOK},
		{"GET", "somekey", "", http.StatusNotFound},
		{"DELETE", "somekey2", "", http.StatusOK},
	}

	for _, c := range cases {
		resp := requireRequest(t, ts, c.method, c.key, c.value)
		switch c.method {
		case "GET":
			require.Equal(t, c.status, resp.StatusCode, "unexpected status code for %s %s", c.method, c.key)
			if resp.StatusCode == http.StatusOK {
				requireBody(t, resp, c.value)
			}
		case "PUT", "POST":
			require.Equal(t, c.status, resp.StatusCode, "unexpected status code for %s %s with value %s", c.method, c.key, c.value)
		default:
			require.Equal(t, c.status, resp.StatusCode, "unexpected status code for %s %s", c.method, c.key)
		}
	}
}

func Test_GetInt(t *testing.T) {
	ts := requireTestServer(t)
	defer requireCleanupTestServer(t, ts)

	_, err := ts.mgr.GetInt(TFThreadsKey)
	require.NoError(t, err)

	val2, err2 := ts.mgr.GetInt("no_val")
	require.Error(t, err2)

	err = ts.mgr.Set("no_val", "4")
	require.NoError(t, err)

	val2, err2 = ts.mgr.GetInt("no_val")
	require.NoError(t, err2)
	require.Equal(t, 4, val2)
}

func Test_Obj(t *testing.T) {
	ts := requireTestServer(t)
	defer requireCleanupTestServer(t, ts)

	values := make(map[string]string)
	values["a"] = "A"
	values["b"] = "B"

	nv := make(map[string]string)
	err := ts.mgr.GetObj("values", &nv)
	require.Error(t, err)

	err = ts.mgr.SetObj("values", values)
	require.NoError(t, err)

	err = ts.mgr.GetObj("values", &nv)
	require.NoError(t, err)
	require.EqualValues(t, values, nv)

	values["b"] = "BB"
	err = ts.mgr.SetObj("values", values)
	require.NoError(t, err)

	err = ts.mgr.GetObj("values", &nv)
	require.NoError(t, err)
	require.EqualValues(t, values, nv)
}

func Test_KeySpec(t *testing.T) {
	ts := requireTestServer(t)
	defer requireCleanupTestServer(t, ts)

	type testCase struct {
		method string
		key    string
		value  string
		status int
	}

	cases := []testCase{
		{"GET", ServerKey, *specs[ServerKey].defaultValue, http.StatusOK},
		{"GET", StatusIconKey, *specs[StatusIconKey].defaultValue, http.StatusOK},
		{"GET", MetricsDisabledKey, *specs[MetricsDisabledKey].defaultValue, http.StatusOK},
		{"GET", HasDoneOnboardingKey, *specs[HasDoneOnboardingKey].defaultValue, http.StatusOK},
		{"DELETE", ServerKey, "", http.StatusInternalServerError},
		{"DELETE", StatusIconKey, "", http.StatusInternalServerError},
		{"DELETE", MetricsDisabledKey, "", http.StatusInternalServerError},
		{"DELETE", HasDoneOnboardingKey, "", http.StatusInternalServerError},
		{"PUT", ServerKey, "http+:bad/url:2020", http.StatusInternalServerError},
		{"PUT", StatusIconKey, "notabool", http.StatusInternalServerError},
		{"PUT", MetricsDisabledKey, "notabool", http.StatusInternalServerError},
		{"PUT", HasDoneOnboardingKey, "notabool", http.StatusInternalServerError},
		{"GET", ServerKey, *specs[ServerKey].defaultValue, http.StatusOK},
		{"GET", StatusIconKey, *specs[StatusIconKey].defaultValue, http.StatusOK},
		{"GET", MetricsDisabledKey, *specs[MetricsDisabledKey].defaultValue, http.StatusOK},
		{"GET", HasDoneOnboardingKey, *specs[HasDoneOnboardingKey].defaultValue, http.StatusOK},
		{"PUT", ServerKey, "http://www.example.com/", http.StatusOK},
		{"PUT", StatusIconKey, "true", http.StatusOK},
		{"PUT", MetricsDisabledKey, "true", http.StatusOK},
		{"PUT", HasDoneOnboardingKey, "true", http.StatusOK},
		{"GET", ServerKey, "http://www.example.com/", http.StatusOK},
		{"GET", StatusIconKey, "true", http.StatusOK},
		{"GET", MetricsDisabledKey, "true", http.StatusOK},
		{"GET", HasDoneOnboardingKey, "true", http.StatusOK},
	}

	for _, c := range cases {
		resp := requireRequest(t, ts, c.method, c.key, c.value)
		switch c.method {
		case "GET":
			require.Equal(t, c.status, resp.StatusCode, "unexpected status code for %s %s", c.method, c.key)
			if resp.StatusCode == http.StatusOK {
				requireBody(t, resp, c.value)
			}
		case "PUT", "POST":
			require.Equal(t, c.status, resp.StatusCode, "unexpected status code for %s %s with value %s", c.method, c.key, c.value)
		default:
			require.Equal(t, c.status, resp.StatusCode, "unexpected status code for %s %s", c.method, c.key)
		}
	}
}

func Test_GetEmptyKey(t *testing.T) {
	ts := requireTestServer(t)
	defer requireCleanupTestServer(t, ts)

	getRequest := httptest.NewRequest("GET", "/clientapi/settings/", strings.NewReader(""))
	resp := httptest.NewRecorder()
	ts.mgr.handleGet(resp, getRequest)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func Test_DeleteEmptyKey(t *testing.T) {
	ts := requireTestServer(t)
	defer requireCleanupTestServer(t, ts)

	getRequest := httptest.NewRequest("DELETE", "/clientapi/settings/", strings.NewReader(""))
	resp := httptest.NewRecorder()
	ts.mgr.handleDelete(resp, getRequest)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func Test_SetEmptyKey(t *testing.T) {
	ts := requireTestServer(t)
	defer requireCleanupTestServer(t, ts)

	getRequest := httptest.NewRequest("POST", "/clientapi/settings/", strings.NewReader(""))
	resp := httptest.NewRecorder()
	ts.mgr.handleSet(resp, getRequest)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func Test_SaveEmptyFile(t *testing.T) {
	m := NewTestManager()
	err := m.save()
	assert.NoError(t, err, "No error expected for empty target path")
}

func Test_SaveInvalidFile(t *testing.T) {
	m := NewManager("/BAD-target-path-which-does-not-exist/settings.json")
	err := m.save()
	assert.Error(t, err)
}

func Test_SaveLoad(t *testing.T) {
	path := filepath.Join(os.TempDir(), "kite-settings-temp.json")
	defer os.Remove(path)

	m := NewManager(path)
	err := m.load()
	assert.NoError(t, err, "No error is returned if the settings doesn't exist")

	m.Set("my-key", "my-value")
	err = m.save()
	assert.NoError(t, err)

	err = m.load()
	assert.NoError(t, err)
	value, exists := m.Get("my-key")
	assert.True(t, exists)
	assert.Equal(t, "my-value", value)

	_, exists = m.Get("invalid-key")
	assert.False(t, exists)
}

// --

type testServer struct {
	mgr *Manager
	ts  *httptest.Server
}

func requireTestServer(t *testing.T) *testServer {
	m := mux.NewRouter()

	mgr := NewTestManager()
	mgr.RegisterHandlers(m)

	return &testServer{
		mgr: mgr,
		ts:  httptest.NewServer(m),
	}
}

func requireCleanupTestServer(t *testing.T, ts *testServer) {
	ts.ts.Close()
}

func requireRequest(t *testing.T, ts *testServer, method, key, body string) *http.Response {
	base, err := url.Parse(ts.ts.URL)
	require.NoError(t, err)

	endpointURL, err := base.Parse("/clientapi/settings/" + key)
	require.NoError(t, err)

	req, err := http.NewRequest(method, endpointURL.String(), bytes.NewReader([]byte(body)))
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Body)

	return resp
}

func requireBody(t *testing.T, resp *http.Response, value string) {
	buf, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, value, string(buf))
}
