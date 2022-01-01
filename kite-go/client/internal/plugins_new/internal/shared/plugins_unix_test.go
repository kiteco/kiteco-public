// +build !windows

package shared

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_DedupePaths(t *testing.T) {
	dir, cleanup := SetupTempDir(t, "kite-dedupe")
	defer cleanup()

	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	c := filepath.Join(dir, "c")
	for _, d := range []string{a, b, c} {
		err := os.MkdirAll(d, 0700)
		require.NoError(t, err)
	}

	a1 := filepath.Join(dir, "a1")
	a2 := filepath.Join(dir, "a2")
	for _, d := range []string{a1, a2} {
		err := os.Symlink(a, filepath.Join(d))
		require.NoError(t, err)
	}

	b1 := filepath.Join(dir, "b1")
	b2 := filepath.Join(dir, "b2")
	for _, d := range []string{b1, b2} {
		err := os.Symlink(b, filepath.Join(d))
		require.NoError(t, err)
	}

	// a
	// links a1 and a2 pointing at a
	// b1 and b2 pointing at b
	// c without any links
	deduped := DedupePaths([]string{a, a1, a1, a2, b1, b2, c})
	sort.Strings(deduped)

	require.Len(t, deduped, 3)
	require.EqualValues(t, a, deduped[0])
	require.EqualValues(t, b, deduped[1])
	require.EqualValues(t, c, deduped[2])
}
