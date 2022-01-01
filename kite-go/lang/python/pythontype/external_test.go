package pythontype

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternal(t *testing.T) {
	intPath := pythonimports.NewPath("builtins", "int")
	m := pythonresource.MockManager(t, map[string]keytypes.TypeInfo{
		"TestExternal.num": keytypes.TypeInfo{Kind: keytypes.ObjectKind, Type: intPath},
	})

	intSym, err := m.PathSymbol(intPath)
	require.NoError(t, err)

	numSym, err := m.PathSymbol(pythonimports.NewPath("TestExternal", "num"))
	require.NoError(t, err)

	v := TranslateExternal(intSym, m)
	assert.Equal(t, "builtins.int", v.Address().Path.String())

	u := TranslateExternal(numSym, m)
	assert.IsType(t, IntInstance{}, u)
}
