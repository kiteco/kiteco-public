package pythontype

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/pkg/errors"
)

// TranslateGlobal translates a Value into a GlobalValue if possible
func TranslateGlobal(val Value, graph pythonresource.Manager) (gv GlobalValue) {
	if val == nil {
		return nil
	}

	var msg string
	defer func() {
		switch {
		case val == nil:
		case gv == nil:
			translateGlobalSuccesRatio.Miss()
			translateGlobalFailures.HitAndAdd(msg)
		default:
			translateGlobalSuccesRatio.Hit()
		}
	}()

	externalByPath := func(path pythonimports.DottedPath) (External, error) {
		if path.Empty() {
			return External{}, errors.New("empty path")
		}
		t, err := graph.PathSymbol(path)
		if err != nil {
			return External{}, errors.New("cannot find symbol for path")
		}
		return NewExternal(t, graph), nil
	}
	instanceByPath := func(path pythonimports.DottedPath) GlobalValue {
		ext, err := externalByPath(path)
		if err != nil {
			return nil
		}
		return ExternalInstance{TypeExternal: ext}
	}
	instanceByPathStr := func(path string) GlobalValue {
		val := instanceByPath(pythonimports.NewDottedPath(path))
		if val == nil {
			panic(fmt.Sprintf("could not find symbol for path %s", path))
		}
		return val
	}

	instanceByAddress := func(addr Address) GlobalValue {
		if addr.Path.Empty() {
			return nil
		}
		return instanceByPath(addr.Path)
	}

	noneInstance := func() GlobalValue {
		return instanceByPathStr("builtins.None.__class__")
	}

	switch val := val.(type) {
	case BoolInstance:
		return instanceByPathStr("builtins.bool")
	case IntInstance:
		return instanceByPathStr("builtins.int")
	case FloatInstance:
		return instanceByPathStr("builtins.float")
	case ComplexInstance:
		return instanceByPathStr("builtins.complex")
	case StrInstance:
		return instanceByPathStr("builtins.str")
	case ListInstance:
		return instanceByPathStr("builtins.list")
	case DictInstance, *KwargDict:
		return instanceByPathStr("builtins.dict")
	case DataFrameInstance:
		return instanceByPathStr(val.Address().Path.String())
	case SetInstance:
		return instanceByPathStr("builtins.set")
	case TupleInstance:
		return instanceByPathStr("builtins.tuple")
	case GeneratorInstance:
		return instanceByPathStr("types.GeneratorType")
	case PropertyUpdater:
		ext, err := externalByPath(pythonimports.NewPath("builtins", "property", val.Which))
		if err != nil {
			return nil
		}
		return ext
	case ExternalRoot:
		return val
	case External:
		return val
	case ExternalInstance:
		return val
	case ExternalReturnValue:
		return val
	case ExplicitModule, ExplicitType, ExplicitFunc, BoundMethod:
		if addr := val.Address(); !addr.Path.Empty() {
			t, err := graph.PathSymbol(addr.Path)
			if err != nil {
				msg = fmt.Sprintf("%T (%s) was not resolved to a valid symbol", val, addr.Path.String())
				return nil
			}
			return NewExternal(t, graph)
		}

		msg = fmt.Sprintf("%T (%s) had no resolvable address", val, String(val))
		return nil
	// collections
	case CounterInstance, OrderedDictInstance,
		DefaultDictInstance, DequeInstance:
		if t := val.Type(); t != nil {
			if n := instanceByAddress(t.Address()); n != nil {
				return n
			}
		}
		return noneInstance()
		// queue
	case QueueInstance, LifoQueueInstance,
		PriorityQueueInstance:

		if t := val.Type(); t != nil {
			if n := instanceByAddress(t.Address()); n != nil {
				return n
			}
		}
		return noneInstance()
	// django
	case QuerySetInstance, ManagerInstance,
		OptionsInstance:
		if t := val.Type(); t != nil {
			if n := instanceByAddress(t.Address()); n != nil {
				return n
			}
		}
		return noneInstance()
	default:
		return nil
	}
}

// TranslateSource translates a Value into a SourceValue if possible
func TranslateSource(val Value) SourceValue {
	srcVal, ok := val.(SourceValue)
	if !ok {
		return nil
	}

	return srcVal
}

// TranslateConstant translates a Value into a ConstantValue if possible
func TranslateConstant(val Value) ConstantValue {
	cnst, ok := val.(ConstantValue)
	if ok {
		return cnst
	}
	return nil
}

// TranslateNoCtx is equivalent to Translate
func TranslateNoCtx(val Value, graph pythonresource.Manager) Value {
	return Translate(kitectx.Background(), val, graph)
}

// Translate translates a Value into a GlobalValue, SourceValue, ConstantValue, or a Union thereof.
func Translate(ctx kitectx.Context, val Value, graph pythonresource.Manager) Value {
	var res Value
	ctx.WithCallLimit(20, func(ctx kitectx.CallContext) error {
		res = translate(ctx, val, graph)
		return nil
	})
	return res
}

func translate(ctx kitectx.CallContext, val Value, graph pythonresource.Manager) Value {
	if ctx.AtCallLimit() || val == nil {
		return nil
	}

	// TODO(juan/naman): for now this seems like the best option
	// since we only create super instances for classes that are from
	// the user's source code atleasat we will be able to link the
	// user to something in their codebase.
	if super, ok := val.(SuperInstance); ok {
		var translated []Value
		for _, base := range super.Bases {
			translated = append(translated, translate(ctx.Call(), base, graph))
		}
		return Unite(ctx.Context, translated...)
	}

	if union, ok := val.(Union); ok {
		var translated []Value
		for _, vi := range union.Constituents {
			translated = append(translated, translate(ctx.Call(), vi, graph))
		}
		return Unite(ctx.Context, translated...)
	}

	if source := TranslateSource(val); source != nil {
		return source
	}
	if cnst := TranslateConstant(val); cnst != nil {
		return cnst
	}
	if global := TranslateGlobal(val, graph); global != nil {
		return global
	}

	return nil

}

// WidenConstant widens ConstantValues into ExternalInstance of the appropiate global type
func WidenConstant(val ConstantValue, graph pythonresource.Manager) GlobalValue {
	switch val.(type) {
	case nil:
		return nil
	case NoneConstant:
		path := pythonimports.NewDottedPath("builtins.None.__class__")
		t, err := graph.PathSymbol(path)
		if err != nil {
			panic("WidenConstant: could not lookup symbol for builtins.None.__class__")
		}
		return ExternalInstance{TypeExternal: NewExternal(t, graph)}
	case BoolConstant:
		return TranslateGlobal(BoolInstance{}, graph)
	case IntConstant:
		return TranslateGlobal(IntInstance{}, graph)
	case FloatConstant:
		return TranslateGlobal(FloatInstance{}, graph)
	case ComplexConstant:
		return TranslateGlobal(ComplexInstance{}, graph)
	case StrConstant:
		return TranslateGlobal(StrInstance{}, graph)
	default:
		panic("unhandled ConstantValue type")
	}
}
