package localpath

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type newAbsoluteTC struct {
	path          string
	expected      Absolute
	expectedError error
}

func TestNewAbsolute(t *testing.T) {
	tcs := []newAbsoluteTC{
		newAbsoluteTC{
			path:     os.Getenv("GOPATH"),
			expected: Absolute(os.Getenv("GOPATH")),
		},
		newAbsoluteTC{
			path:          "alpha",
			expectedError: ErrPathNotAbsolute,
		},
		newAbsoluteTC{
			path:          filepath.Join("alpha", "beta", "gamma"),
			expectedError: ErrPathNotAbsolute,
		},
	}

	for _, tc := range tcs {
		abs, err := NewAbsolute(tc.path)
		require.Equal(t, tc.expectedError, err)
		require.Equal(t, tc.expected, abs)
	}
}

type isSupportedTC struct {
	base     Relative
	expected bool
}

func TestIsSupported(t *testing.T) {
	tcs := []isSupportedTC{
		isSupportedTC{
			base:     "foo.py",
			expected: true,
		},
		isSupportedTC{
			base:     "foo.go",
			expected: true,
		},
		isSupportedTC{
			base:     "foo.java",
			expected: true,
		},
		isSupportedTC{
			base:     "foo.h",
			expected: true,
		},

		isSupportedTC{
			base:     "foopy",
			expected: false,
		},
		isSupportedTC{
			base:     "foo.csv",
			expected: false,
		},
		isSupportedTC{
			base:     "foo.pl",
			expected: false,
		},
		isSupportedTC{
			base:     "foo.",
			expected: false,
		},
		isSupportedTC{
			base:     "foo",
			expected: false,
		},
	}
	gopath, err := NewAbsolute(os.Getenv("GOPATH"))
	require.NoError(t, err)
	testDir := gopath.Join(
		"src", "github.com", "kiteco", "kiteco",
		"kite-go", "navigation", "offline", "testdata",
	)
	for _, tc := range tcs {
		path := string(testDir.Join(tc.base))
		ext := Extension(filepath.Ext(path))
		require.Equal(t, tc.expected, ext.IsSupported(), tc.base)

		abs := testDir.Join(tc.base)
		require.Equal(t, tc.expected, abs.HasSupportedExtension(), tc.base)
	}
}
