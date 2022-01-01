package pythontype

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	setAddAddr                       = SplitAddress("builtins.set.add")
	setClearAddr                     = SplitAddress("builtins.set.clear")
	setCopyAddr                      = SplitAddress("builtins.set.copy")
	setDifferenceAddr                = SplitAddress("builtins.set.difference")
	setDifferenceUpdateAddr          = SplitAddress("builtins.set.difference_update")
	setDiscardAddr                   = SplitAddress("builtins.set.discard")
	setIntersectionAddr              = SplitAddress("builtins.set.intersection")
	setIntersectionUpdateAddr        = SplitAddress("builtins.set.intersection_update")
	setIsdisjointAddr                = SplitAddress("builtins.set.isdisjoint")
	setIssubsetAddr                  = SplitAddress("builtins.set.issubset")
	setIssupersetAddr                = SplitAddress("builtins.set.issuperset")
	setPopAddr                       = SplitAddress("builtins.set.pop")
	setRemoveAddr                    = SplitAddress("builtins.set.remove")
	setSymmetricDifferenceAddr       = SplitAddress("builtins.set.symmetric_difference")
	setSymmetricDifferenceUpdateAddr = SplitAddress("builtins.set.symmetric_difference_update")
	setUnionAddr                     = SplitAddress("builtins.set.union")
	setUpdateAddr                    = SplitAddress("builtins.set.update")
)

// SetInstance represents an instance of a builtins.set
type SetInstance struct {
	Element Value
}

// NewSet creates a new set instance
func NewSet(elem Value) Value {
	return SetInstance{elem}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v SetInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v SetInstance) Type() Value { return Builtins.Set }

// Address gets the fully qualified path to this value in the import graph
func (v SetInstance) Address() Address { return Address{} }

// Elem gets the value of elements within this set
func (v SetInstance) Elem() Value { return v.Element }

func (v SetInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "add":
		return SingleResult(BoundMethod{setAddAddr, v.pyAdd}, v), nil
	case "clear":
		return SingleResult(BoundMethod{setClearAddr, v.pyClear}, v), nil
	case "copy":
		return SingleResult(BoundMethod{setCopyAddr, v.pyCopy}, v), nil
	case "difference":
		return SingleResult(BoundMethod{setDifferenceAddr, v.pyDifference}, v), nil
	case "difference_update":
		return SingleResult(BoundMethod{setDifferenceUpdateAddr, v.pyDifferenceUpdate}, v), nil
	case "discard":
		return SingleResult(BoundMethod{setDiscardAddr, v.pyDiscard}, v), nil
	case "intersection":
		return SingleResult(BoundMethod{setIntersectionAddr, v.pyIntersection}, v), nil
	case "intersection_update":
		return SingleResult(BoundMethod{setIntersectionUpdateAddr, v.pyIntersectionUpdate}, v), nil
	case "isdisjoint":
		return SingleResult(BoundMethod{setIsdisjointAddr, v.pyIsdisjoint}, v), nil
	case "issubset":
		return SingleResult(BoundMethod{setIssubsetAddr, v.pyIssubset}, v), nil
	case "issuperset":
		return SingleResult(BoundMethod{setIssupersetAddr, v.pyIssuperset}, v), nil
	case "pop":
		return SingleResult(BoundMethod{setPopAddr, v.pyPop}, v), nil
	case "remove":
		return SingleResult(BoundMethod{setRemoveAddr, v.pyRemove}, v), nil
	case "symmetric_difference":
		return SingleResult(BoundMethod{setSymmetricDifferenceAddr, v.pySymmetricDifference}, v), nil
	case "symmetric_difference_update":
		return SingleResult(BoundMethod{setSymmetricDifferenceUpdateAddr, v.pySymmetricDifferenceUpdate}, v), nil
	case "union":
		return SingleResult(BoundMethod{setUnionAddr, v.pyUnion}, v), nil
	case "update":
		return SingleResult(BoundMethod{setUpdateAddr, v.pyUpdate}, v), nil
	default:
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

// equal determines whether this value is equal to another value
func (v SetInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(SetInstance); ok {
		return equal(ctx, v.Element, u.Element)
	}
	return false
}

// hash gets a unique ID for this value (used during serialization)
func (v SetInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltSet, v.Element)
}

// Flatten creates a flat version of this type
func (v SetInstance) Flatten(f *FlatValue, r *Flattener) {
	f.Set = &FlatSet{r.Flatten(v.Element)}
}

// String provides a string representation of this value
func (v SetInstance) String() string {
	return fmt.Sprintf("{%v}", v.Element)
}

// these members can be called from python code
func (v SetInstance) pyAdd(args Args) Value                       { return nil }
func (v SetInstance) pyClear(args Args) Value                     { return nil }
func (v SetInstance) pyCopy(args Args) Value                      { return nil }
func (v SetInstance) pyDifference(args Args) Value                { return nil }
func (v SetInstance) pyDifferenceUpdate(args Args) Value          { return nil }
func (v SetInstance) pyDiscard(args Args) Value                   { return nil }
func (v SetInstance) pyIntersection(args Args) Value              { return nil }
func (v SetInstance) pyIntersectionUpdate(args Args) Value        { return nil }
func (v SetInstance) pyIsdisjoint(args Args) Value                { return nil }
func (v SetInstance) pyIssubset(args Args) Value                  { return nil }
func (v SetInstance) pyIssuperset(args Args) Value                { return nil }
func (v SetInstance) pyPop(args Args) Value                       { return nil }
func (v SetInstance) pyRemove(args Args) Value                    { return nil }
func (v SetInstance) pySymmetricDifference(args Args) Value       { return nil }
func (v SetInstance) pySymmetricDifferenceUpdate(args Args) Value { return nil }
func (v SetInstance) pyUnion(args Args) Value                     { return nil }
func (v SetInstance) pyUpdate(args Args) Value                    { return nil }

// constructSet is the version of the dict constructor that gets called
// directly by the analyzer
func constructSet(args Args) Value {
	if len(args.Keywords) != 0 {
		return nil
	}
	switch len(args.Positional) {
	case 0:
		return SetInstance{nil}
	case 1:
		if seq, ok := args.Positional[0].(Iterable); ok {
			return NewSet(seq.Elem())
		}
	}
	return nil
}
