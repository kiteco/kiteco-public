package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// httpKinesisAPI is a simple implementation to send segment-style messages to our Kinesis endpoint
// this client is send the messages asynchronously
type httpClientAPI struct {
	stream   string
	key      string
	client   http.Client
	endpoint string

	ctx       context.Context
	cancel    func()
	eventChan chan Message
}

// init prepares this client for work. This needs to be called before messages are posted
func (a *httpClientAPI) init() {
	a.ctx, a.cancel = context.WithCancel(context.Background())
	a.eventChan = make(chan Message, 50)

	go a.eventLoop()
}

// eventLoop sends out pending events to the remote server
// it's supposed to be run as a go routine
func (a *httpClientAPI) eventLoop() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case msg := <-a.eventChan:
			payload, err := json.Marshal(msg)
			if err != nil {
				log.Printf("error marshalling json for event: %v", err)
				continue
			}

			req, err := http.NewRequest("POST", a.endpoint, bytes.NewReader(payload))
			if err != nil {
				log.Printf("error creating event request: %v", err)
				continue
			}
			req.Header.Set("x-api-key", a.key)
			req.Header.Set("Content-Type", "application/json")

			resp, err := a.client.Do(req)
			if err != nil {
				log.Printf("error sending event request: %v", err)
				continue
			}

			resp.Body.Close()
			status := resp.StatusCode

			if status != http.StatusOK {
				log.Printf("http status error sending Kite event %s: %d/%s", msg.Event, status, resp.Status)
			}
		}
	}
}

// Close releases resources of the Kinesis API client
func (a *httpClientAPI) Close() error {
	a.cancel()
	close(a.eventChan)

	// Go 1.11 doesn't have http.Client.CloseIdleConnections() yet
	// this is a copy of Go's implementation
	type closeIdler interface {
		CloseIdleConnections()
	}
	if tr, ok := a.client.Transport.(closeIdler); ok {
		tr.CloseIdleConnections()
	}

	return nil
}

// Track implements interface Client
func (a *httpClientAPI) Track(ctx context.Context, userID, event string, data map[string]interface{}) error {
	msg := createMessage(userID, event, data)
	select {
	case a.eventChan <- msg:
		return nil
	default:
		return errors.Errorf("event was dropped: %s", event)
	}
}
