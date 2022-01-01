package ignore

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/stretchr/testify/require"
)

type ignoreTC struct {
	ignorer  Ignorer
	pathname localpath.Absolute
	isDir    bool
	expected bool
}

func TestIgnore(t *testing.T) {
	tcs := []ignoreTC{
		ignoreTC{
			ignorer: ignorer{
				opts: Options{
					Root: localpath.Absolute(filepath.Join("alpha", "beta")),
				},
				patterns: patternSet{
					simplePattern{
						base:     true,
						sequence: []string{"gamma"},
					},
					simplePattern{
						base:     true,
						sequence: []string{"delta"},
					},
					simplePattern{
						inverted: true,
						base:     true,
						sequence: []string{"epsilon"},
					},
				},
			},
			pathname: localpath.Absolute(filepath.Join("alpha", "beta", "eta", "gamma")),
			expected: true,
		},
	}

	for _, tc := range tcs {
		ignore := tc.ignorer.Ignore(tc.pathname, tc.isDir)
		require.Equal(t, tc.expected, ignore)
	}
}

type ignorerTC struct {
	pathname localpath.Absolute
	isDir    bool
	expected bool
}

func TestIgnorer(t *testing.T) {
	kiteco := localpath.Absolute(filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "kiteco", "kiteco"))

	tcs := []ignorerTC{
		ignorerTC{
			pathname: kiteco.Join("node_modules"),
			isDir:    true,
			expected: true,
		},
		ignorerTC{
			pathname: kiteco.Join("alpha/beta.pyc"),
			expected: true,
		},
		ignorerTC{
			pathname: kiteco.Join("alpha/beta.py"),
			expected: false,
		},
		ignorerTC{
			pathname: kiteco.Join("alpha", "node_modules"),
			isDir:    true,
			expected: true,
		},
		ignorerTC{
			pathname: kiteco.Join("alpha", "osx", "build"),
			isDir:    true,
			expected: false,
		},
		ignorerTC{
			pathname: localpath.Absolute(filepath.Join(os.Getenv("GOPATH"), "alpha")),
			expected: true,
		},
		ignorerTC{
			pathname: kiteco.Join("alpha", "beta", "parse"),
			expected: true,
		},
		ignorerTC{
			pathname: kiteco.Join("alpha", "beta", "parse"),
			isDir:    true,
			expected: false,
		},
		ignorerTC{
			pathname: kiteco,
			isDir:    true,
			expected: false,
		},
	}

	opts := Options{
		Root:            kiteco,
		IgnorePatterns:  []string{".*"},
		IgnoreFilenames: []localpath.Relative{GitIgnoreFilename},
	}
	ignorer, err := New(opts)
	require.NoError(t, err)
	for _, tc := range tcs {
		ignore := ignorer.Ignore(tc.pathname, tc.isDir)
		require.Equal(t, tc.expected, ignore, tc.pathname)
	}
}

func TestShouldRebuild(t *testing.T) {
	kiteco := localpath.Absolute(filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "kiteco", "kiteco"))
	opts := Options{
		Root:            kiteco,
		IgnorePatterns:  []string{".*"},
		IgnoreFilenames: []localpath.Relative{GitIgnoreFilename},
	}
	i, err := newIgnorer(opts)

	require.NoError(t, err)

	before, err := i.ShouldRebuild()

	require.NoError(t, err)
	require.False(t, before)

	gitignoreState := i.fileStates[GitIgnoreFilename]
	i.fileStates[GitIgnoreFilename] = fileState{
		exists:  gitignoreState.exists,
		modTime: gitignoreState.modTime.Add(-time.Second),
	}
	after, err := i.ShouldRebuild()

	require.NoError(t, err)
	require.True(t, after)
}

type hiddenDirectoriesAsFallbackTC struct {
	root     localpath.Absolute
	pathname localpath.Absolute
	isDir    bool
	expected bool
}

func TestHiddenDirectoriesAsFallback(t *testing.T) {
	kiteco := localpath.Absolute(filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "kiteco", "kiteco"))
	tcs := []hiddenDirectoriesAsFallbackTC{
		hiddenDirectoriesAsFallbackTC{
			root:     kiteco,
			pathname: kiteco.Join("alpha", ".beta"),
			isDir:    true,
			expected: false,
		},
		hiddenDirectoriesAsFallbackTC{
			root:     kiteco,
			pathname: kiteco.Join("alpha", ".beta"),
			isDir:    false,
			expected: false,
		},
		hiddenDirectoriesAsFallbackTC{
			root:     kiteco.Join("kite-go"),
			pathname: kiteco.Join("kite-go", "alpha", ".beta"),
			isDir:    true,
			expected: true,
		},
		hiddenDirectoriesAsFallbackTC{
			root:     kiteco.Join("kite-go"),
			pathname: kiteco.Join("kite-go", "alpha", ".beta"),
			isDir:    false,
			expected: false,
		},
	}
	for _, tc := range tcs {
		opts := Options{
			Root:            tc.root,
			IgnorePatterns:  []string{HiddenDirectoriesPattern},
			IgnoreFilenames: []localpath.Relative{GitIgnoreFilename, KiteIgnoreFilename},
		}
		i, err := New(opts)

		require.NoError(t, err)
		require.Equal(t, tc.expected, i.Ignore(tc.pathname, tc.isDir))
	}
}

type cleanRootTC struct {
	root     localpath.Absolute
	pathname localpath.Absolute
	isDir    bool
	expected bool
}

func TestCleanRoot(t *testing.T) {
	tcs := []cleanRootTC{
		cleanRootTC{
			root:     fromSlash("/alpha/"),
			pathname: fromSlash("/alpha"),
			isDir:    true,
			expected: false,
		},
		cleanRootTC{
			root:     fromSlash("/alpha/"),
			pathname: fromSlash("/alpha/"),
			isDir:    true,
			expected: false,
		},
		cleanRootTC{
			root:     fromSlash("/alpha/"),
			pathname: fromSlash("/alph"),
			isDir:    true,
			expected: true,
		},
		cleanRootTC{
			root:     fromSlash("/alpha"),
			pathname: fromSlash("/alpha/"),
			isDir:    true,
			expected: false,
		},
		cleanRootTC{
			root:     fromSlash("/alpha"),
			pathname: fromSlash("/alpha"),
			isDir:    true,
			expected: false,
		},
		cleanRootTC{
			root:     fromSlash("/alpha"),
			pathname: fromSlash("/alph"),
			isDir:    true,
			expected: true,
		},
	}

	for _, tc := range tcs {
		i, err := New(Options{Root: tc.root})
		require.NoError(t, err)
		require.Equal(t, tc.expected, i.Ignore(tc.pathname, tc.isDir))
	}
}

func fromSlash(slashPath string) localpath.Absolute {
	if runtime.GOOS != "windows" {
		return localpath.Absolute(slashPath)
	}
	return localpath.Absolute("C:" + filepath.FromSlash(slashPath))
}
