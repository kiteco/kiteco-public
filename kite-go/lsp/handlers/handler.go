package handlers

const (
	completeURL      = "http://localhost:46624/clientapi/editor/complete"
	eventURL         = "http://localhost:46624/clientapi/editor/event"
	statusURL        = "http://localhost:46624/clientapi/status"
	trackMixpanelURL = "http://localhost:46624/clientapi/metrics/mixpanel"
	onboardingURL    = "http://localhost:46624/clientapi/plugins/onboarding_file"
	contentType      = "application/json"
	kiteTypesEnabled = "kiteTypesEnabled"
)

// Handlers routes LSP requests to Kite and vice versa.
// It contains a cache to store text document state.
type Handlers struct {
	files   map[string]string
	Options interface{}
}

// New creates a new Handlers
func New() *Handlers {
	return &Handlers{
		files: make(map[string]string),
	}
}
