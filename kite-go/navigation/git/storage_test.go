package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadLimitExceeded(t *testing.T) {
	opts := StorageOptions{
		UseDisk: true,
		Path: filepath.Join(
			os.Getenv("GOPATH"), "src", "github.com", "kiteco", "kiteco",
			"kite-go", "navigation", "offline", "testdata",
			"astgo.py",
		),
	}
	s, err := NewStorage(opts)
	require.NoError(t, err)
	data, err := s.read(10)
	require.Equal(t, errDataExceedsMaxSize, err)
	require.Nil(t, data)
}
