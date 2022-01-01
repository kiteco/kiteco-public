package track

import (
	analytics "gopkg.in/segmentio/analytics-go.v3"
)

// EventField determines the property under which the tracked message lies in the Segment JSON message.
const EventField = "event_250816"

// Client wraps the segmentio/analytics-go Client
type Client struct {
	client analytics.Client
}

// NewClient returns a new Client for a specific class of event
func NewClient(segmentToken string, callback analytics.Callback) (*Client, error) {
	client, err := analytics.NewWithConfig(segmentToken, analytics.Config{
		Callback: callback,
		Logger:   noopLogger{},
	})
	if err != nil {
		return nil, err
	}

	return &Client{client: client}, nil
}

type noopLogger struct{}

func (s noopLogger) Logf(string, ...interface{}) {}

func (s noopLogger) Errorf(string, ...interface{}) {}

// Track queues data for processing by Segment.
func (c *Client) Track(eventName string, userID string, v interface{}) error {
	t := analytics.Track{
		Event:  eventName,
		UserId: userID,
		Properties: map[string]interface{}{
			EventField: v,
		},
	}
	return c.client.Enqueue(t)
}
