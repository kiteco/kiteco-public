package messagebox

// Options describes how a message box should be displayed.
type Options struct {
	Key   string // key for warning suppression
	Text  string // text displayed to the user
	Title string // window title
	Info  string // info text displayed to the user
}

// ShowAlert shows an alert message box
func ShowAlert(opts Options) error {
	return showAlert(opts)
}

// DispatchWarning shows a warning message box.
// On OS X, it should not be called by the main thread (inside initialization routines).
func DispatchWarning(opts Options) error {
	return showWarning(opts)
}
