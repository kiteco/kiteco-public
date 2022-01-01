package sidebar

import (
	"log"

	"github.com/kiteco/kiteco/kite-go/client/component"
)

// controller is a platform-agnostic API to the sidebar application
type controller interface {
	Start() error
	Focus() error
	Stop() error
	SetWasVisible(bool) error
	WasVisible() (bool, error)

	Notify(id string) error

	// We don't include Running because it's no longer reliable
	// in the presence of Electron-powered notifications.
}

var (
	c controller
)

// Init creates the platform specific sidebar controller
func Init(settings component.SettingsManager) {
	c = newController(settings)
}

// TestInit takes a TestController to listen in on sidebar calls
func TestInit(tc *TestController) {
	c = tc
}

// Start will start the sidebar. If the sidebar is already running, it will bring it into focus.
func Start() error {
	// Reset the value of WasVisible if we force a Start
	c.SetWasVisible(false)
	return c.Start()
}

// Focus will bring the sidebar into focus if it is running, otherwise it will do nothing
func Focus() error {
	return c.Focus()
}

// Stop will stop the sidebar. If the sidebar is not running, it will return successfully.
func Stop() error {
	return c.Stop()
}

// --

// SetRestartIfPreviouslyVisible sets state such that StartIfPreviouslyVisible actually
// starts the sidebar. Other calls to Start before StartIfPreviouslyVisible will erase
// this state. This logic is meant to enable restarting of the sidebar in the case Kite
// was shutdown and restarted via update.
func SetRestartIfPreviouslyVisible(val bool) error {
	return c.SetWasVisible(val)
}

// StartIfPreviouslyVisible starts the sidebar if the sidebar was running when Kite was shut down.
func StartIfPreviouslyVisible() error {
	visible, err := c.WasVisible()
	if err != nil {
		return err
	}
	if visible {
		return Start()
	}

	// Reset the value of WasVisible after it is used
	return c.SetWasVisible(false)
}

// Notify displays the notification window with the contents specified by the notification ID
func Notify(id string) error {
	log.Println("triggering desktop notification", id)
	return c.Notify(id)
}
