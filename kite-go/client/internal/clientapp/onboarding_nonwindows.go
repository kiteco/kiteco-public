// +build !windows

package clientapp

// launchOnboarding does nothing on non-windows platforms
func launchOnboarding() error {
	return nil
}
