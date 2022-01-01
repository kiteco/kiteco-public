package startup

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMode(t *testing.T) {
	os.Setenv("KITE_SKIP_ONBOARDING", "0")

	var mode = GetMode([]string{})
	assert.Equal(t, ManualLaunch, mode)

	mode = GetMode([]string{systemBootFlag})
	assert.Equal(t, SystemBoot, mode)

	mode = GetMode([]string{relaunchAfterUpdateFlag})
	assert.Equal(t, RelaunchAfterUpdate, mode)

	mode = GetMode([]string{pluginLaunchFlag})
	assert.Equal(t, PluginLaunch, mode)

	mode = GetMode([]string{pluginLaunchWithCopilotFlag})
	assert.Equal(t, PluginLaunchWithSidebar, mode)

	mode = GetMode([]string{sidebarRestartFlag})
	assert.Equal(t, SidebarRestart, mode)
}

func TestGetChannel(t *testing.T) {
	channel := GetChannel([]string{})
	assert.Equal(t, "", channel)

	channel = GetChannel([]string{systemBootFlag})
	assert.Equal(t, "", channel)

	channel = GetChannel([]string{"--channel="})
	assert.Equal(t, "", channel)

	channel = GetChannel([]string{"--channel=acp"})
	assert.Equal(t, "acp", channel)
}

func TestEnvOverride(t *testing.T) {
	os.Setenv("KITE_SKIP_ONBOARDING", "0")
	var mode = GetMode([]string{})
	assert.Equal(t, ManualLaunch, mode)

	//the env var must override the default
	os.Setenv("KITE_SKIP_ONBOARDING", "1")
	mode = GetMode([]string{})
	assert.Equal(t, PluginLaunch, mode)

	//the env var must not override command line flags
	os.Setenv("KITE_SKIP_ONBOARDING", "1")
	mode = GetMode([]string{systemBootFlag})
	assert.Equal(t, SystemBoot, mode)
}

func TestModeString(t *testing.T) {
	assert.Equal(t, "ManualLaunch", ManualLaunch.String())
	assert.Equal(t, "SystemBoot", SystemBoot.String())
	assert.Equal(t, "PluginLaunch", PluginLaunch.String())
	assert.Equal(t, "PluginLaunchWithSidebar", PluginLaunchWithSidebar.String())
	assert.Equal(t, "RelaunchAfterUpdate", RelaunchAfterUpdate.String())
	assert.Equal(t, "SidebarRestart", SidebarRestart.String())
}
