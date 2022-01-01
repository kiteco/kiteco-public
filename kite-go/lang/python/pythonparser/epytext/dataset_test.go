package epytext

import (
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/internal/testparser"
	"github.com/stretchr/testify/assert"
)

func TestParseDataSet(t *testing.T) {
	testparser.WithDataSet(t, runDataSetCase)
}

func runDataSetCase(t *testing.T, index int, key, src string) {
	t.Run(key, func(t *testing.T) {
		t.Parallel()

		// MaxLines(2000) required for scipy.linalg.cython_lapack
		// MaxExpressions(1e6) required for multiple cases (e.g. OpenGL.GL.NV.path_rendering)
		// If source is whitespace only, then n may be nil (nothing got parsed, e.g. Cheetah.FileUtils.SourceFileStats)
		n, err := Parse([]byte(src), MaxExpressions(1e6), MaxLines(2000))
		if assert.NoError(t, err) {
			if strings.TrimSpace(src) != "" && assert.NotNil(t, n) {
				assert.NotEmpty(t, n.Nodes)
			}
		}
		if t.Failed() {
			// prevent logging too much text
			sz := len(src)
			if sz > 500 {
				src = src[:500]
			}
			t.Logf("index: %d; key: %s\nsource (%d bytes):\n%s\n", index, key, sz, src)
		}
	})
}
