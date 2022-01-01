package telemetry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/kiteco/kiteco/kite-golib/domains"
)

var clientVersion string

// SetClientVersion sets the global client version, which is send with PostRecord...()
// Clients automatically pick up this value
func SetClientVersion(version string) {
	clientVersion = version
}

// Message defines the data which is send to our Kinesis endpoint
type Message struct {
	MessageID string `json:"messageId"`
	Version   int    `json:"version"`

	UserID            string                 `json:"userId"`
	Event             string                 `json:"event"`
	Timestamp         time.Time              `json:"timestamp"`
	OriginalTimestamp time.Time              `json:"originalTimestamp"`
	Properties        map[string]interface{} `json:"properties"`
}

// Client is an API to talk to Kite's custom Kinesis endpoint at t.kite.com
type Client interface {
	io.Closer

	// Track sends a HTTP POST request to the configured endpoint
	// It uses the stream name and api key which were set for the current instance of the api
	Track(ctx context.Context, userID, event string, data map[string]interface{}) error
}

// NewCommonClient returns a new Kite client for t.kite.com, which is configured with the given source
func NewCommonClient(config StreamConfig) Client {
	endpoint := fmt.Sprintf("https://%s/%s", domains.Telemetry, url.PathEscape(config.Stream()))
	return newConfiguredClient(endpoint, config.Stream(), config.Key())
}

// NewClient returns a new Kite client for t.kite.com, which is ready for production use
func NewClient(streamName, apiKey string) Client {
	endpoint := fmt.Sprintf("https://%s/%s", domains.Telemetry, url.PathEscape(streamName))
	return newConfiguredClient(endpoint, streamName, apiKey)
}

// newConfiguredClient returns a new Kite kinesis client, which is ready for production use
func newConfiguredClient(httpEndpoint, streamName, apiKey string) Client {
	// fixme set proxy server
	httpClient := http.Client{}

	client := &httpClientAPI{
		stream:   streamName,
		key:      apiKey,
		client:   httpClient,
		endpoint: httpEndpoint,
	}
	client.init()
	return client
}
