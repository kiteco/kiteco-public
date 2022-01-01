package telemetry

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_HttpClient(t *testing.T) {
	var receivedEvents []Message
	expectedEvents := 2

	wg := sync.WaitGroup{}
	wg.Add(1)

	serverRequestHandler := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if len(receivedEvents) == expectedEvents {
				wg.Done()
			}
		}()

		if r.Header.Get("Content-Type") == "application/json" && r.Header.Get("x-api-key") == "api-key" {
			if bytes, err := ioutil.ReadAll(r.Body); err == nil {
				var body Message
				err := json.Unmarshal(bytes, &body)
				if err == nil {
					receivedEvents = append(receivedEvents, body)
					w.WriteHeader(http.StatusOK)
					return
				}
			}
		}

		w.WriteHeader(http.StatusInternalServerError)
	}

	server := httptest.NewServer(http.HandlerFunc(serverRequestHandler))
	defer server.Close()

	apiClient := newConfiguredClient(server.URL, "stream-name", "api-key")
	defer apiClient.Close()

	SetClientVersion("client-version-value")

	err := apiClient.Track(context.Background(), "userId1", "test-event1", map[string]interface{}{
		"property1": "propertyValue1",
	})
	require.NoError(t, err)

	err = apiClient.Track(context.Background(), "userId1", "test-event1", AugmentProps(map[string]interface{}{
		"propertyCommon1": "propertyValueCommon1",
		// to test that common props don't override
		"os": "event-os",
	}))
	require.NoError(t, err)

	// wait, but make sure we don't block forever
	go func() {
		time.Sleep(10 * time.Second)
		wg.Done()
	}()
	wg.Wait()
	require.Len(t, receivedEvents, expectedEvents, "expected %d successful event requests")

	// event without common properties
	msg := receivedEvents[0]
	require.EqualValues(t, "userId1", msg.UserID)
	require.EqualValues(t, "test-event1", msg.Event)
	require.EqualValues(t, "propertyValue1", msg.Properties["property1"])
	require.Empty(t, msg.Properties["sent_at"])
	require.Empty(t, msg.Properties["os"])
	require.Empty(t, msg.Properties["client_version"])

	// event with common properties
	msg = receivedEvents[1]
	require.EqualValues(t, "userId1", msg.UserID)
	require.EqualValues(t, "test-event1", msg.Event)
	require.EqualValues(t, "propertyValueCommon1", msg.Properties["propertyCommon1"])
	require.NotEmpty(t, msg.Properties["sent_at"])
	require.EqualValues(t, "event-os", msg.Properties["os"])
	require.EqualValues(t, "client-version-value", msg.Properties["client_version"])
}
