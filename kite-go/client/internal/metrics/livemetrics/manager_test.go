package livemetrics

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/auth"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/permissions"
	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics"
	complmetrics "github.com/kiteco/kiteco/kite-go/client/internal/metrics/completions"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	plugins "github.com/kiteco/kiteco/kite-go/client/internal/plugins_new"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/community/account"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/proto"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/tfserving"
	"github.com/kiteco/kiteco/kite-golib/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Component(t *testing.T) {
	s, m, err := newTestResources()
	assert.NoError(t, err)
	defer s.Close()

	component.TestImplements(t, m, component.Implements{
		Initializer:      true,
		Terminater:       true,
		Handlers:         true,
		UserAuth:         true,
		ProcessedEventer: true,
		EventResponser:   true,
	})
}

func Test_TrackWhitelistedPythonEvent(t *testing.T) {
	s, m, err := newTestResources()
	assert.NoError(t, err)
	defer s.Close()

	// edit of a whitelisted python file must update the events
	// TrackEvent's implementation calls FromUnix on the path
	filePath, err := localpath.ToUnix(s.GetFilePath("file.py"))
	require.NoError(t, err)

	m.TrackEvent(newEvent(42, "edit", filePath))
	assert.True(t, m.visibility.hasBeenCoding)
	assert.True(t, m.visibility.hasBeenPythonCoding)
}

func Test_TrackWhitelistedJavascriptEvent(t *testing.T) {
	s, m, err := newTestResources()
	require.NoError(t, err)
	defer s.Close()

	// edit of a whitelisted python file must update the
	filePath, err := localpath.ToUnix(s.GetFilePath("file.js"))
	require.NoError(t, err)
	m.TrackEvent(newEvent(42, "edit", filePath))
	assert.True(t, m.visibility.hasBeenCoding)
	assert.False(t, m.visibility.hasBeenPythonCoding)
}

func Test_VisibilityUpdates(t *testing.T) {
	s, m, err := newTestResources()
	assert.NoError(t, err)
	defer s.Close()

	m.TrackEvent(newEvent(42, "edit", s.GetFilePath("file.js")))
	assert.True(t, m.visibility.hasBeenCoding)
	assert.False(t, m.visibility.hasBeenPythonCoding)
}

func Test_EditorStatuses(t *testing.T) {
	s, m, err := newTestResources()
	assert.NoError(t, err)
	defer s.Close()

	// register the handler in Kited
	// returns ATOM as installed&running, VSCode as installed & not running, and VIM as not installed
	s.Kited.Router.HandleFunc("/clientapi/plugins", func(w http.ResponseWriter, r *http.Request) {
		s.Backend.IncrementRequestCount("/clientapi/plugins")
		status := plugins.PluginResponse{
			Plugins: []*plugins.PluginStatus{
				{ID: "atom", Running: true, Editors: []plugins.EditorStatus{{PluginInstalled: true}}},
				{ID: "vscode", Running: false, Editors: []plugins.EditorStatus{{PluginInstalled: true}}},
			},
		}
		body, err := json.Marshal(status)
		if err != nil {
			http.Error(w, "Error marshalling status", http.StatusInternalServerError)
			return
		}
		w.Write(body)
	})

	editorStatus := m.editorStatuses(m.editorStatusesResponse())
	assert.EqualValues(t, 6, len(editorStatus))
	assert.EqualValues(t, true, editorStatus["atom_plugin_installed"])
	assert.EqualValues(t, true, editorStatus["atom_installed"])
	assert.EqualValues(t, true, editorStatus["atom_running"])

	assert.EqualValues(t, true, editorStatus["vscode_plugin_installed"])
	assert.EqualValues(t, true, editorStatus["vscode_installed"])
	assert.EqualValues(t, false, editorStatus["vscode_running"])
}

func Test_SendStatus(t *testing.T) {
	s, m, err := newTestResources()
	assert.NoError(t, err)
	defer s.Close()

	// register plan endpoint in kited, it's not yet a component
	s.Kited.Router.HandleFunc("/clientapi/plan", func(w http.ResponseWriter, r *http.Request) {
		body, _ := json.Marshal(account.PlanResponse{})
		w.Write(body)
	})

	// override telemetry client of metrics manager
	tracker := telemetry.MockClient{}

	//inactive manager (no user)
	require.False(t, m.loggedin)
	m.telemetry = &tracker
	m.sendStatusMetrics()

	require.EqualValues(t, 1, len(tracker.Tracked()))

	// active manager must send telemetry events
	_, err = s.SendLoginRequest("user@example.com", "secret", true)
	require.NoError(t, err)

	// override client after LoggedIn() initialized the telemetry client in manager.go
	// delay this a bit to let the test client process the login and notify components using LoggedIn()
	time.Sleep(1000 * time.Millisecond)
	m.mu.Lock()
	tracker = telemetry.MockClient{}
	m.telemetry = &tracker
	m.mu.Unlock()

	m.sendStatusMetrics()
	require.EqualValues(t, 1, len(tracker.Tracked()))
}

func Test_CounterRequest(t *testing.T) {
	s, m, err := newTestResources()
	require.NoError(t, err)
	defer s.Close()

	assert.EqualValues(t, 0, m.counters["my-counter"])
	resp, err := s.DoKitedPost("/clientapi/metrics/counters", strings.NewReader(`{"name":"my-counter", "value":1}`))
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	assert.EqualValues(t, 1, m.counters["my-counter"], "Expected increased counter value: %s", m.counters)
}

func Test_StatusRequest(t *testing.T) {
	s, _, err := newTestResources()
	require.NoError(t, err)
	defer s.Close()

	resp, err := s.DoKitedGet("/clientapi/metrics")
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
}

func Test_SidebarStatusRequest(t *testing.T) {
	s, m, err := newTestResources()
	require.NoError(t, err)
	defer s.Close()

	status := SidebarStatus{MostRecent: map[string]int{"recent1": 1, "recent10": 10}, Summations: map[string]int{"sum1": 1, "sum10": 10}}
	body, err := json.Marshal(status)
	require.NoError(t, err)

	// update and check values
	resp, err := s.DoKitedPost("/clientapi/metrics/sidebar", bytes.NewReader(body))
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	assert.EqualValues(t, 1, m.sidebarUpdates)
	assert.EqualValues(t, 1, m.sidebarSumStatus.data["sum1"])
	assert.EqualValues(t, 10, m.sidebarSumStatus.data["sum10"])
	assert.EqualValues(t, 1, m.sidebarMostRecentStatus.data["recent1"])
	assert.EqualValues(t, 10, m.sidebarMostRecentStatus.data["recent10"])

	// update and check again
	resp, err = s.DoKitedPost("/clientapi/metrics/sidebar", bytes.NewReader(body))
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	assert.EqualValues(t, 2, m.sidebarUpdates)
}

func newTestResources() (*mockserver.TestClientServer, *Manager, error) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	if err != nil {
		return nil, nil, err
	}

	settingsMgr := settings.NewTestManager()

	permMgr := permissions.NewTestManager(s.Languages...)

	sigMetrics := &metrics.SignaturesMetric{}
	complMetrics := complmetrics.NewMetrics()
	watcherMetrics := &metrics.WatcherMetric{}
	proSelectedMetrics := metrics.NewSmartSelectedMetrics()
	tfservingMetrics := tfserving.GetMetrics()
	metricsMgr := NewManager(sigMetrics, complMetrics, watcherMetrics, proSelectedMetrics, tfservingMetrics)
	authClient := auth.NewTestClient(300 * time.Millisecond)

	// the default setup via auth.SetupComponents isn't possible because metrics is needed to create the auth manager
	// and permissions is needed to create to the metrics manager
	err = s.SetupComponents(authClient, settingsMgr, permMgr, metricsMgr)
	if err != nil {
		return nil, nil, err
	}

	authClient.SetTarget(s.Backend.URL)
	return s, metricsMgr, nil
}

func newEvent(userID int64, action, filename string) *event.Event {
	return &event.Event{
		UserId:        proto.Int64(userID),
		ClientVersion: proto.String("unit test"),
		Action:        proto.String(action),
		Text:          proto.String("content"),
		Filename:      proto.String(filename),
	}
}
