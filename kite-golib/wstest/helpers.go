package wstest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/websocket"
)

// SocketConnCounter has a method to count the number of connections
type SocketConnCounter interface {
	ActiveConnections() []*websocket.Conn
}

// ReadFromWS reads from a websocket connection and sends the data over outgoing
func ReadFromWS(ctx context.Context, ws *websocket.Conn, outgoing chan string) {
	msg := make([]byte, 1024)
	ticker := time.NewTicker(100 * time.Millisecond)

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			n, err := ws.Read(msg)
			if err == nil {
				data := msg[:n]
				outgoing <- string(data)
			}
		}
	}
}

// WaitForConnections waits till all sockets are connected. Exported for testing other packages.
func WaitForConnections(t *testing.T, ws SocketConnCounter, expected int) {
	timeout := time.NewTimer(time.Second)
	tick := time.NewTicker(50 * time.Millisecond)
	for {
		select {
		case <-tick.C:
			if len(ws.ActiveConnections()) == expected {
				return
			}
		case <-timeout.C:
			require.EqualValues(t, expected, len(ws.ActiveConnections()),
				"WS connection must be established before proceeding")
			return
		}
	}
}
