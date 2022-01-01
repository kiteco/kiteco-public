package pythontype

import (
	"fmt"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	tupleCountAddr = SplitAddress("builtins.tuple.count")
	tupleIndexAddr = SplitAddress("builtins.tuple.index")
)

// TupleInstance represents an instance of a builtins.tuple
type TupleInstance struct {
	Elements []Value
}

// NewTuple is the version of the tuple constructor that we call internally
// from Go code
func NewTuple(elts ...Value) Value {
	return TupleInstance{elts}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v TupleInstance) Kind() Kind { return InstanceKind }

// Type gets the value representing the result of calling python's type() on this value
func (v TupleInstance) Type() Value { return Builtins.Tuple }

// Address gets the fully qualified path to this value in the import graph
func (v TupleInstance) Address() Address { return Address{} }

// Elem gets the value of elements within this tuple
func (v TupleInstance) Elem() Value {
	return Unite(kitectx.TODO(), v.Elements...)
}

// Index gets the value of the elements returned when indexing into this tuple
func (v TupleInstance) Index(index Value, allowValueMutation bool) Value {
	switch index := index.(type) {
	case IntConstant:
		i := int(index)
		if i >= 0 && i < len(v.Elements) {
			return v.Elements[i]
		}
	}
	return Unite(kitectx.TODO(), v.Elements...)
}

// SetIndex creates and returns the new value for the tuple that results
// from setting the element at the provided index to the provided value
func (v TupleInstance) SetIndex(index Value, val Value, allowValueMutation bool) Value {
	// return nil since tuples are immutable
	return nil
}

// attr looks up an attribute on this list
func (v TupleInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "count":
		return SingleResult(BoundMethod{tupleCountAddr, v.pyCount}, v), nil
	case "index":
		return SingleResult(BoundMethod{tupleIndexAddr, v.pyIndex}, v), nil
	default:
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

// Copy creates a copy of this tuple
func (v TupleInstance) Copy() Value {
	elts := make([]Value, len(v.Elements))
	for i, vi := range v.Elements {
		elts[i] = vi
	}
	return TupleInstance{elts}
}

// equal determines whether this value is equal to another value
func (v TupleInstance) equal(ctx kitectx.CallContext, u Value) bool {
	tup, ok := u.(TupleInstance)
	if !ok {
		return false
	}
	if len(v.Elements) != len(tup.Elements) {
		return false
	}
	for i := range v.Elements {
		if !equal(ctx, v.Elements[i], tup.Elements[i]) {
			return false
		}
	}
	return true
}

// hash gets a unique ID for this value (used during serialization)
func (v TupleInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltTuple, v.Elements...)
}

// Flatten creates a flat version of this type
func (v TupleInstance) Flatten(f *FlatValue, r *Flattener) {
	ids := make([]FlatID, len(v.Elements))
	for i, vi := range v.Elements {
		ids[i] = r.Flatten(vi)
	}
	f.Tuple = &FlatTuple{ids}
}

// String provides a string representation of this value
func (v TupleInstance) String() string {
	var strs []string
	for _, x := range v.Elements {
		strs = append(strs, fmt.Sprintf("%v", x))
	}
	return "<" + strings.Join(strs, ",") + ">"
}

// these members can be called from python code
func (v TupleInstance) pyCount(args Args) Value { return IntConstant(len(v.Elements)) }
func (v TupleInstance) pyIndex(args Args) Value { return IntInstance{} }

// constructTuple is the version of the dict constructor that gets called
// directly by the analyzer
func constructTuple(args Args) Value {
	if len(args.Keywords) != 0 {
		return nil
	}
	if len(args.Positional) == 1 {
		if tup, ok := args.Positional[0].(TupleInstance); ok {
			return tup.Copy()
		}
		if seq, ok := args.Positional[0].(Iterable); ok {
			// we have no way to represent tuples of indeterminate length so just
			// punt and represent as a list
			return NewList(seq.Elem())
		}
		// if there is only one argument to tuple() and it is not iterable then error
		return nil
	}
	return NewTuple(args.Positional...)
}
