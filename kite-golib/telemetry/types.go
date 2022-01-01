package telemetry

// CustomEvent is used to post to the client tracking apis
type CustomEvent struct {
	Event string                 `json:"event"`
	Key   string                 `json:"key"`
	Props map[string]interface{} `json:"props"`
}
