package vim

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-golib/version"
)

// ValidateExecutable validates the output using validator, extracts the version number with versioner
// and then ensures compatibility by checking against minVersion.
// Used by both Vim and Neovim, which implements its own respective validator funcions.
func ValidateExecutable(output string, executable string, validator func(string) error,
	versioner func(string) (version.Info, error), minVersion version.Info) (system.Editor, error) {
	// found binary but unable to parse output
	if err := validator(output); err != nil {
		return system.Editor{
			Path:          executable,
			Compatibility: fmt.Sprintf("Invalid installation: %v", err),
		}, err
	}

	// found binary but unable to parse version
	v, err := versioner(output)
	if err != nil {
		return system.Editor{
			Path:          executable,
			Compatibility: fmt.Sprintf("Unable to parse version: %v", err),
		}, err
	}

	// invalid version
	if !v.LargerThanOrEqualTo(minVersion) {
		return system.Editor{
			Path:            executable,
			Version:         v.String(),
			Compatibility:   fmt.Sprintf("Version must be >= to %s", minVersion),
			RequiredVersion: minVersion.String(),
		}, err
	}

	// all good
	return system.Editor{
		Path:    executable,
		Version: v.String(),
	}, nil
}
