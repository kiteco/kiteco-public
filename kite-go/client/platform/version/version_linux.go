package version

// this is set via "ldflags -X" for release builds
var version = "unknown"

// Version returns a string representation of the current Kite version
func Version() string {
	return version
}

// IsDebugBuild returns true if the build is a debug build
func IsDebugBuild() bool {
	return Version() == "unknown"
}

// IsDevMode returns true if the current build is run in development mode
func IsDevMode() bool {
	return IsDebugBuild()
}
