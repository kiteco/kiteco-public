package telemetry

// our available streams
var (
	StreamKiteStatus   StreamConfig = streamConfig{stream: "kite_status", key: "XXXXXXX"}
	StreamKiteService  StreamConfig = streamConfig{stream: "kite_service", key: "XXXXXXX"}
	StreamClientEvents StreamConfig = streamConfig{stream: "client_events", key: "XXXXXXX"}
)

// StreamConfig is a matching pair of stream name + API Key
// It's an interface to make this data read-only
type StreamConfig interface {
	Stream() string
	Key() string
}

// StreamConfig defines a stream with the matching API key
type streamConfig struct {
	stream string
	key    string
}

func (c streamConfig) Stream() string {
	return c.stream
}

func (c streamConfig) Key() string {
	return c.key
}
