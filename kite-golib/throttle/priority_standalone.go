// +build standalone

package throttle

// SetLowPriority lowers the calling process (including all threads) priority.
func SetLowPriority() error {
	return nil
}
