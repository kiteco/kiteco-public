package clienttelemetry

import (
	"runtime"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/telemetry"
	"github.com/stretchr/testify/require"
)

func Test_Track(t *testing.T) {
	mock := &telemetry.MockClient{}
	SetCustomTelemetryClient(mock)

	ids := userids.NewUserIDs("", "machine")
	ids.SetUser(1, "", true)
	SetClientVersion("version")
	SetUserIDs(ids)

	KiteTelemetry("test", map[string]interface{}{
		"testprop": "testvalue",
	})

	require.Equal(t, 1, len(mock.Tracked()))

	track := mock.GetTracked(0)

	// Check common values
	require.EqualValues(t, "1", track.UserID)
	require.True(t, time.Now().Sub(track.Timestamp) < 10*time.Second, "timestamp not in required range: %s", track.Timestamp)
	require.True(t, time.Now().Sub(track.OriginalTimestamp) < 10*time.Second, "timestamp not in required range: %s", track.OriginalTimestamp)
	require.EqualValues(t, "machine", track.Properties["machine"])
	require.EqualValues(t, runtime.GOOS, track.Properties["platform"])
	require.EqualValues(t, "version", track.Properties["client_version"])
	require.Contains(t, track.Properties, "client_uptime_ns")

	// Check custom values
	require.EqualValues(t, "testvalue", track.Properties["testprop"])
}

func Test_Deferred(t *testing.T) {
	mock := &telemetry.MockClient{}
	SetCustomTelemetryClient(mock)

	SetUserIDs(nil)
	KiteTelemetry("test 1", map[string]interface{}{
		"testprop": "testvalue",
	})
	require.Equal(t, 0, len(mock.Tracked()), "messages can't be send when no userids are present")

	ids := userids.NewUserIDs("install-id", "machine-id")
	SetUserIDs(ids)

	SetClientVersion("version")
	require.Equal(t, 0, len(mock.Tracked()))

	SetUserIDs(ids)
	require.Equal(t, 0, len(mock.Tracked()))

	KiteTelemetry("test 2", map[string]interface{}{
		"testprop": "testvalue",
	})
	require.Equal(t, 1, len(mock.Tracked()))

	track := mock.GetTracked(0)

	// Check common values
	require.EqualValues(t, "install-id", track.UserID, "the install id must be used as fallback id when there's no user id")
	require.EqualValues(t, "machine-id", track.Properties["machine"])
	require.EqualValues(t, runtime.GOOS, track.Properties["platform"])
	require.EqualValues(t, "version", track.Properties["client_version"])
	require.EqualValues(t, "install-id", track.Properties["install_id"])
	require.Contains(t, track.Properties, "client_uptime_ns")

	// Check custom values
	require.EqualValues(t, "testvalue", track.Properties["testprop"])

	ids.SetUser(1, "email@example.com", true)
	KiteTelemetry("test 3", map[string]interface{}{
		"testprop": "testvalue",
	})
	require.Equal(t, 2, len(mock.Tracked()))
	track = mock.GetTracked(1)
	require.EqualValues(t, "1", track.UserID)
	require.EqualValues(t, "install-id", track.Properties["install_id"])
}
