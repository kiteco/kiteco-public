package shared

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/editor"
	"github.com/kiteco/kiteco/kite-go/client/internal/plugins_new/system"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/stretchr/testify/require"
)

// AlreadyInstalledError is used, when a plugin version shouldn't be installed (again)
var AlreadyInstalledError = errors.Errorf("plugin of the same or a newer version is already installed")

// MapEditors takes a set of home paths and a plugin implementation, which is used to map a path to a system.Editor.
// It returns a set of matching editors found at the given home paths
// MapEditors handles duplicate entries in paths and skips empty paths values.
func MapEditors(ctx context.Context, paths []string, plugin editor.Plugin) []system.Editor {
	deduped := DedupePaths(paths)

	var installs []system.Editor
	for _, path := range deduped {
		result, err := plugin.EditorConfig(ctx, path)
		if err != nil {
			// log.Printf("error retrieving editor properties for %s: %s", path, err.Error())
			continue
		}
		installs = append(installs, result)
	}

	// sort the editor by path for stable results in tools using this
	sort.Slice(installs, func(i, j int) bool {
		return strings.Compare(installs[i].Path, installs[j].Path) < 0
	})
	return installs
}

// StringsContain returns true if the item is contained in the string array.
func StringsContain(data []string, item string) bool {
	for _, d := range data {
		if d == item {
			return true
		}
	}
	return false
}

// MapStrings maps each entry of the input string slice to a new value in the output slice using the mapping function
func MapStrings(data []string, mapping func(item string) string) []string {
	var mapped []string
	for _, d := range data {
		mapped = append(mapped, mapping(d))
	}
	return mapped
}

// DedupePaths resolves symlinks in all the given paths and only retains the unique paths
func DedupePaths(paths []string) []string {
	dedupedPaths := make(map[string]bool, len(paths))
	for _, p := range paths {
		// some functions of plugins_new return empty paths, we're safe to ignore these
		if p == "" {
			continue
		}

		// filepath.EvalSymlinks resolves relative paths based on the working dir
		// we don't want that
		if !filepath.IsAbs(p) {
			dedupedPaths[p] = true
			continue
		}

		if resolved, err := filepath.EvalSymlinks(p); err != nil {
			// keeps paths which could not be handled by EvalSymlinks
			dedupedPaths[p] = true
		} else {
			dedupedPaths[resolved] = true
		}
	}

	var dedupedList []string
	for k := range dedupedPaths {
		dedupedList = append(dedupedList, k)
	}
	return dedupedList
}

// SetupTempDir creates a new temp directory and returns the path and a cleanup function to remove it
func SetupTempDir(t *testing.T, prefix string) (string, func()) {
	dir, err := ioutil.TempDir("", prefix)
	require.NoError(t, err)

	tempDir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	return tempDir, func() {
		os.RemoveAll(tempDir)
	}
}
