package clienttelemetry

var (
	// InstallIDs sends to Mixpanel & Customer.io under the install ID
	InstallIDs = Options{mp: InstallID, cio: InstallID}
	// KiteOnly sends to Kite under the metrics ID
	KiteOnly = Options{kite: MetricsID}
)

// EventWithKiteTelemetry sends events to Mixpanel, CIO, and t.kite.com
func EventWithKiteTelemetry(name string, props map[string]interface{}) {
	Default.Kite(MetricsID).Event(name, props)
}

// KiteTelemetry aliases KiteOnly.Event
func KiteTelemetry(name string, props map[string]interface{}) {
	KiteOnly.Event(name, props)
}

// Event aliases Default.Event
func Event(name string, props map[string]interface{}) {
	Default.Event(name, props)
}

// Update aliases Default.Update
func Update(props map[string]interface{}) {
	Default.Update(props)
}
