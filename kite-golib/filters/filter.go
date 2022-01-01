package filters

import (
	"strings"
)

type pathFilter interface {
	matches(path string) bool
	name() string
}

type containsFilter struct {
	substr string
}

func (c containsFilter) matches(path string) bool {
	return strings.Contains(path, c.substr)
}

func (c containsFilter) name() string {
	return c.substr
}

type startsWithFilter struct {
	prefix string
}

func (s startsWithFilter) matches(path string) bool {
	return strings.HasPrefix(path, s.prefix)
}

func (s startsWithFilter) name() string {
	return s.prefix
}

var defaultFilters = map[string][]pathFilter{
	"darwin": {
		containsFilter{"/Library"},
		containsFilter{"/Applications"},
		containsFilter{"/Pictures"},
		containsFilter{"/.Trash"},
		startsWithFilter{"/dev"},
		startsWithFilter{"/private"},
		startsWithFilter{"/tmp"},
		startsWithFilter{"/opt"},
		startsWithFilter{"/usr"},
		startsWithFilter{"/sbin"},
	},
	"windows": {
		containsFilter{"\\appdata"},
	},
	"linux": {
		containsFilter{"/."},
		startsWithFilter{"/dev"},
		startsWithFilter{"/tmp"},
		startsWithFilter{"/opt"},
		startsWithFilter{"/usr"},
		startsWithFilter{"/sbin"},
		containsFilter{"/node_modules"},
	},
}

var defaultLibraryFilters = []pathFilter{
	containsFilter{"/site-packages"},
	containsFilter{"/dist-packages"},
}

// IsFilteredDir returns true if the provided directory path should
// be excluded from syncing on the provided operating system.
func IsFilteredDir(operatingSystem, path string) bool {
	return GetMatchingFilterName(operatingSystem, path) != ""
}

// GetMatchingFilterName returns the matching directory filter
// for the provided operating system and path. If no filter
// matches, it returns the empty string.
func GetMatchingFilterName(operatingSystem, path string) string {
	patterns := defaultFilters[operatingSystem]
	if patterns == nil {
		return ""
	}
	for _, f := range patterns {
		if operatingSystem == "windows" {
			// windows filters are lower-cased since
			// FromUnix returns lower-cased paths
			if f.matches(strings.ToLower(path)) {
				return f.name()
			}
		} else if f.matches(path) {
			return f.name()
		}
	}
	return ""
}

// IsLibraryDir returns true if the provided directory path includes a library directory.
func IsLibraryDir(path string) bool {
	for _, f := range defaultLibraryFilters {
		if f.matches(path) {
			return true
		}
	}
	return false
}
