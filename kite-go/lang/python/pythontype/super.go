package pythontype

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// SuperInstance represents an instance of builtins.super
type SuperInstance struct {
	Bases    []Value
	Instance Value
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v SuperInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v SuperInstance) Type() Value { return Builtins.Super }

// Address gets the fully qualified path to this value in the import graph
func (v SuperInstance) Address() Address { return Address{} }

// String gets a string representation of this value
func (v SuperInstance) String() string { return fmt.Sprintf("<super: %v>", v.Bases) }

// hash gets a unique ID for this value (for serialization)
func (v SuperInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltSuper, append(v.Bases, v.Instance)...)
}

// Flatten creates a non-recursive version of this type for serialization
func (v SuperInstance) Flatten(f *FlatValue, r *Flattener) {
	var bases []FlatID
	for _, b := range v.Bases {
		bases = append(bases, r.Flatten(b))
	}
	f.Super = &FlatSuper{
		Bases:    bases,
		Instance: r.Flatten(v.Instance),
	}
}

// attr gets the value of an attribute. Steps is a counter to avoid infinite loops.
func (v SuperInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	// TODO(alex): this won't work with ExplicitTypes that implement bound methods at
	// the instance level. But it will work with user-defined code and import graph
	// nodes, which is the vast majority of cases.

	// Search for attribute in base classes
	for _, b := range v.Bases {
		if b == nil {
			continue
		}
		if res, err := attr(ctx, b, name); res.Found() {
			return res, nil
		} else if err == ErrTooManySteps {
			return AttrResult{}, ErrTooManySteps
		}
	}
	return AttrResult{}, ErrNotFound
}

// equal determines whether this value is equal to another value
func (v SuperInstance) equal(ctx kitectx.CallContext, u Value) bool {
	su, eq := u.(SuperInstance)
	if !eq {
		return false
	}

	if !equal(ctx, v.Instance, su.Instance) {
		return false
	}

	for _, bv := range v.Bases {
		var ok bool
		for _, bu := range su.Bases {
			if equal(ctx, bv, bu) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	return true
}

// NewSuper constructs a SuperInstance for the given type and instance.
// In python 3, the constructor for super cannot be implemented as
// an ordinary function because it recieves no arguments. Instead
// super is a special case in evaluator.go, which gets the enclosing
// type to and calls this function.
func NewSuper(bases []Value, instance Value) SuperInstance {
	return SuperInstance{Bases: bases, Instance: instance}
}
