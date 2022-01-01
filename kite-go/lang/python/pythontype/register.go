package pythontype

import "github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"

// A map of names to singleton values, so that we can flatten and inflate things like
// Builtin.Super or Builtin.NoneType

var singletons = make(map[pythonimports.Hash]Value)

// newRefModule creates a new module and registers it as a singleton
func newRegModule(addr string, dict map[string]Value) Value {
	mod := NewModule(addr, dict)
	singletons[mod.Address().Path.Hash] = mod
	return mod
}

func newRegType(addr string, ctor func(Args) Value, base Value, dict map[string]Value) ExplicitType {
	typ := NewType(addr, ctor, base, dict)
	singletons[typ.Address().Path.Hash] = typ
	return typ
}

func newRegFunc(addr string, f func(args Args) Value) Value {
	fun := NewFunc(addr, f)
	singletons[fun.Address().Path.Hash] = fun
	return fun
}
