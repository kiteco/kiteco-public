package pythontype

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	dictClearAddr      = SplitAddress("builtins.dict.clear")
	dictCopyAddr       = SplitAddress("builtins.dict.copy")
	dictHasKeyAddr     = SplitAddress("builtins.dict.has_key")
	dictGetAddr        = SplitAddress("builtins.dict.get")
	dictPopAddr        = SplitAddress("builtins.dict.pop")
	dictPopitemAddr    = SplitAddress("builtins.dict.popitem")
	dictSetdefaultAddr = SplitAddress("builtins.dict.setdefault")
	dictUpdateAddr     = SplitAddress("builtins.dict.update")
	dictItemsAddr      = SplitAddress("builtins.dict.items")
	dictKeysAddr       = SplitAddress("builtins.dict.keys")
	dictValuesAddr     = SplitAddress("builtins.dict.values")
	dictViewkeysAddr   = SplitAddress("builtins.dict.viewkeys")
	dictViewvaluesAddr = SplitAddress("builtins.dict.viewvalues")
	dictViewitemsAddr  = SplitAddress("builtins.dict.viewitems")
)

// DictLike is an interface representing values for which we track keys that can be used as subscript for index access
// Ex: DictInstance, OrderedDictInstance, pandas.DataFrame
type DictLike interface {
	// GetTrackedKeys return the list of keys tracked for this value
	// These key will be used to provider completion on them for index access
	GetTrackedKeys() map[ConstantValue]Value
}

// DictInstance represents a python instance of builtins.dict
type DictInstance struct {
	Key         Value
	Element     Value
	TrackedKeys map[ConstantValue]Value
}

// NewDict creates a dictionary instance
// A keymap will be automatically added if the key passed as arg is a ConstantValue (IntConstant or StringConstant)
func NewDict(key, value Value) Value {
	result := DictInstance{key, value, nil}
	if constant, ok := key.(ConstantValue); ok {
		switch constant.(type) {
		case IntConstant, StrConstant:
			result.TrackedKeys = make(map[ConstantValue]Value)
			result.TrackedKeys[constant] = value
		}
	}
	return result
}

// NewDictWithMap creates a dictionary and use the keymap argument for it.
// No copy is made so make sure to deep copy the map before calling this function if you don't want it to be shared
// No map is created if nil is passed as argument apart if the key arg is a ConstantValue
// In this case a keymap is created and the key is added to it
func NewDictWithMap(key, value Value, keyMap map[ConstantValue]Value) Value {
	result := DictInstance{key, value, keyMap}
	if constant, ok := key.(ConstantValue); ok {
		switch constant.(type) {
		case IntConstant, StrConstant:
			if keyMap == nil {
				result.TrackedKeys = make(map[ConstantValue]Value)
			}
			result.TrackedKeys[constant] = value
		}
	}
	return result
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v DictInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v DictInstance) Type() Value { return Builtins.Dict }

// Address gets the fully qualified path to this value in the import graph
func (v DictInstance) Address() Address { return Address{} }

// Elem gets the value returned when we iterate over this value,
// i.e the value of `x` in `for x in somedict:...`
func (v DictInstance) Elem() Value { return v.Key }

// Index gets the value returned when we access an element of
// this value at the provided index, i.e the value of `x` in `x = somedict["key"]`
func (v DictInstance) Index(index Value, allowValueMutation bool) Value {
	if cst, ok := index.(ConstantValue); ok {
		switch cst.(type) {
		case IntConstant, StrConstant:
			if value, present := v.TrackedKeys[cst]; present && value != nil {
				return value
			}
			if allowValueMutation && v.TrackedKeys != nil {
				v.TrackedKeys[cst] = nil
			}
		}
	}
	return v.Element
}

// SetIndex returns the new value of the dict that results from setting
// the element at index to the provided value
func (v DictInstance) SetIndex(index Value, value Value, allowValueMutation bool) Value {
	var newTrackedKeys map[ConstantValue]Value
	if cst, ok := index.(ConstantValue); ok {
		switch cst.(type) {
		case IntConstant, StrConstant:
			newTrackedKeys = make(map[ConstantValue]Value, 1)

			if oldValue, ok := v.TrackedKeys[cst]; ok {
				newTrackedKeys[cst] = UniteNoCtx(value, oldValue)
			} else {
				newTrackedKeys[cst] = value
			}
		}
	}
	// widen constants for consistency with evaluating a dict literal
	return NewDictWithMap(WidenConstants(index), WidenConstants(value), newTrackedKeys)
}

// GetTrackedKeys returns all know keys for this dictionary. They can then be used to generate completions
func (v DictInstance) GetTrackedKeys() map[ConstantValue]Value {
	return v.TrackedKeys
}

// attr looks up an attribute on this value
func (v DictInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "clear":
		return SingleResult(BoundMethod{dictClearAddr, v.pyClear}, v), nil
	case "copy":
		return SingleResult(BoundMethod{dictCopyAddr, v.pyCopy}, v), nil
	case "get":
		return SingleResult(BoundMethod{dictGetAddr, v.pyGet}, v), nil
	case "has_key":
		return SingleResult(BoundMethod{dictHasKeyAddr, v.pyHasKey}, v), nil
	case "items":
		return SingleResult(BoundMethod{dictItemsAddr, v.pyItems}, v), nil
	case "keys":
		return SingleResult(BoundMethod{dictKeysAddr, v.pyKeys}, v), nil
	case "pop":
		return SingleResult(BoundMethod{dictPopAddr, v.pyPop}, v), nil
	case "popitem":
		return SingleResult(BoundMethod{dictPopitemAddr, v.pyPopitem}, v), nil
	case "setdefault":
		return SingleResult(BoundMethod{dictSetdefaultAddr, v.pySetdefault}, v), nil
	case "update":
		return SingleResult(BoundMethod{dictUpdateAddr, v.pyUpdate}, v), nil
	case "values":
		return SingleResult(BoundMethod{dictValuesAddr, v.pyValues}, v), nil
	case "viewitems":
		return SingleResult(BoundMethod{dictViewitemsAddr, v.pyViewitems}, v), nil
	case "viewkeys":
		return SingleResult(BoundMethod{dictViewkeysAddr, v.pyViewkeys}, v), nil
	case "viewvalues":
		return SingleResult(BoundMethod{dictViewvaluesAddr, v.pyViewvalues}, v), nil
	default:
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

// equal determines whether this value is equal to another value
func (v DictInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(DictInstance); ok {
		if len(u.TrackedKeys) > len(v.TrackedKeys) {
			return false
		}
		if !(equal(ctx, v.Key, u.Key) && equal(ctx, v.Element, u.Element)) {
			return false
		}
		for k, v1 := range v.TrackedKeys {
			v2, ok := u.TrackedKeys[k]
			if !ok {
				return false
			}
			if !equal(ctx, v1, v2) {
				return false
			}
		}
		return true
	}
	return false
}

// Flatten creates a flat version of this type
func (v DictInstance) Flatten(f *FlatValue, r *Flattener) {
	f.Dict = &FlatDict{r.Flatten(v.Key), r.Flatten(v.Element)}
}

// hash gets a unique ID for this value (used during serialization)
func (v DictInstance) hash(ctx kitectx.CallContext) FlatID {
	keys := make([]Value, 0, len(v.TrackedKeys)+2)
	keys = append(keys, v.Key, v.Element)
	for k := range v.TrackedKeys {
		keys = append(keys, k)
	}
	return rehashValues(ctx, saltDict, keys...)
}

// String provides a string representation of this value
func (v DictInstance) String() string {
	keys := make([]string, 0, len(v.TrackedKeys))
	for k := range v.TrackedKeys {
		keys = append(keys, fmt.Sprint(k))
	}
	sort.Strings(keys)
	return fmt.Sprintf("{%v: %v} [%s]", v.Key, v.Element, strings.Join(keys, ","))
}

func (v DictInstance) pySetdefault(args Args) Value { return v.Element }
func (v DictInstance) pyHasKey(args Args) Value     { return BoolInstance{} }
func (v DictInstance) pyPop(args Args) Value        { return v.Key }
func (v DictInstance) pyPopitem(args Args) Value    { return NewTuple(v.Key, v.Element) }
func (v DictInstance) pyClear(args Args) Value      { return Builtins.None }
func (v DictInstance) pyUpdate(args Args) Value     { return nil }

// these members can be called from python code
func (v DictInstance) pyGet(args Args) Value {
	if len(args.Positional) > 0 {
		arg := args.Positional[0]
		switch a := arg.(type) {
		case IntConstant:
			if val, ok := v.TrackedKeys[a]; ok {
				return val
			}
		case StrConstant:
			if val, ok := v.TrackedKeys[a]; ok {
				return val
			}
		}
	}
	return v.Element
}

func (v DictInstance) pyCopy(args Args) Value {
	result := DictInstance{v.Key, v.Element, make(map[ConstantValue]Value)}
	for k, v := range v.TrackedKeys {
		result.TrackedKeys[k] = v
	}
	return result
}

// TODO: (hrysoula) these now return Views instead of Lists
func (v DictInstance) pyKeys(args Args) Value   { return NewList(v.Key) }
func (v DictInstance) pyValues(args Args) Value { return NewList(v.Element) }
func (v DictInstance) pyItems(args Args) Value  { return NewList(NewTuple(v.Key, v.Element)) }

func (v DictInstance) pyViewkeys(args Args) Value   { return NewList(v.Key) }
func (v DictInstance) pyViewvalues(args Args) Value { return NewList(v.Element) }
func (v DictInstance) pyViewitems(args Args) Value  { return NewList(NewTuple(v.Key, v.Element)) }

// constructDict is the version of the dict constructor that gets called
// directly by the analyzer
func constructDict(args Args) Value {
	if len(args.Positional) != 1 {
		return NewDictWithMap(nil, nil, make(map[ConstantValue]Value))
	}

	if d, ok := args.Positional[0].(DictInstance); ok {
		return NewDictWithMap(d.Key, d.Element, make(map[ConstantValue]Value))
	}

	seq, ok := args.Positional[0].(Iterable)
	if !ok {
		return NewDictWithMap(nil, nil, make(map[ConstantValue]Value))
	}
	key, val := extractKeyVal(seq.Elem())
	return NewDictWithMap(key, val, make(map[ConstantValue]Value))
}

// extractKeyVal extracts the key and element value for
// a dict like object based on the value of elem
func extractKeyVal(elem Value) (Value, Value) {
	var key, val Value
	switch elem := elem.(type) {
	case Indexable:
		// This can cause some weird edge cases
		// e.g for the case: d = dict([{1:0, 2:"hello"}])
		// in python this would be a dict containing the key 1 mapped to the element 2
		// however since we call Index we get the values for the elements in the dict (instead of the key value),
		// thus we get key value <str | int> for the key 0 (as it doesn't exists in the dict)
		// and element value <int> as it match the key 1 stored in the dict
		// see pythonstatic/collections_test.go::TestAssembler_DefaultDict
		// and pythonstatic/collections_test.go::TestAssembler_OrderedDict
		// and pythonstatic/assembler_test.go::TestAssembler_CompoundConstructors
		key = elem.Index(IntConstant(0), false)
		val = elem.Index(IntConstant(1), false)
	case Iterable:
		key = elem.Elem()
		val = key
	}
	return WidenConstants(key), WidenConstants(val)
}
