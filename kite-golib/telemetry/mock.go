package telemetry

import (
	"context"
	"strings"
	"sync"
	"time"
)

// MockClient is a mock implementation of a Kinesis Client. It's intended to be used in test cases.
// It can be used as a custom with "track.SetCustomClient()"
type MockClient struct {
	mu    sync.Mutex
	track []Message
}

// Tracked returns the messages sent by this mock client
func (m *MockClient) Tracked() []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.track
}

// TrackedFilteredByEvent returns the messages sent by this mock client, filtered by event name
func (m *MockClient) TrackedFilteredByEvent(name string) []Message {
	return m.TrackedFiltered(func(msg Message) bool {
		return msg.Event == name
	})
}

// TrackedFiltered returns the messages sent by this mock client, filtered with the given condition
func (m *MockClient) TrackedFiltered(condition func(msg Message) bool) []Message {
	all := m.Tracked()
	var result []Message

	for _, m := range all {
		if condition(m) {
			result = append(result, m)
		}
	}

	return result
}

// GetTracked returns the tracked message at 'index' or nil if it's not available
func (m *MockClient) GetTracked(index int) *Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.track) <= index {
		return nil
	}

	return &m.track[index]
}

// Close implements closer and releases acquired resources
func (m *MockClient) Close() error {
	return nil
}

// Track implements interface Client
func (m *MockClient) Track(ctx context.Context, userID, event string, data map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	m.track = append(m.track, Message{
		UserID:            userID,
		Event:             event,
		Version:           3,
		Timestamp:         now,
		OriginalTimestamp: now,
		Properties:        data,
	})

	return nil
}

func (m *MockClient) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var l []string
	for _, t := range m.track {
		l = append(l, t.Event)
	}
	return strings.Join(l, ",")
}
