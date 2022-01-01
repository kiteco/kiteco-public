package test

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/autosearch"
	"github.com/kiteco/kiteco/kite-go/client/internal/clientapp"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"golang.org/x/net/websocket"
)

// AutosearchClient simplifies testing of the autosearch component
// it implements the client side
type AutosearchClient struct {
	wsManager     *autosearch.Manager
	wsClient      *websocket.Conn
	ctx           context.Context
	ctxCancel     func()
	clientMsgChan chan string
}

// NewClient returns a configured client, which is using the kited server of testEnv
func NewClient(testEnv *clientapp.TestEnvironment) (*AutosearchClient, error) {
	var mgr *autosearch.Manager
	for _, comp := range testEnv.Kited.Components() {
		var ok bool
		if mgr, ok = comp.(*autosearch.Manager); ok {
			break
		}
	}

	if mgr == nil {
		return nil, errors.Errorf("no autosearch manager found")
	}

	return NewConfiguredClient(*testEnv.Kited.URL, mgr)
}

// NewConfiguredClient returns a new, fully configured and ready autosearch client
func NewConfiguredClient(serverURL url.URL, mgr *autosearch.Manager) (*AutosearchClient, error) {
	ctx, ctxCancel := context.WithCancel(context.Background())
	c := &AutosearchClient{
		wsManager:     mgr,
		clientMsgChan: make(chan string, 10),
		ctx:           ctx,
		ctxCancel:     ctxCancel,
	}

	serverURL.Scheme = "ws"
	serverURL.Path = "/autosearch"

	var err error
	if c.wsClient, err = websocket.Dial(serverURL.String(), "", "http://localhost"); err != nil {
		return nil, err
	}

	if err = c.waitForConnections(1); err != nil {
		return nil, err
	}

	go c.receiveClientMessages(c.ctx, c.clientMsgChan)
	return c, nil
}

// Close implements io.Closer
func (c *AutosearchClient) Close() error {
	c.ctxCancel()

	return c.wsClient.Close()
}

// BroadcastServerMessage triggers the websocket server to send a message to the client
func (c *AutosearchClient) BroadcastServerMessage(autosearchID string) {
	c.wsManager.EventResponse(&response.Root{
		Results: []interface{}{
			map[string]interface{}{"autosearch_id": autosearchID},
		},
	})
}

// BroadcastServerAutosearchMessage broadcasts a message in the style of kited local
func (c *AutosearchClient) BroadcastServerAutosearchMessage(autosearchID string) {
	c.wsManager.EventResponse(&response.Root{
		Results: []interface{}{
			&response.Autosearch{
				AutosearchID: autosearchID,
			},
		},
	})
}

// ReceiveClientMessage uses the websocket client to receive a message from the server
// it returns an error if the receive operation timed out
func (c *AutosearchClient) ReceiveClientMessage() (string, error) {
	type autosearchID struct {
		ID string `json:"id"`
	}

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case payload := <-c.clientMsgChan:
		var aid autosearchID
		err := json.Unmarshal([]byte(payload), &aid)
		return aid.ID, err
	}
}

func (c *AutosearchClient) waitForConnections(expected int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

WAIT:
	for {
		select {
		case <-ctx.Done():
			break WAIT
		default:
			if c.wsManager.ActiveConnectionCount() == expected {
				return nil
			}
		}
	}

	return errors.Errorf("WS connection must be established before proceeding")
}

func (c *AutosearchClient) receiveClientMessages(ctx context.Context, target chan string) {
	msg := make([]byte, 1024)
	ticker := time.NewTicker(100 * time.Millisecond)

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			n, err := c.wsClient.Read(msg)
			if err == nil {
				data := msg[:n]
				target <- string(data)
			}
		}
	}
}
