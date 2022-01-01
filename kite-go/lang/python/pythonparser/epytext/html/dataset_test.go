package html

import (
	"io/ioutil"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/internal/testparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderDataSet(t *testing.T) {
	testparser.WithDataSet(t, runDataSetCase)
}

func runDataSetCase(t *testing.T, index int, key, src string) {
	t.Run(key, func(t *testing.T) {
		t.Parallel()

		n, err := epytext.Parse([]byte(src))
		require.NoError(t, err)

		if n != nil {
			assert.NoError(t, Render(n, ioutil.Discard))
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
