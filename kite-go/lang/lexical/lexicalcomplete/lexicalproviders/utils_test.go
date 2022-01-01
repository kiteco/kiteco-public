package lexicalproviders

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/stretchr/testify/require"
)

func Test_SuppressMultiSelection(t *testing.T) {
	src := `
alpha = beta(gamma=$...$
`
	initModels(t, lexicalmodels.DefaultModelOptions)

	ps := map[string]Provider{
		"./src.go": Text{},
		"./src.js": Text{},
		"./src.py": Python{},
	}
	for filePath, provider := range ps {
		res := requireRes(t, provider, src, filePath)
		require.Empty(t, res.out)
	}
}
