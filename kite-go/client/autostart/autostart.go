package autostart

// SetDisabled enables or disables autostart depending on the value of disabled
func SetDisabled(disabled bool) error {
	return setEnabled(!disabled)
}
