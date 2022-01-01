package remotectrl

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/kiteco/websocket"
)

// Message type variables for the possible message types.
var (
	DesktopNotification = "desktop_notification"
	SetNotification     = "set_notification"
	SetConversionCohort = "set_conversion_cohort"
	SetConfig           = "set_config"
)

// messageTypeSet contains the possible messages remotectrl will pass to its handlers.
var messageTypeSet = map[string]struct{}{
	DesktopNotification: {},
	SetNotification:     {},
	SetConversionCohort: {},
	SetConfig:           {},
}

// Message is a remote control message sent by the backend.
type Message struct {
	// ID is used by the message publisher to allow for client-side deduping.
	// We can't use the nchan event ID for this, as it is not controllable by the publisher.
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Handler implementers can get passed an rc message.
type Handler interface {
	HandleRemoteMessage(msg Message) error
}

// Listen listens to messages from a remote control endpoint
func Listen(u string, c *http.Client, onMessage func(Message)) *Listener {
	l := &Listener{
		url:       u,
		client:    c,
		onMessage: onMessage,
	}
	f, cancel := kitectx.Background().ClosureWithCancel(func(ctx kitectx.Context) error {
		for {
			err := l.listen(ctx)
			if err != nil {
				log.Println("connection dropped:", err)
			}
		}
	})
	l.cancel = cancel
	kitectx.Go(f)
	return l
}

// Close closes the underlying resources
func (l *Listener) Close() error {
	l.cancel()
	return nil
}

// Listener encapsulates a websocket connection to our pub/sub system.
type Listener struct {
	url       string
	client    *http.Client
	cancel    context.CancelFunc
	onMessage func(Message)

	conn *websocket.Conn
}

type event struct {
	// ID is the *event* ID (separate from the message ID)
	ID string `json:"-"`
	// Message is a server -> client message frame
	Messages []Message `json:"messages,omitempty"`
}

func (l *Listener) processEvent(ctx kitectx.Context, evt event) {
	ctx.CheckAbort()

	if len(evt.Messages) == 0 {
		log.Println("no messages on event")
	}
	for _, msg := range evt.Messages {
		// Range vars do not automatically bind to new var on each iteration
		msg := msg
		kitectx.Go(func() error {
			if _, ok := messageTypeSet[msg.Type]; !ok {
				rollbar.Error(errors.New("unhandled remote message type", msg.Type))
				return nil
			}
			// don't process if the ack failed to avoid double-processing.
			l.onMessage(msg)
			return nil
		})
	}
}

func (l *Listener) write(ctx kitectx.Context, evt event) error {
	wc, err := l.conn.Writer(ctx.Context(), websocket.MessageText)
	if err != nil {
		return err
	}
	defer wc.Close()
	return json.NewEncoder(wc).Encode(evt)
}

const minBackoff = 200 * time.Millisecond
const maxBackoff = 10 * time.Minute

// listen connects, retrying until a connection is made,
// and then handles events on the connection until the connection closes.
// The returned error is the error that caused the connection to close.
func (l *Listener) listen(ctx kitectx.Context) error {
	ctx.CheckAbort()

	opts := &websocket.DialOptions{
		HTTPClient:   l.client,
		Subprotocols: []string{"ws+meta.nchan"},
	}

	var conn *websocket.Conn
	for backoff := minBackoff; conn == nil; backoff = min(2*backoff, maxBackoff) {
		ctx.CheckAbort()
		err := ctx.WithTimeout(15*time.Second, func(ctx kitectx.Context) error {
			var err error
			conn, _, err = websocket.Dial(ctx.Context(), l.url, opts)
			return err
		})
		if conn != nil {
			break
		}
		log.Println("connection error:", err, backoff)
		sleep(ctx, backoff)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	l.conn = conn

	for {
		ctx.CheckAbort()
		ty, msg, err := l.conn.Read(ctx.Context())
		if err != nil {
			return err
		}
		if ty != websocket.MessageText {
			continue
		}
		evt, err := parseEvent(msg)
		if err != nil {
			log.Printf("could not parse event from backend with id %#v: %s", evt.ID, err)
		}
		l.processEvent(ctx, evt)
	}
}

// sleep polls for ctx.Done() or expires after the passed duration.
// It returns false if Done was called or true if the sleep expired naturally.
func sleep(ctx kitectx.Context, dur time.Duration) {
	t := time.NewTimer(dur)
	defer t.Stop()

	for {
		select {
		case <-ctx.AbortChan():
			ctx.Abort()
		case <-t.C:
			return
		}
	}
}

var errMalformedEvent = errors.Errorf("malformed event received")

func parseEvent(m []byte) (event, error) {
	/* Data format with the ws+meta.nchan subprotocol:
	 * id: STRING
	 * content-type: STRING
	 * \n
	 * ...body
	 */
	splitdata := bytes.SplitN(m, []byte("\n\n"), 2)
	if len(splitdata) != 2 {
		return event{}, errMalformedEvent
	}

	splitmeta := bytes.SplitN(splitdata[0], []byte("\n"), 2)
	idSegs := bytes.Split(splitmeta[0], []byte(" "))
	if len(idSegs) < 2 {
		return event{}, errMalformedEvent
	}

	evt := event{
		ID: string(idSegs[1]),
	}
	err := json.Unmarshal(splitdata[1], &evt)
	return evt, err
}

func min(t1, t2 time.Duration) time.Duration {
	if t1 < t2 {
		return t1
	}
	return t2
}
