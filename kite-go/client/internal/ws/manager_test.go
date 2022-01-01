package ws

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-golib/wstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/websocket"
)

func Test_Active(t *testing.T) {
	m := NewManager()
	defer m.CloseConnections()

	assert.True(t, m.IsActive(), "The connection must be active after init")

	m.AcceptConnections(false)
	assert.False(t, m.IsActive(), "The connection must be inactive after call of config method")
	m.AcceptConnections(true)
	assert.True(t, m.IsActive(), "The connection must be active again after call of config method")
}

func Test_WS(t *testing.T) {
	mgr := NewManager()
	defer mgr.CloseConnections()

	s := httptest.NewServer(websocket.Handler(mgr.HandleEventsWS))
	defer s.Close()

	serverURL, err := url.Parse(s.URL)
	require.NoError(t, err)
	serverURL.Scheme = "ws"

	ws1, err := websocket.Dial(serverURL.String(), "", "http://localhost")
	require.NoError(t, err)
	defer ws1.Close()

	ws2, err := websocket.Dial(serverURL.String(), "", "http://localhost")
	require.NoError(t, err)
	defer ws2.Close()

	wstest.WaitForConnections(t, mgr, 2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws1data := make(chan string)
	ws2data := make(chan string)

	go wstest.ReadFromWS(ctx, ws1, ws1data)
	go wstest.ReadFromWS(ctx, ws2, ws2data)

	// two websocket client connections at this point
	// a broadcast has to send the message to both of them
	msgPayload := map[string]string{"myKey": "myValue"}
	mgr.BroadcastJSON(msgPayload)

	expected, err := json.Marshal(msgPayload)
	require.NoError(t, err)
	expectedStr := string(expected)

	var gotws1, gotws2 bool
	for !gotws1 || !gotws2 {
		select {
		case <-ctx.Done():
			require.Fail(t, "did not receive all expected messages from websocket connections")
		case payload := <-ws1data:
			require.Equal(t, payload, expectedStr)
			gotws1 = true
		case payload := <-ws2data:
			require.Equal(t, payload, expectedStr)
			gotws2 = true
		}
	}

	// close client connection and make sure the server notices this
	assert.EqualValues(t, 2, len(mgr.ActiveConnections()))
	err = ws1.Close()
	assert.NoError(t, err)
	// allow the connection to close
	time.Sleep(100 * time.Millisecond)
	assert.EqualValues(t, 1, len(mgr.ActiveConnections()))

	// close last server connection
	mgr.CloseConnections()
	assert.EqualValues(t, 0, len(mgr.ActiveConnections()))
}
