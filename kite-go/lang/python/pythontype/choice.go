package pythontype

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// all choice & specificity functions operate on *translated* Values
// i.e. Values internal to analysis are not explicitly accounted for
// there are currently callers that pass untranslated values, but we should fix this (TODO)

// specificity implements the following ordering:
//   instance
//   function
//   class
//   module / package
//   external instance
//   external
//   list / dict / set / tuple
//   scalar
//   none
//   unknown
func specificity(v Value) int {
	switch v.(type) {
	// source
	case *PropertyInstance:
		return 9
	case SourceInstance:
		return 8
	case *SourceFunction:
		return 7
	case *SourceClass:
		return 6
	case *SourcePackage, *SourceModule:
		return 5

	// global
	case ExternalInstance:
		return 4
	case External:
		return 3
	case ExternalRoot:
		return 0

	// constant
	case NoneConstant:
		return 0
	case ConstantValue:
		return 1

	// special cases
	case SuperInstance:
		return 0
	case Union:
		return 0

	default:
		// TODO rollbar.Error(errors.Errorf("specificity: unhandled Value type %T", v))
		return 0
	}
}

// MoreSpecific checks if val is more specific than than.
// if val & than are both External/Instance, the specificity check is deterministic
func MoreSpecific(val, than Value) bool {
	sVal := specificity(val)
	sThan := specificity(than)
	if sVal == sThan {
		switch sVal {
		case 3: // External - see implementation of specificity above
			uSym := val.(External).Symbol()
			vSym := than.(External).Symbol()
			return uSym.Less(vSym)
		case 4: // ExternalInstance
			uSym := val.(ExternalInstance).TypeExternal.Symbol()
			vSym := than.(ExternalInstance).TypeExternal.Symbol()
			return uSym.Less(vSym)
		}
	}
	return sVal < sThan
}

// MostSpecific picks the "most specific" of a set of values.
// This is used to pick one value to expose to clients that do not understand Unions.
func MostSpecific(ctx kitectx.Context, v Value) Value {
	ctx.CheckAbort()
	if _, ok := v.(Union); !ok {
		// if we don't have a union, there's no choice to make
		return v
	}

	disjuncts := Disjuncts(ctx, v)
	var choice Value
	for _, vi := range disjuncts {
		if choice == nil || MoreSpecific(vi, choice) {
			choice = vi
		}
	}
	return choice
}

// ChooseExternal for the provided value
func ChooseExternal(ctx kitectx.Context, rm pythonresource.Manager, val Value) (pythonresource.Symbol, error) {
	ctx.CheckAbort()

	val = Translate(ctx, val, rm)
	if val == nil {
		return pythonresource.Symbol{}, errors.Errorf("translation failed")
	}

	val = MostSpecific(ctx, val)

	var sym pythonresource.Symbol
	switch val := val.(type) {
	case External:
		sym = val.Symbol()
	case ExternalInstance:
		sym = val.TypeExternal.Symbol()
	default:
		return pythonresource.Symbol{}, errors.Errorf("non-external value: %s", String(val))
	}

	return sym, nil
}

// AllExternals returns all external Symbols for the provided value
func AllExternals(ctx kitectx.Context, rm pythonresource.Manager, val Value) []pythonresource.Symbol {
	ctx.CheckAbort()
	if val == nil {
		return nil
	}

	var gvs []GlobalValue

disjuncts:
	for _, dv := range Disjuncts(ctx, val) {
		gv := TranslateGlobal(WidenConstants(dv), rm)
		if gv == nil {
			continue
		}

		for _, g := range gvs {
			if Equal(ctx, g, gv) {
				continue disjuncts
			}
		}

		gvs = append(gvs, gv)
	}

	var syms []pythonresource.Symbol
	for _, gv := range gvs {
		switch gv := gv.(type) {
		case External:
			syms = append(syms, gv.Symbol())
		case ExternalInstance:
			syms = append(syms, gv.TypeExternal.Symbol())
		}
	}
	return syms
}
