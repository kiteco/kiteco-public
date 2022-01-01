package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/require"
)

func TestRepo(t *testing.T) {
	kiteco := localpath.Absolute(filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "kiteco", "kiteco"))
	s, err := NewStorage(StorageOptions{})
	require.NoError(t, err)

	var noCache []Commit
	for i := 0; i < 3; i++ {
		repo, err := Open(kiteco, DefaultComputedCommitsLimit, s)
		require.NoError(t, err)

		var batch []Commit
		for j := 0; j < 10; j++ {
			commit, err := repo.Next(kitectx.Background())
			require.NoError(t, err)
			batch = append(batch, commit)
			if i == 0 {
				noCache = append(noCache, commit)
				continue
			}
			require.Equal(t, noCache[j], commit)
		}

		err = repo.Save(s)
		require.NoError(t, err)
	}
}
