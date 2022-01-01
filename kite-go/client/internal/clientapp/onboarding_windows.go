package clientapp

import "github.com/kiteco/kiteco/kite-go/client/internal/windowsui"

// launchOnboarding runs KiteOnboarding.exe
func launchOnboarding() error {
	return windowsui.RunOnboarding()
}
