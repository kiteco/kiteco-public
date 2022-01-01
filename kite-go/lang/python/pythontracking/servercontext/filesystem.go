package servercontext

import (
	"os"

	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type memFS struct {
	files map[string]*localfiles.File
}

func newMemFS(files []*localfiles.File) memFS {
	fm := make(map[string]*localfiles.File, len(files))
	for _, f := range files {
		fm[f.Name] = f
	}

	return memFS{
		files: fm,
	}
}

// Stat implements localcode.FileSystem
func (m memFS) Stat(path string) (localcode.FileInfo, error) {
	if _, found := m.files[path]; found {
		return localcode.FileInfo{IsDir: false, Size: 100}, nil
	}
	return localcode.FileInfo{}, os.ErrNotExist
}

// Walk implements localcode.FileSystem
func (m memFS) Walk(ctx kitectx.Context, path string, walkFn localcode.WalkFunc) error {
	panic("not implemented")
}

// Glob implements localcode.FileSystem
func (m memFS) Glob(dir, pattern string) ([]string, error) {
	panic("not implemented")
}
