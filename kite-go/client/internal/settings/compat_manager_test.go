package settings

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSettings map[string]string

func defaultTestSettings() testSettings {
	return map[string]string{
		ServerKey:                      "https://test-2.kite.com",
		StatusIconKey:                  "true",
		CompletionsDisabledKey:         "false",
		MetricsDisabledKey:             "false",
		HasDoneOnboardingKey:           "false",
		HaveShownWelcome:               "false",
		InstallTimeKey:                 time.Time{}.Format(time.RFC3339),
		AutosearchEnabledKey:           "true",
		AutoInstallPluginsEnabledKey:   "true",
		NotifyUninstalledPluginsKey:    "true",
		proxyModeKey:                   "environment",
		proxyURLKey:                    "",
		TFThreadsKey:                   "1",
		MaxFileSizeKey:                 "1024",
		PredictiveNavMaxFilesKey:       "100000",
		ProLaunchNotificationDismissed: "false",
		ShowCompletionsCTA:             "true",
		ShowCompletionsCTANotif:        "true",
		CompletionsCTALastShown:        time.Time{}.Format(time.RFC3339),
		RCDisabledCompletionsCTA:       "false",
		RCDisabledLexicalPython:        "false",
		PaywallLastUpdated:             time.Time{}.Format(time.RFC3339),
		PaywallCompletionsLimit:        "3",
		PaywallCompletionsRemaining:    "3",
		ShowPaywallExhaustedNotif:      "true",
		KiteServer:                     "",
		ChooseEngineKey:                "false",
		SelectedEngineKey:              "local",
		AllFeaturesPro:                 "false",
	}
}

func defaultSettings() testSettings {
	return map[string]string{
		ServerKey:                      *specs[ServerKey].defaultValue,
		StatusIconKey:                  *specs[StatusIconKey].defaultValue,
		CompletionsDisabledKey:         *specs[CompletionsDisabledKey].defaultValue,
		MetricsDisabledKey:             *specs[MetricsDisabledKey].defaultValue,
		HasDoneOnboardingKey:           *specs[HasDoneOnboardingKey].defaultValue,
		HaveShownWelcome:               *specs[HaveShownWelcome].defaultValue,
		InstallTimeKey:                 *specs[InstallTimeKey].defaultValue,
		AutosearchEnabledKey:           *specs[AutosearchEnabledKey].defaultValue,
		AutoInstallPluginsEnabledKey:   *specs[AutoInstallPluginsEnabledKey].defaultValue,
		NotifyUninstalledPluginsKey:    *specs[NotifyUninstalledPluginsKey].defaultValue,
		proxyModeKey:                   *specs[proxyModeKey].defaultValue,
		proxyURLKey:                    *specs[proxyURLKey].defaultValue,
		TFThreadsKey:                   *specs[TFThreadsKey].defaultValue,
		MaxFileSizeKey:                 *specs[MaxFileSizeKey].defaultValue,
		PredictiveNavMaxFilesKey:       *specs[PredictiveNavMaxFilesKey].defaultValue,
		ProLaunchNotificationDismissed: *specs[ProLaunchNotificationDismissed].defaultValue,
		ShowCompletionsCTA:             *specs[ShowCompletionsCTA].defaultValue,
		ShowCompletionsCTANotif:        *specs[ShowCompletionsCTANotif].defaultValue,
		CompletionsCTALastShown:        *specs[CompletionsCTALastShown].defaultValue,
		RCDisabledCompletionsCTA:       *specs[RCDisabledCompletionsCTA].defaultValue,
		RCDisabledLexicalPython:        *specs[RCDisabledLexicalPython].defaultValue,
		PaywallLastUpdated:             *specs[PaywallLastUpdated].defaultValue,
		PaywallCompletionsLimit:        *specs[PaywallCompletionsLimit].defaultValue,
		PaywallCompletionsRemaining:    *specs[PaywallCompletionsRemaining].defaultValue,
		ShowPaywallExhaustedNotif:      *specs[ShowPaywallExhaustedNotif].defaultValue,
		KiteServer:                     *specs[KiteServer].defaultValue,
		ChooseEngineKey:                *specs[ChooseEngineKey].defaultValue,
		SelectedEngineKey:              *specs[SelectedEngineKey].defaultValue,
		AllFeaturesPro:                 *specs[AllFeaturesPro].defaultValue,
	}
}

func requireManager(t *testing.T, init testSettings) *Manager {
	path := filepath.Join(os.TempDir(), "settings.json")
	if init != nil {
		file, err := os.Create(path)
		require.NoError(t, err)
		defer file.Close()
		require.NoError(t, json.NewEncoder(file).Encode(init))
	}

	return NewManager(path)
}

func requireCleanup(t *testing.T, mgr *Manager) {
	require.NoError(t, os.Remove(mgr.filepath))
}

func assertSettings(t *testing.T, expected, actual testSettings) {
	require.NotNil(t, expected)
	require.NotNil(t, actual)

	for ek, ev := range expected {
		av, ok := actual[ek]
		if !ok {
			t.Errorf("expected to find key %s with value %s. Actual value: %s\n", ek, ev, av)
			continue
		}

		assert.Equal(t, ev, av, "for key %s expected %s but got %s", ek, ev, av)
	}

	for ak, av := range actual {
		_, ok := expected[ak]
		if !ok {
			t.Errorf("got unexpected key %s with value %s\n", ak, av)
		}
	}
}

type compatTestServer struct {
	mgr *Manager
	ts  *httptest.Server
}

func requireCompatTestServer(t *testing.T, init testSettings) *compatTestServer {
	mgr := requireManager(t, init)
	mux := mux.NewRouter()
	mux.HandleFunc("/clientapi/settings/{key}", mgr.handleSet).Methods("PUT", "POST")
	mux.HandleFunc("/clientapi/settings/{key}", mgr.handleGet).Methods("GET")
	mux.HandleFunc("/clientapi/settings/{key}", mgr.handleDelete).Methods("DELETE")
	return &compatTestServer{
		mgr: mgr,
		ts:  httptest.NewServer(mux),
	}
}

func requireCompatCleanupTestServer(t *testing.T, ts *compatTestServer) {
	requireCleanup(t, ts.mgr)
	ts.ts.Close()
}

func requireResponse(t *testing.T, ts *compatTestServer, method, key, body string) *http.Response {
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

func assertPut(t *testing.T, ts *compatTestServer, k, v string) {
	// copy settings to check update
	expected := make(testSettings)
	ts.mgr.settings.Range(func(key, value interface{}) bool {
		expected[key.(string)] = value.(string)
		return true
	})
	expected[k] = v

	resp := requireResponse(t, ts, "PUT", k, v)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assertSettings(t, expected, ts.mgr.settingsCopy())
}

func assertGet(t *testing.T, ts *compatTestServer, k, v string) {
	resp := requireResponse(t, ts, "GET", k, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	buf, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, v, string(buf))
}

func assertDelete(t *testing.T, ts *compatTestServer, k string) {
	// copy settings to compare
	expected := ts.mgr.settingsCopy()
	delete(expected, k)

	resp := requireResponse(t, ts, "DELETE", k, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assertSettings(t, expected, ts.mgr.settingsCopy())
}

func TestEmptySettings(t *testing.T) {
	mgr := requireManager(t, nil)
	defer requireCleanup(t, mgr)

	assertSettings(t, defaultSettings(), mgr.settingsCopy())
}

func TestCorruptedSettings(t *testing.T) {
	path := filepath.Join(os.TempDir(), "settings.json")
	mgr := NewManager(path)
	defer requireCleanup(t, mgr)
	assertSettings(t, defaultSettings(), mgr.settingsCopy())
}

func TestRevert(t *testing.T) {
	ts := requireCompatTestServer(t, defaultTestSettings())
	defer requireCompatCleanupTestServer(t, ts)

	// change file perms to read only
	if runtime.GOOS == "windows" {
		require.NoError(t, os.Chmod(ts.mgr.filepath, 1))
	} else {
		require.NoError(t, os.Chmod(ts.mgr.filepath, 0444))
	}

	// try to update settings, should fail
	resp := requireResponse(t, ts, "PUT", ServerKey, "https://test-4.kite.com")
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// mgr settings should be the same as original
	assertSettings(t, defaultTestSettings(), ts.mgr.settingsCopy())

	// revert perm changes before cleanup
	if runtime.GOOS == "windows" {
		require.NoError(t, os.Chmod(ts.mgr.filepath, 128))
	} else {
		require.NoError(t, os.Chmod(ts.mgr.filepath, 777))
	}
}

func TestServer(t *testing.T) {
	ts := requireCompatTestServer(t, defaultTestSettings())
	defer requireCompatCleanupTestServer(t, ts)

	var observed string
	ts.mgr.AddNotificationTargetKey(ServerKey, func(s string) {
		observed = s
	})

	url := "https://test-4.kite.com"
	assertPut(t, ts, ServerKey, url)
	assertGet(t, ts, ServerKey, url)

	assert.Equal(t, url, observed)
}

func TestServerInvalid(t *testing.T) {
	ts := requireCompatTestServer(t, defaultTestSettings())
	defer requireCompatCleanupTestServer(t, ts)

	var observed bool
	var observedVal string
	ts.mgr.AddNotificationTargetKey(ServerKey, func(s string) {
		observed = true
		observedVal = s
	})

	// make sure error is triggered when trying to set to invalid aboslute url
	resp := requireResponse(t, ts, "PUT", ServerKey, "/http://test.kite.com")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// make sure error is triggered when trying to delete setting
	resp = requireResponse(t, ts, "DELETE", ServerKey, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// make sure observer is not triggered for an invalid value
	if observed {
		t.Errorf("observe server called with invalid server: %s", observedVal)
	}
}

func TestMetricsDisabled(t *testing.T) {
	ts := requireCompatTestServer(t, defaultTestSettings())
	defer requireCompatCleanupTestServer(t, ts)

	var observed bool
	ts.mgr.AddNotificationTarget(&stringKeyListener{watchedKey: MetricsDisabledKey, callback: func(value string) {
		observed, _ = ts.mgr.GetBool(MetricsDisabledKey)
	}})

	assertPut(t, ts, MetricsDisabledKey, "true")
	assertGet(t, ts, MetricsDisabledKey, "true")

	assert.Equal(t, true, observed)
}

func TestMetricsDisabledInvalid(t *testing.T) {
	ts := requireCompatTestServer(t, defaultTestSettings())
	defer requireCompatCleanupTestServer(t, ts)

	var observedVal, observed bool
	ts.mgr.AddNotificationTarget(&stringKeyListener{watchedKey: MetricsDisabledKey, callback: func(value string) {
		observed = true
		observedVal, _ = ts.mgr.GetBool(MetricsDisabledKey)
	}})

	// make sure we get an error when trying to set an invalid value
	resp := requireResponse(t, ts, "PUT", MetricsDisabledKey, "FOO")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// make sure we get an error when trying to delete setting
	resp = requireResponse(t, ts, "DELETE", MetricsDisabledKey, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// make sure that observer is not triggered for invalid value
	if observed {
		t.Errorf("observe status icon called with invalid value, got: %v", observedVal)
	}
}

func TestShowStatusIcon(t *testing.T) {
	ts := requireCompatTestServer(t, defaultTestSettings())
	defer requireCompatCleanupTestServer(t, ts)

	var observed bool
	ts.mgr.AddNotificationTarget(&stringKeyListener{watchedKey: StatusIconKey, callback: func(value string) {
		observed, _ = ts.mgr.GetBool(StatusIconKey)
	}})

	assertPut(t, ts, StatusIconKey, "false")
	assertGet(t, ts, StatusIconKey, "false")

	assert.Equal(t, false, observed)
}

func TestShowStatusIconInvalid(t *testing.T) {
	ts := requireCompatTestServer(t, defaultTestSettings())
	defer requireCompatCleanupTestServer(t, ts)

	var observedVal, observed bool
	ts.mgr.AddNotificationTarget(&stringKeyListener{watchedKey: StatusIconKey, callback: func(value string) {
		observed = true
		observedVal, _ = ts.mgr.GetBool(StatusIconKey)
	}})

	// make sure we get an error when trying to set an invalid value
	resp := requireResponse(t, ts, "PUT", StatusIconKey, "FOO")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// make sure we get an error when trying to delete setting
	resp = requireResponse(t, ts, "DELETE", StatusIconKey, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// make sure that observer is not triggered for invalid value
	if observed {
		t.Errorf("observe status icon called with invalid value, got: %v", observedVal)
	}
}

func TestHasDoneOnboarding(t *testing.T) {
	ts := requireCompatTestServer(t, defaultTestSettings())
	defer requireCompatCleanupTestServer(t, ts)

	var observed bool
	ts.mgr.AddNotificationTarget(&stringKeyListener{watchedKey: HasDoneOnboardingKey, callback: func(value string) {
		observed, _ = ts.mgr.GetBool(HasDoneOnboardingKey)
	}})

	assertPut(t, ts, HasDoneOnboardingKey, "true")
	assertGet(t, ts, HasDoneOnboardingKey, "true")

	assert.Equal(t, true, observed)
}

func TestHasDoneOnboardingInvalid(t *testing.T) {
	ts := requireCompatTestServer(t, defaultTestSettings())
	defer requireCompatCleanupTestServer(t, ts)

	var observedVal, observed bool
	ts.mgr.AddNotificationTarget(&stringKeyListener{watchedKey: HasDoneOnboardingKey, callback: func(value string) {
		observed = true
		observedVal, _ = ts.mgr.GetBool(HasDoneOnboardingKey)
	}})

	// make sure we get an error when trying to set an invalid value
	resp := requireResponse(t, ts, "PUT", HasDoneOnboardingKey, "FOO")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// make sure we get an error when trying to delete setting
	resp = requireResponse(t, ts, "DELETE", HasDoneOnboardingKey, "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// make sure that observer is not triggered for invalid value
	if observed {
		t.Errorf("observe has done onboarding called with invalid value, got: %v", observedVal)
	}
}

func TestPutRandom(t *testing.T) {
	ts := requireCompatTestServer(t, defaultTestSettings())
	defer requireCompatCleanupTestServer(t, ts)

	k := "random"
	v := `{foo:"bar", car:1}`
	assertPut(t, ts, k, v)
	assertGet(t, ts, k, v)
	assertDelete(t, ts, k)
}

func TestServerMethod(t *testing.T) {
	init := defaultTestSettings()
	server := init[ServerKey]

	mgr := requireManager(t, init)
	defer requireCleanup(t, mgr)

	var observed string
	mgr.AddNotificationTargetKey(ServerKey, func(s string) {
		observed = s
	})

	// check that we fetch properly
	assert.Equal(t, server, mgr.Server())

	// check that we update properly
	server = "https://test-4.kite.com"
	mgr.Set(ServerKey, server)
	init[ServerKey] = server

	// make sure updated in settings
	assertSettings(t, init, mgr.settingsCopy())

	// make sure updated on disk
	err := mgr.load()
	require.NoError(t, err)

	assertSettings(t, init, mgr.settingsCopy())

	// make sure observer is triggered
	assert.Equal(t, server, observed)
}

func (m *Manager) settingsCopy() testSettings {
	expected := make(testSettings)
	for k, v := range specs {
		if v.defaultValue != nil {
			expected[k] = *v.defaultValue
		}
	}

	m.settings.Range(func(key, value interface{}) bool {
		expected[key.(string)] = value.(string)
		return true
	})
	return expected
}
