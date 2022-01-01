package sidebar

// TestController implements controller for testing purposes
type TestController struct {
	StartReturns error
}

// Start implements controller
func (t *TestController) Start() error {
	return t.StartReturns
}

// Focus implements controller
func (t *TestController) Focus() error {
	return nil
}

// Stop implements controller
func (t *TestController) Stop() error {
	return nil
}

// SetWasVisible implements controller
func (t *TestController) SetWasVisible(bool) error {
	return nil
}

// WasVisible implements controller
func (t *TestController) WasVisible() (bool, error) {
	return false, nil
}

// Notify implements controller
func (t *TestController) Notify(id string) error {
	return nil
}
