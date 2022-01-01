package pythontype

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	listAppendAddr  = SplitAddress("builtins.list.append")
	listClearAddr   = SplitAddress("builtins.list.clear")
	listCopyAddr    = SplitAddress("builtins.list.copy")
	listCountAddr   = SplitAddress("builtins.list.count")
	listExtendAddr  = SplitAddress("builtins.list.extend")
	listIndexAddr   = SplitAddress("builtins.list.index")
	listInsertAddr  = SplitAddress("builtins.list.insert")
	listPopAddr     = SplitAddress("builtins.list.pop")
	listRemoveAddr  = SplitAddress("builtins.list.remove")
	listReverseAddr = SplitAddress("builtins.list.reverse")
	listSortAddr    = SplitAddress("builtins.list.sort")
)

// ListInstance represents an instance of a builtins.list
type ListInstance struct {
	Element Value
}

// NewList creates a list instance
func NewList(elem Value) Value {
	return ListInstance{elem}
}

// Kind gets the kind of this value
func (v ListInstance) Kind() Kind { return InstanceKind }

// Type gets the value representing the result of calling python's type() on this value
func (v ListInstance) Type() Value { return Builtins.List }

// Address gets the fully qualified path to this value in the import graph
func (v ListInstance) Address() Address { return Address{} }

// Elem gets the value of elements within this list
func (v ListInstance) Elem() Value { return v.Element }

// Index gets the value of the elements returned when indexing into this list
func (v ListInstance) Index(index Value, allowValueMutation bool) Value { return v.Element }

// SetIndex returns the new value of the list obtained after setting the value of
// of the element at the provided index to the provided value
func (v ListInstance) SetIndex(index Value, value Value, allowValueMutation bool) Value {
	return NewList(Unite(kitectx.TODO(), v.Element, value))
}

// attr looks up an attribute on this value
func (v ListInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "append":
		return SingleResult(BoundMethod{listAppendAddr, v.pyAppend}, v), nil
	case "clear":
		return SingleResult(BoundMethod{listClearAddr, v.pyClear}, v), nil
	case "copy":
		return SingleResult(BoundMethod{listCopyAddr, v.pyCopy}, v), nil
	case "count":
		return SingleResult(BoundMethod{listCountAddr, v.pyCount}, v), nil
	case "extend":
		return SingleResult(BoundMethod{listExtendAddr, v.pyExtend}, v), nil
	case "index":
		return SingleResult(BoundMethod{listIndexAddr, v.pyIndex}, v), nil
	case "insert":
		return SingleResult(BoundMethod{listInsertAddr, v.pyInsert}, v), nil
	case "pop":
		return SingleResult(BoundMethod{listPopAddr, v.pyPop}, v), nil
	case "remove":
		return SingleResult(BoundMethod{listRemoveAddr, v.pyRemove}, v), nil
	case "reverse":
		return SingleResult(BoundMethod{listReverseAddr, v.pyReverse}, v), nil
	case "sort":
		return SingleResult(BoundMethod{listSortAddr, v.pySort}, v), nil
	default:
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

// equal determines whether this value is equal to another value
func (v ListInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(ListInstance); ok {
		return equal(ctx, v.Element, u.Element)
	}
	return false
}

// Flatten creates a flat version of this type
func (v ListInstance) Flatten(f *FlatValue, r *Flattener) {
	f.List = &FlatList{r.Flatten(v.Element)}
}

// hash gets a unique ID for this value (used during serialization)
func (v ListInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltList, v.Element)
}

// String provides a string representation of this value
func (v ListInstance) String() string {
	return fmt.Sprintf("[%v]", v.Element)
}

// these members can be called from python code
func (v ListInstance) pyAppend(args Args) Value  { return Builtins.None }
func (v ListInstance) pyCount(args Args) Value   { return IntInstance{} }
func (v ListInstance) pyExtend(args Args) Value  { return Builtins.None }
func (v ListInstance) pyIndex(args Args) Value   { return IntInstance{} }
func (v ListInstance) pyInsert(args Args) Value  { return Builtins.None }
func (v ListInstance) pyPop(args Args) Value     { return v.Element }
func (v ListInstance) pyRemove(args Args) Value  { return Builtins.None }
func (v ListInstance) pyReverse(args Args) Value { return Builtins.None }
func (v ListInstance) pySort(args Args) Value    { return Builtins.None }
func (v ListInstance) pyClear(args Args) Value   { return Builtins.None }
func (v ListInstance) pyCopy(args Args) Value    { return NewList(v.Element) }

// constructList is the version of the dict constructor that gets called
// directly by the analyzer
func constructList(args Args) Value {
	switch len(args.Positional) {
	case 0:
		return NewList(nil)
	case 1:
		if v, ok := args.Positional[0].(Iterable); ok {
			return NewList(v.Elem())
		}
	}
	return NewList(nil)
}
