package component

// MetricsManager is used by components to access and modify metrics
type MetricsManager interface {
	Core

	// SetMenubarVisible updates the status whether the Kite icon is visible in the menubar
	SetMenubarVisible(v bool)

	// Returns the current status about menubar visibility
	IsMenubarVisible() bool

	// GetRegion
	GetRegion() string

	// Updates the region used by the client. This will be used in metrics processed later
	SetRegion(region string)

	Identify()

	// UpdateUser adds new properties to the remotely stored user information
	// Both mixpanel and segments.io user data are updated
	UpdateUser(traits map[string]interface{})

	GitSeeker
}

// GitSeeker determines whether Git was found on the system
type GitSeeker interface {
	GitFound() bool
}
