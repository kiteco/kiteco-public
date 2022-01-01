package python

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
)

func TestRenderValueRepr(t *testing.T) {
	manager := pythonresource.MockManager(t, nil, "matplotlib.pyplot", "json.dumps", "xml.Encoder")
	s := editorServices{&Services{
		ResourceManager: manager,
	}}

	repr := func(v pythontype.Value) string {
		return s.renderValueRepr(kitectx.Background(), valueBundle{
			val: v,
			indexBundle: indexBundle{
				graph: manager,
			},
		})
	}

	// explcitly modeled values
	assert.Equal(t, "float", repr(pythontype.FloatInstance{}))
	assert.Equal(t, "123", repr(pythontype.IntConstant(123)))
	assert.Equal(t, "\"abc\" | int", repr(pythontype.UniteNoCtx(pythontype.IntInstance{}, pythontype.StrConstant("abc"))))

	// global import graph nodes
	pyplotSym, err := manager.PathSymbol(pythonimports.NewPath("matplotlib", "pyplot"))
	assert.NoError(t, err)

	pyplot := pythontype.NewExternal(pyplotSym, manager)

	dumpsSym, err := manager.PathSymbol(pythonimports.NewPath("json", "dumps"))
	assert.NoError(t, err)
	dumps := pythontype.NewExternal(dumpsSym, manager)

	assert.Equal(t, "matplotlib.pyplot", repr(pyplot))
	assert.Equal(t, "json.dumps", repr(dumps))

	// instances of global import graph nodes
	encoderSym, err := manager.PathSymbol(pythonimports.NewPath("xml", "Encoder"))
	encoder := pythontype.ExternalInstance{TypeExternal: pythontype.NewExternal(encoderSym, manager)}
	assert.Equal(t, "xml.Encoder", repr(encoder))

	mt := pythontype.NewSymbolTable(pythontype.Address{File: "/usr/src/code.py"}, nil)

	// user-defined modules
	srcmod := &pythontype.SourceModule{
		Members: mt,
	}
	assert.Equal(t, "code", repr(srcmod))

	// user-defined function
	ft := pythontype.NewSymbolTable(pythontype.SplitAddress("SomeClass.SomeFunc"), nil)
	ft.Name.File = "/usr/src/code.py"
	srcfunc := &pythontype.SourceFunction{
		Locals: ft,
	}

	assert.Equal(t, "SomeClass.SomeFunc", repr(srcfunc))
}
