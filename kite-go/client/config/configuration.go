package config

import (
	"fmt"
	"os"
	"runtime"

	"github.com/kiteco/kiteco/kite-go/client/platform"
)

// Configuration contains build configuration specific options
type Configuration struct {
	Name               string
	RollbarToken       string
	RollbarEnvironment string
	MixpanelToken      string
	CIOSiteID          string
	CIOToken           string
}

var (
	debugConfig = Configuration{
		Name:               "Debug",
		RollbarToken:       "",
		RollbarEnvironment: "",
		MixpanelToken:      "XXXXXXX",
	}

	releaseConfig = Configuration{
		Name:               "Release",
		RollbarToken:       "XXXXXXX",
		RollbarEnvironment: fmt.Sprintf("prod-%s", runtime.GOOS),
		MixpanelToken:      "XXXXXXX",
		CIOSiteID:          "XXXXXXX",
		CIOToken:           "XXXXXXX",
	}
)

// GetConfiguration returns the configuration to use for the current environment
func GetConfiguration(platform *platform.Platform) Configuration {
	// No matter what, debug builds use debug configuration
	if platform.IsDebugBuild || platform.IsUnitTestMode {
		return debugConfig
	}

	// We aren't doing enterprise on windows for now
	if runtime.GOOS == "windows" {
		return releaseConfig
	}

	// Fix Kited startup on Linux
	if runtime.GOOS == "linux" {
		return releaseConfig
	}

	// Only OS X left... these are set by Kite.app
	switch os.Getenv("KITE_CONFIGURATION") {
	case "release":
		return releaseConfig
	case "debug":
		return debugConfig
	}

	// Something weird happened
	panic("unknown configuration")
}
