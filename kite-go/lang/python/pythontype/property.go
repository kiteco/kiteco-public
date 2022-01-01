package pythontype

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// PropertyUpdater represents the property().getter/setter/deleter functions,
// typically used as decorators to set the appropriate underlying functions in the property.
// It holds a reference to the underlying PropertyInstance, so we can update it during propagation
type PropertyUpdater struct {
	// Which is one of "getter", "setter", "deleter": the attribute accessed on the PropertyInstance
	Which    string
	Instance PropertyInstance
}

// Kind implements Value
func (p PropertyUpdater) Kind() Kind {
	return FunctionKind
}

// Type implements Value
func (p PropertyUpdater) Type() Value {
	return Builtins.Method
}

// Address implements Value
func (p PropertyUpdater) Address() Address {
	return Address{}
}

// equal implements Value
func (p PropertyUpdater) equal(ctx kitectx.CallContext, other Value) bool {
	if other, ok := other.(PropertyUpdater); ok {
		return p.Which == other.Which && equal(ctx, p.Instance, other.Instance)
	}
	return false
}

// Flatten implements Value
func (p PropertyUpdater) Flatten(f *FlatValue, r *Flattener) {
	f.PropertyUpdater = &FlatPropertyUpdater{
		Which:    p.Which,
		Instance: r.Flatten(p.Instance),
	}
}

// Hash implements Value
func (p PropertyUpdater) hash(ctx kitectx.CallContext) FlatID {
	return rehashBytes(rehashValues(ctx, saltPropertyUpdater, p.Instance), []byte(p.Which))
}

// attr implements Value
func (p PropertyUpdater) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	return AttrResult{}, ErrNotFound
}

// Call implements Callable
func (p PropertyUpdater) Call(args Args) Value {
	// As it stands, we depend on the behavior of Unite to unify the return Value of this Call with the previous instance e.g. without a setter.
	// Since the setter, getter functions are idiomatically given the same name in standard code, this is probably fine, but note that we fail to
	// correctly handle cases where the names are distinct since Unite won't be called.
	// Alternatively to below, we could make *PropertyInstance implement Value, but this complicates things more.
	if len(args.Positional) < 1 {
		return p.Instance
	}

	newF := args.Positional[0]
	switch p.Which {
	case "getter":
		p.Instance.FGet = newF
	case "setter":
		p.Instance.FSet = newF
	case "deleter":
	default:
		panic(fmt.Sprintf("Invalid PropertyUpdater.Which value %s", p.Which))
	}
	return p.Instance
}

// String implements fmt.Stringer
func (p PropertyUpdater) String() string {
	return fmt.Sprintf("%s.%s", p.Instance.String(), p.Which)
}

// PropertyInstance represents an @property
type PropertyInstance struct {
	FGet Value
	FSet Value
	// TODO(naman) FDel, Doc
}

// NewPropertyInstance returns a new property instance
func NewPropertyInstance(fget, fset Value) Value {
	return PropertyInstance{
		FGet: fget,
		FSet: fset,
	}
}

func (p PropertyInstance) source() {}

// Kind implements Value
func (p PropertyInstance) Kind() Kind {
	return DescriptorKind
}

// Type implements Value
func (p PropertyInstance) Type() Value {
	return Builtins.Property
}

// Address implements Value
func (p PropertyInstance) Address() Address {
	return Address{}
}

// equal implements Value
func (p PropertyInstance) equal(ctx kitectx.CallContext, other Value) bool {
	if other, ok := other.(PropertyInstance); ok {
		return equal(ctx, p.FGet, other.FGet) && equal(ctx, p.FSet, other.FSet)
	}
	return false
}

// Flatten implements Value
func (p PropertyInstance) Flatten(f *FlatValue, r *Flattener) {
	f.Property = &FlatProperty{
		FGet: r.Flatten(p.FGet),
		FSet: r.Flatten(p.FSet),
	}
}

// Hash implements Value
func (p PropertyInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltProperty, p.FGet, p.FSet)
}

// attr implements Value
func (p PropertyInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "getter":
		return SingleResult(PropertyUpdater{"getter", p}, p), nil
	case "setter":
		return SingleResult(PropertyUpdater{"setter", p}, p), nil
	case "deleter":
		return SingleResult(PropertyUpdater{"deleter", p}, p), nil
	case "fget":
		return SingleResult(p.FGet, p), nil
	case "fset":
		return SingleResult(p.FSet, p), nil
	default:
		return resolveAttr(ctx, name, p, nil, p.Type())
	}
}

// String implements fmt.Stringer
func (p PropertyInstance) String() string {
	return fmt.Sprintf("%s(%v, %v)", Builtins.Property.Address().String(), p.FGet, p.FSet)
}

func constructProperty(args Args) Value {
	var fget, fset Value
	if len(args.Positional) > 0 {
		fget = args.Positional[0]
	}
	if len(args.Positional) > 1 {
		fset = args.Positional[1]
	}

	for _, kwarg := range args.Keywords {
		switch kwarg.Key {
		case "fget":
			fget = kwarg.Value
		case "fset":
			fset = kwarg.Value
		}
	}

	return NewPropertyInstance(fget, fset)
}
