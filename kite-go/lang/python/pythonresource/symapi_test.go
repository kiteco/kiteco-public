package pythonresource

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/symgraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/stretchr/testify/require"
)

func Test_Marshal(t *testing.T) {
	sym := Symbol{}

	buf, err := sym.MarshalBinary()
	require.NoError(t, err)

	sym2 := Symbol{}
	err = sym2.UnmarshalBinary(buf)
	require.NoError(t, err)
	require.EqualValues(t, sym, sym2)
}

func Test_Marshal2(t *testing.T) {
	sym := Symbol{
		Symbol: keytypes.Symbol{
			Dist: keytypes.Distribution{
				Name:    "dist-name",
				Version: "dist-version",
			},
			Path: pythonimports.DottedPath{
				Hash:  1234,
				Parts: []string{"part1", "part2"},
			},
		},
		canonical: keytypes.Symbol{
			Dist: keytypes.Distribution{
				Name:    "dist-name-canonical",
				Version: "dist-version-canonical",
			},
			Path: pythonimports.DottedPath{
				Hash:  1234,
				Parts: []string{"canonical-part1", "canonical-part2", "canonical-part3"},
			},
		},
		ref: symgraph.Ref{
			TopLevel: "path.toplevel",
			Internal: 10,
		},
	}

	buf, err := sym.MarshalBinary()
	require.NoError(t, err)

	sym2 := Symbol{}
	err = sym2.UnmarshalBinary(buf)
	require.NoError(t, err)
	require.EqualValues(t, sym, sym2)
}
