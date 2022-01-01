package errors

// NewUI returns a new UI error message
func NewUI(ui, msg string) error {
	return UI{
		ui:  ui,
		msg: msg,
	}
}

// UI is an error with a message for the user interface and a fallback message, e.g. for rollbar
type UI struct {
	msg string
	ui  string
}

// Error implements the error interface and returns the fallback message
func (e UI) Error() string {
	return e.msg
}

// UI returns the message intended for the user interface or, when empty, the fallback message
func (e UI) UI() string {
	if e.ui != "" {
		return e.ui
	}
	return e.msg
}
