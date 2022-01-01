// +build !darwin

package health

// IsResponsive is a mock for non-macOS platforms
func IsResponsive() bool {
	return true
}
