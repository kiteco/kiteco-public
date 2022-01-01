package python

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type indexBundle struct {
	idx   *pythonlocal.SymbolIndex
	graph pythonresource.Manager
	bi    *bufferIndex
}

type valueBundle struct {
	// The underlying value should always be the output of pythontype.Translate
	val pythontype.Value
	indexBundle
}

type symbolBundle struct {
	// The value the symbol holds. It may be a union.
	valueBundle
	// The namespace of this symbol. It should not be a union
	ns valueBundle
	// The name of the symbol in the ns of the attr,
	// this should be used to validate the symbol
	nsName string
	// The name of the symbol that we should render for the editor api
	name string
}

func newValueBundle(ctx kitectx.Context, val pythontype.Value, idx indexBundle) valueBundle {
	val = pythontype.Translate(ctx, val, idx.graph)
	return valueBundle{
		val:         val,
		indexBundle: idx,
	}
}

func (vb valueBundle) bufferIndex() *bufferIndex {
	return vb.bi
}

func (vb valueBundle) valueType(ctx kitectx.Context) valueBundle {
	ctx.CheckAbort()

	if vb.val == nil {
		return valueBundle{}
	}

	return newValueBundle(ctx, vb.val.Type(), vb.indexBundle)
}

func (vb valueBundle) valueModule(ctx kitectx.Context) valueBundle {
	ctx.CheckAbort()

	if vb.val == nil {
		return valueBundle{}
	}
	modID := pythonenv.ModuleLocator(vb.val)
	if modID == "" {
		return valueBundle{}
	}

	var mod pythontype.Value
	if pythonenv.IsLocator(modID) {
		if vb.idx != nil {
			if modVal, err := vb.idx.Locate(ctx, modID); err == nil {
				mod = modVal
			}
		}
	} else {
		if addr, _, err := pythonenv.ParseLocator(modID); err == nil {
			if modSym, err := vb.graph.PathSymbol(addr.Path); err == nil {
				mod = pythontype.NewExternal(modSym, vb.graph)
			}
		}
	}

	if mod == nil {
		return valueBundle{}
	}
	return newValueBundle(ctx, mod, vb.indexBundle)
}

// memberSymbol returns a symbolBundle representing the symbol defined in
// the calling value i.e. the calling value is the namespace of the symbol.
// The input value `v` represents the value the symbol holds and `attr` is
// the name of the symbol in the current namespace and the name of
// the symbol that should be rendered for the editors.
//
// This function is used internally when the symbol's value and name are
// known prior to calling this function.
func (vb valueBundle) memberSymbol(ctx kitectx.Context, v pythontype.Value, attr string) symbolBundle {
	ctx.CheckAbort()

	return symbolBundle{
		valueBundle: newValueBundle(ctx, v, vb.indexBundle),
		ns:          vb,
		nsName:      attr,
		name:        attr,
	}
}

func (vb valueBundle) disjuncts(ctx kitectx.Context) []valueBundle {
	ctx.CheckAbort()

	var vbs []valueBundle
	var bools []pythontype.Value
	for _, v := range pythontype.Disjuncts(ctx, vb.val) {
		switch v.(type) {
		case pythontype.BoolInstance, pythontype.BoolConstant:
			bools = append(bools, v)
		default:
			vbs = append(vbs, newValueBundle(ctx, v, vb.indexBundle))
		}
	}

	if len(bools) > 0 {
		var seenTrue, seenFalse, seenInstance bool
		for _, b := range bools {
			switch b := b.(type) {
			case pythontype.BoolConstant:
				if b {
					seenTrue = true
				} else {
					seenFalse = true
				}
			case pythontype.BoolInstance:
				seenInstance = true
			}
		}

		var b pythontype.Value
		switch {
		case seenInstance:
			b = pythontype.BoolInstance{}
		case seenTrue && seenFalse:
			b = pythontype.BoolInstance{}
		case seenTrue:
			b = pythontype.BoolConstant(true)
		default:
			b = pythontype.BoolConstant(false)
		}
		vbs = append(vbs, newValueBundle(ctx, b, vb.indexBundle))
	}

	return vbs
}
