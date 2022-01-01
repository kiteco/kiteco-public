package mixpanel

import (
	"github.com/dukex/mixpanel"
)

// Metrics encapsulates logic to talk to mixpanel and record events.
type Metrics struct {
	Metrics mixpanel.Mixpanel
	enabled bool
	token   string
}

// NewMetrics constructs a new client to talk to mixpanel.
func NewMetrics(token string) *Metrics {
	return &Metrics{
		Metrics: mixpanel.New(token, ""),
		enabled: token != "",
		token:   token,
	}
}

// NewMockMetrics constructs a new mock client, suitable for unit tests
func NewMockMetrics() *Metrics {
	return &Metrics{
		Metrics: mixpanel.NewMock(),
		enabled: true,
	}
}

// Track tracks the specified event in mixpanel against the given user.
func (m *Metrics) Track(id string, event string, props map[string]interface{}) error {
	if !m.enabled {
		return nil
	}
	return m.Metrics.Track(id, event, &mixpanel.Event{
		Properties: props,
	})
}

// Identify tells mixpanel about a user.
func (m *Metrics) Identify(id string, props map[string]interface{}) error {
	if !m.enabled {
		return nil
	}

	return m.Metrics.Update(id, &mixpanel.Update{
		Operation:  "$set",
		Properties: props,
	})
}
