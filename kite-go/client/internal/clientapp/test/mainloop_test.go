package test

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/clientapp"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/stretchr/testify/require"
)

// tests the clients main loop
func Test_MainLoop(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("skipping tests because it's too slow on Travis for Windows & macOS")
	}

	counter := component.NewCountingComponent("mock-counter")

	// project without login
	// a failing component must not break the main loop
	p, err := clientapp.StartDefaultTestEnvironment(false, nil, counter)
	require.NoError(t, err)
	defer p.Close()

	// Allow time for the main loop to start. If something went wrong, the checks below will trigger a failure
	require.NoError(t, p.WaitForReady(20*time.Second))

	require.EqualValues(t, 1, counter.GetInitCount(), "component must be initialized")
	require.EqualValues(t, p.Backend.URL, p.Kited.AuthClient.Target(), "kited's loop must update the target URL")
	require.EqualValues(t, 1, p.MockUpdater.GetCheckedCount(), "kited's loop must check for updates. Requests: \n%s", p.Backend.CountDebugString())

	assertNoUser(t, p)
	require.EqualValues(t, 0, counter.GetLoggedInCount(), "no login must be send at startup if auth can't be restored")

	require.NotNil(t, clienttelemetry.GetUserIDs(), "expected non-nil userids wrapper before login")
	require.EqualValues(t, 0, clienttelemetry.GetUserIDs().UserID(), "expected user id 0 before login")
	require.EqualValues(t, clienttelemetry.GetUserIDs().InstallID(), clienttelemetry.GetUserIDs().MetricsID(), "expected metricsid to be the installid")
	require.True(t, len(clienttelemetry.GetUserIDs().MetricsID()) > 10, "expected metrics id to be set")

	// make sure that events before login do not jam the event channel
	// NOTE: we use 50 events here because this is the size of the event response buffer
	// in kitelocal.Manager. Keeping this to 50 events ensures we never drop responses and
	// enter a situation where the counters below can never hit 50.
	sendEditorEvents(p, p.Files[0], 50)

	// allow processing in the editor component
	waitFor(t, func() bool { return 50 == counter.GetPluginEventCount() })
	require.EqualValues(t, 50, counter.GetPluginEventCount(), "all editor events must be processed")

	waitFor(t, func() bool { return 50 == counter.GetProcessedEventsCount() })
	require.EqualValues(t, 50, counter.GetProcessedEventsCount(), "all events must be processed")

	waitFor(t, func() bool { return 50 == counter.GetEventResponseCount() })
	require.EqualValues(t, 50, counter.GetEventResponseCount(), "all backend responses must be allowed even if no user is logged in")

	// make sure that the main loop is not confused by superfluous logouts
	p.KitedClient.SendLogoutRequest(true)
	p.KitedClient.SendLogoutRequest(true)
	assertNoUser(t, p)
	require.EqualValues(t, 0, counter.GetLoggedOutCount(), "no logout must be send if no user was logged in when the logout occurred")

	// login to proceed in main loop
	p.Backend.AddValidUser(community.User{Email: "user@example.com", ID: 42, EmailVerified: true}, "secret")
	_, err = p.KitedClient.SendLoginRequest("user@example.com", "secret", true)
	require.NoError(t, err)
	assertUserAndPlan(t, p)
	waitFor(t, func() bool {
		return counter.GetLoggedInCount() == 1
	})

	require.NotNil(t, clienttelemetry.GetUserIDs())
	require.EqualValues(t, "42", clienttelemetry.GetUserIDs().MetricsID(), "expected valid user id for metrics")
	require.NotEmpty(t, clienttelemetry.GetUserIDs().MachineID(), "expected valid machine id")

	// event processing, reset counts for easier testing
	counter.Reset()

	sendEditorEvents(p, p.Files[0], 50)
	// allow processing in the editor component
	waitFor(t, func() bool { return 50 == counter.GetPluginEventCount() })
	require.EqualValues(t, 50, counter.GetPluginEventCount(), "all editor events must be processed")

	waitFor(t, func() bool { return 50 == counter.GetProcessedEventsCount() })
	require.EqualValues(t, 50, counter.GetProcessedEventsCount(), "all responses from the backend must be processed")

	waitFor(t, func() bool { return 50 == counter.GetEventResponseCount() })
	require.EqualValues(t, 50, counter.GetEventResponseCount(), "all backend responses must be processed if a user is logged in")

	// make sure that additional login requests to not mix up the loop
	for i := 0; i < 50; i++ {
		p.KitedClient.SendLoginRequest("user@example.com", "secret", false)
		p.KitedClient.SendLoginRequest("pro@example.com", "secret", false)
		p.KitedClient.SendLoginRequest("unknown@example.com", "secret", false)
	}

	// logout and make sure that user, plan, and event handling is good
	_, err = p.KitedClient.SendLogoutRequest(true)
	require.NoError(t, err)

	assertNoUser(t, p)
	time.Sleep(500 * time.Millisecond)
	require.EqualValues(t, 0, counter.GetInitCount(), "components must not be initialized again")
}

// test that panicing components don't break the main loop and component workflow
func TestPanicInComponents(t *testing.T) {
	failingComponent := &panicComponent{
		initPanic: false, // init must not fail, we need a working client

		// all other component methods are marked to panic
		eventResponsePanic:  true,
		goTickPanic:         true,
		handlersPanic:       true,
		loggedInPanic:       true,
		loggedOutPanic:      true,
		pluginEventPanic:    true,
		processedEventPanic: true,
		settingsPanic:       true,
		terminatePanic:      true,
	}

	// project without login
	// a failing component must not break the main loop
	p, err := clientapp.StartEmptyTestEnvironment(failingComponent)
	require.NoError(t, err)
	defer p.Close()

	// Allow time for the main loop to start. If something went wrong, the checks below will trigger a failure
	require.NoError(t, p.WaitForReady(10*time.Second))

	_, err = p.KitedClient.SendLoginRequest("user@example.com", "secret", true)
	require.NoError(t, err)

	_, err = p.KitedClient.SendLogoutRequest(true)
	require.NoError(t, err)
}

// startup without logged-in user, do a few requests and make sure that a later login will succeed
func TestAccountOptionalLogin(t *testing.T) {
	// the panic recovery of the main loop was restarting after a nil ptr access.
	// The restarted loop was then using the stored auth info and returned a valid username
	rollbar.WithPanic(t)

	p, err := clientapp.StartDefaultTestEnvironment(false, nil)
	require.NoError(t, err)
	defer p.Close()

	// Allow time for the main loop to start. If something went wrong, the checks below will trigger a failure
	require.NoError(t, p.WaitForReady(10*time.Second))

	assertNoUser(t, p)

	_, err = p.KitedClient.SendLoginRequest("user@example.com", "secret", true)
	require.NoError(t, err)

	u, err := p.KitedClient.CurrentUser()
	require.NoError(t, err)
	require.EqualValues(t, "user@example.com", u.Email)
}

func sendEditorEvents(p *clientapp.TestEnvironment, file string, count int) {
	for i := 0; i < count; i++ {
		p.KitedClient.PostEditEvent("test_client", file, strings.Repeat("x", i), int64(i))
	}
}

func assertUserAndPlan(t *testing.T, p *clientapp.TestEnvironment) {
	user, err := p.Kited.AuthClient.GetUser()
	require.NoError(t, err, "expected user")
	require.NotNil(t, user)

	plan, err := p.KitedClient.CurrentPlan()
	require.NoError(t, err, "expected plan")
	require.NotNil(t, plan)
}

func assertNoUser(t *testing.T, p *clientapp.TestEnvironment) {
	user, err := p.Kited.AuthClient.GetUser()
	require.Error(t, err, "expected no user before initial login")
	require.Nil(t, user)
}
