package pythonstatic

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
)

func TestBuiltin_Enumerate(t *testing.T) {
	src := `for i, x in enumerate(["a", "b", "c"]): pass`

	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"i": pythontype.IntInstance{},
		"x": pythontype.UniteNoCtx(pythontype.StrConstant("a"), pythontype.StrConstant("b"), pythontype.StrConstant("c")),
	})
}

func TestBuiltin_Map(t *testing.T) {
	src := `out = map(str, [1, 2, 3])`
	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"out": pythontype.NewList(pythontype.StrInstance{}),
	})
}

func TestBuiltin_Sum(t *testing.T) {
	src := `
out = sum(1.1, 2.2, 3.3)
out2 = sum(*[1, 2, 3, 4, 5])
`
	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"out":  pythontype.FloatInstance{},
		"out2": pythontype.IntInstance{},
	})
}

func TestBuiltin_Zip(t *testing.T) {
	src := `
out = zip([1, 2, 3], ["foo", "bar", "baz"])
out2 = zip(*unknown)
`
	assertTypes(t, src, pythonresource.MockManager(t, nil), map[string]pythontype.Value{
		"out":  pythontype.NewList(pythontype.NewTuple(pythontype.IntInstance{}, pythontype.StrInstance{})),
		"out2": pythontype.NewList(pythontype.NewList(nil)),
	})
}
