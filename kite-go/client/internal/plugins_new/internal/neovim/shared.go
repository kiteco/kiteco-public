package neovim

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/kiteco/kiteco/kite-golib/version"
)

const (
	neovimID    = "neovim"
	neovimName  = "Neovim"
	argsVersion = "--version"
	// The default path for all platforms is $HOME/{...}/pack/kite/start/vim-plugin
	// where {...} differs for Windows & UNIX.
	neovimPluginDir        = "pack/kite"
	neovimPluginPathPrefix = "start"
	neovimPluginName       = "vim-plugin"
)

var (
	requiredEditorVersion = version.MustParse("0.2")
	// https://github.com/neovim/neovim/releases/tag/v0.2.0 is the first version which works with our plugin
	minimumMajorVersion = 0
	minimumMinorVersion = 2
	versionMatcher      = regexp.MustCompile(`^(?:NVIM\sv([^\s\r\n]*))|([0-9.]+)`)
	errInvalidBinary    = errors.New("binary does not correspond to a neovim install")
)

// Validate Neovim version.
func validate(output string) error {
	v, err := parseVersion(output)
	if err != nil {
		return err
	}
	if v.Major() >= minimumMajorVersion && v.Minor() >= minimumMinorVersion {
		return nil
	}
	return fmt.Errorf("found version %d.%d, please update to NVIM v0.2.0 or later", v.Major(), v.Minor())
}

// parseVersion extracts the version. It returns an error if the version could not be parsed.
func parseVersion(output string) (version.Info, error) {
	groups := versionMatcher.FindStringSubmatch(output)
	if len(groups) != 3 {
		return version.Info{}, errInvalidBinary
	}
	v := groups[1]
	if v == "" {
		v = groups[2]
	}
	return version.Parse(v)
}
