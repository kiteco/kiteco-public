package startup

import (
	"os"
	"strings"
)

const (
	// SystemBoot mode represents startup via machine boot / user login
	SystemBoot Mode = iota

	// RelaunchAfterUpdate mode represents startup after an update has been applied
	RelaunchAfterUpdate

	// ManualLaunch mode represents a manual startup by the user
	ManualLaunch

	// PluginLaunch mode represents being launched by a plugin
	PluginLaunch

	// PluginLaunchWithSidebar mode represents being launched by a plugin which requests the Copilot setup workflow
	PluginLaunchWithSidebar

	// SidebarRestart mode represents being restarted by an action in the Sidebar
	SidebarRestart
)

// Mode defines the possible startup states we care about
type Mode int

// String implements fmt.Stringer
func (m Mode) String() string {
	switch m {
	case SystemBoot:
		return "SystemBoot"
	case RelaunchAfterUpdate:
		return "RelaunchAfterUpdate"
	case ManualLaunch:
		return "ManualLaunch"
	case PluginLaunch:
		return "PluginLaunch"
	case PluginLaunchWithSidebar:
		return "PluginLaunchWithSidebar"
	case SidebarRestart:
		return "SidebarRestart"
	}

	return "Undefined"
}

const (
	systemBootFlag              = "--system-boot"
	relaunchAfterUpdateFlag     = "--relaunch-after-update"
	pluginLaunchFlag            = "--plugin-launch"
	pluginLaunchWithCopilotFlag = "--plugin-launch-with-copilot"
	manualLaunchFlag            = "--manual-launch"
	sidebarRestartFlag          = "--sidebar-restart"
)

// GetMode returns the way in which the current process was started
func GetMode(args []string) Mode {
	defer reset()

	// Check flags that we use which are consistent across platforms:
	for _, arg := range args {
		switch arg {
		case systemBootFlag:
			return SystemBoot
		case relaunchAfterUpdateFlag:
			return RelaunchAfterUpdate
		case pluginLaunchFlag:
			return PluginLaunch
		case pluginLaunchWithCopilotFlag:
			return PluginLaunchWithSidebar
		case manualLaunchFlag:
			return ManualLaunch
		case sidebarRestartFlag:
			return SidebarRestart
		}
	}

	// On Windows, we set the KITE_SKIP_ONBOARDING environment variable. This is
	// here for backwards compatability.
	if os.Getenv("KITE_SKIP_ONBOARDING") == "1" {
		return PluginLaunch
	}

	// Check platform specific method here.
	return mode()
}

// GetChannel returns the channel specified during startup, or empty string if none
func GetChannel(args []string) string {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--channel=") {
			parts := strings.Split(arg, "=")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
