// +build windows darwin
// +build !standalone
//go:generate go-bindata -o bindata.go -pkg statusicon asset/...

package statusicon

import (
	"fmt"
	"log"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/client/internal/updates"
	"github.com/kiteco/kiteco/kite-go/client/platform"
	"github.com/kiteco/kiteco/kite-go/client/sidebar"
	"github.com/kiteco/kiteco/kite-golib/domains"
	"github.com/kiteco/kiteco/kite-golib/systray"
	"github.com/skratchdot/open-golang/open"
)

const restartTimeout = time.Second * 5

// The server URLs that appear in the "server" submenu
var serverURLs = []string{
	fmt.Sprintf("https://%s/", domains.Alpha),
	fmt.Sprintf("https://%s/", domains.Staging),
	"https://test-0.kite.com/",
	"https://test-1.kite.com/",
	"https://test-2.kite.com/",
	"https://test-3.kite.com/",
	"https://test-4.kite.com/",
	"https://test-5.kite.com/",
	"https://test-6.kite.com/",
	"https://test-7.kite.com/",
	"https://test-8.kite.com/",
	"https://192.168.30.10/",
	"http://127.0.0.1:9090/",
}

// UI is a handle to the status icon
type UI struct {
	kitedURL *url.URL // still in use by statusicon_linux.go
	platform *platform.Platform
	auth     component.AuthClient
	settings component.SettingsManager
	metrics  component.MetricsManager
	updater  updates.Manager

	mu         sync.Mutex
	signedInAs *systray.MenuItem
	servers    []*systray.MenuItem
	icon       []byte
}

// NewManager returns a new statusicon component
func NewManager(updater updates.Manager) *UI {
	return &UI{
		updater: updater,
	}
}

// Name implements interface Core
func (ui *UI) Name() string {
	return "statusicon"
}

// Initialize implements component interface Initializer
func (ui *UI) Initialize(opts component.InitializerOptions) {
	ui.kitedURL = opts.KitedURL
	ui.auth = opts.AuthClient
	ui.settings = opts.Settings
	ui.platform = opts.Platform
	ui.metrics = opts.Metrics

	ui.start()
}

// Start runs the systray loop and passes the Kite client to onReady
func (ui *UI) start() {
	ui.onBeforeRun() // do platform-specific things
	go systray.Run(ui.onReady)
}

// setVisible changes the visibility of the tray icon.
func (ui *UI) setVisible(v bool) {
	ui.metrics.SetMenubarVisible(v)
	if v {
		systray.Show("Kite", "", ui.icon)
	} else {
		systray.Hide()
	}
}

func (ui *UI) updateServerMenu() {
	current := ui.settings.Server()
	for i, url := range serverURLs {
		if url == current {
			ui.servers[i].Check()
		} else {
			ui.servers[i].Uncheck()
		}
	}
}

// onReady builds the menubar UI using systray and sets up the appropriate
// handlers for each menu item.
func (ui *UI) onReady(h systray.Handle) {
	ui.onHandleReceived(h) // do platform-specific things

	// Setup icon
	switch runtime.GOOS {
	case "windows":
		ui.icon = MustAsset("asset/kite_monochrome.ico")
	case "linux":
		ui.icon = MustAsset("asset/kite_light.png")
	default:
		ui.icon = MustAsset("asset/kite_monochrome.tiff")
	}

	// Signed in as ...
	ui.signedInAs = systray.AddMenuItem(ui.signedInTitle(), "", ui.onSignedInAsClicked)
	systray.AddMenuItem("Settings...", "", ui.onSettingsClicked)
	systray.AddSeparator()

	// Support
	systray.AddMenuItem("Support", "", ui.onGetSupportClicked)

	// Open/quit sidebar
	systray.AddMenuItem("Open Copilot", "", ui.onOpenSidebarClicked)
	systray.AddSeparator()

	// Check for updates
	if runtime.GOOS == "darwin" {
		systray.AddMenuItem("Check for Updates...", "", ui.onUpdateClicked)
	}

	// Quit
	systray.AddMenuItem("Quit Kite", "", ui.onQuitClicked)

	// Version
	version := systray.AddMenuItem(ui.platform.ClientVersion, "", nil)
	version.Disable()

	// Update visibility
	visible, _ := ui.settings.GetBool(settings.StatusIconKey)
	ui.setVisible(visible)

	// update user info menu item
	ui.updateUserInfo()
}

// SettingUpdated implements component.Settings
func (ui *UI) SettingUpdated(key, value string) {
	if key == settings.StatusIconKey {
		visible, _ := ui.settings.GetBool(settings.StatusIconKey)
		ui.setVisible(visible)
	}
}

// SettingDeleted implements component.Settings
func (ui *UI) SettingDeleted(key string) {
	// Noop
}

// LoggedIn implements component UserAuth
func (ui *UI) LoggedIn() {
	ui.updateUserInfo()
}

// LoggedOut implements component UserAuth
func (ui *UI) LoggedOut() {
	ui.updateUserInfo()
}

func (ui *UI) updateUserInfo() {
	// the menu item may not yet be initialized.
	// This happens when the logged in event is triggered before the UI loop is ready
	if ui.signedInAs != nil {
		ui.signedInAs.SetTitle(ui.signedInTitle())
	}
}

func (ui *UI) signedInTitle() string {
	user, err := ui.auth.GetUser()
	if err != nil {
		return "Login or Create Account"
	}
	return fmt.Sprintf("Signed in as %s", user.Email)
}

func (ui *UI) onOpenSidebarClicked() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	if err := sidebar.Start(); err != nil {
		log.Printf("onOpenSidebarClicked: %s", err)
	}
}

func (ui *UI) onQuitSidebarClicked() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	if err := sidebar.Stop(); err != nil {
		log.Printf("onQuitSidebarClicked: %s", err)
	}
}

func (ui *UI) onUpdateClicked() {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.updater.CheckForUpdates(true)
}

func (ui *UI) onQuitClicked() {
	sidebar.Stop()
	systray.Hide()
	// call platform specific terminate method
	terminate()
}

func (ui *UI) onGetSupportClicked() {
	open.Run("kite://help")
}
