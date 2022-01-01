package testcorpus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/require"
)

const (
	// UserID used for test corpus analysis
	UserID int64 = 1
	// MachineID used for test corpus analysis
	MachineID string = "test"
)

// DoTest calls the provided check function with the builder & local index for the test corpus, and the path of the buffer.
// The buffer path may or may not be included in the local index (so tests should pass in both scenarios).
func DoTest(t *testing.T, subdir string, check func(builder *pythonbatch.BuilderLoader, index *pythonlocal.SymbolIndex, path string)) {
	// Assume that testcorpus directory has at least main.py & buffer.py
	// buffer.py will not be included in the index, while main.py will be the start file for indexing
	// both buffer.py and main.py will be explored, and we attempt to resolve all contained names & attributes
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		t.Fatal("failed to lookup GOPATH")
	}
	corpusDir := filepath.Join(gopath, "src/github.com/kiteco/kiteco/kite-go/lang/python/testcorpus", subdir)
	mainPath := filepath.Join(corpusDir, "main.py")
	bufferPath := filepath.Join(corpusDir, "buffer.py")

	// simulate what Kite Local does for creating the builder
	builder := &pythonbatch.BuilderLoader{
		Graph:   pythonresource.DefaultTestManager(t),
		Options: pythonbatch.DefaultLocalOptions,
	}

	fs := localcode.LocalFileSystem{Include: func(path string, info localcode.FileInfo) bool {
		// exclude non-`.py` files
		if !info.IsDir && !strings.HasSuffix(path, ".py") {
			return false
		}

		// exclude the buffer file
		if path == bufferPath {
			return false
		}

		// exclude files outside the corpus dir
		if rel, err := filepath.Rel(corpusDir, path); err != nil || strings.HasPrefix(rel, "..") {
			return false
		}

		return true
	}}

	// build
	build, err := builder.Build(kitectx.Background(), localcode.BuilderParams{
		UserID:     UserID,
		MachineID:  MachineID,
		Filename:   mainPath,
		FileGetter: fs,
		FileSystem: fs,
		Local:      true,
	})
	require.NoError(t, err)
	index := build.LocalArtifact.(*pythonlocal.SymbolIndex)

	check(builder, index, bufferPath)
	check(builder, index, mainPath)
}
