package vim

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/version"
)

const (
	vimID   = "vim"
	vimName = "Vim"
	args    = "--version"
	// The default path for all platforms is $HOME/{...}/pack/kite/start/vim-plugin
	// where {...} differs for Windows & UNIX.
	vimPluginDir        = "pack/kite"
	vimPluginPathPrefix = "start"
	vimPluginName       = "vim-plugin"
)

var (
	requiredEditorVersion = version.MustParse("8.0")
	// https://github.com/vim/vim-win32-installer/releases/tag/v8.0.0027 is the first version which works with our plugin
	minimumRequiredPatchset = 27
	minimumMajorVersion     = 8
	minimumMinorVersion     = 1
	// simple match to avoid problems with translated strings before ": 1-123"
	patchsetMatcher             = regexp.MustCompile(`: (\d+)-(\d+)`)
	errUnableToFindPatchVersion = errors.New("unable to find patch version, please update to VIM 8.0.0027 or later")
	versionMatcher              = regexp.MustCompile(`^VIM - Vi IMproved (\d+\.\d+)`)
	errInvalidBinary            = errors.New("binary does not correspond to a vim install")
)

// Validate Vim version and patch information.
func validate(output string) error {
	v, err := parseVersion(output)
	// vim >= 8.1 is always valid
	// the patchset must only be checked for version 8.0
	if err == nil && v.Major() >= minimumMajorVersion && v.Minor() >= minimumMinorVersion {
		return nil
	}
	patchVersion := getPatchVersion(output)
	switch {
	case patchVersion == -1:
		return errUnableToFindPatchVersion
	case patchVersion < minimumRequiredPatchset:
		return fmt.Errorf("found patch version %d, please update to VIM 8.0.0027 or later", patchVersion)
	default:
		return nil
	}
}

// getPatchVersion from output of vim --version.
func getPatchVersion(output string) int {
	scanner := bufio.NewScanner(strings.NewReader(output))
	lineNo := 1
	for scanner.Scan() && lineNo <= 5 {
		matches := patchsetMatcher.FindStringSubmatch(scanner.Text())
		if len(matches) == 3 {
			latest, err := strconv.Atoi(matches[2])
			if err == nil {
				return latest
			}
			return -1
		}
		lineNo++
	}
	return -1
}

// parseVersion extracts the version. It returns an error if the version could not be parsed.
func parseVersion(output string) (version.Info, error) {
	groups := versionMatcher.FindStringSubmatch(output)
	if len(groups) < 2 {
		return version.Info{}, errInvalidBinary
	}
	return version.Parse(groups[1])
}
