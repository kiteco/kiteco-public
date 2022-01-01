package pythontype

import (
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// KwargDict represents a dictionary created from a **kwargs parameter in a function.
// In python such a value is just a regular dictionary, but if we represent it as an
// ordinary dict then we cannot keep track of the specific value of each of the keyword
// arguments. So instead we use KwargDict, which assumes that all keys are strings and
// tracks values separately for each known key. The implementation contains a
// DictInstance<str, nil> and most calls delegate to this. The only exception is Index(),
// which looks up string constants in the entry map, and SetIndex, which returns nil.
// There are also special implementations of dict.get, dict.pop, and dict.setdefault.
// KwargDict is passed around by pointer rather than by value, which is different to
// DictInstance. The reason for this choice is:
//  - KwargDict contains a map, which makes Equal() and hash() expensive if everything
//    must be compared by value
//  - We only ever construct a *KwargDict when constructing a *Function, which means that
//    each function will give rise to exactly one *KwargDict
type KwargDict struct {
	base    DictInstance     // provides most of the dict implementation
	m       sync.RWMutex     // lock for Entries map
	Entries map[string]Value // Entries contains the real values of this dict
}

// NewKwargDict creates an empty kwarg dictionary
func NewKwargDict() *KwargDict {
	return &KwargDict{
		base:    NewDict(StrInstance{}, nil).(DictInstance),
		Entries: make(map[string]Value),
	}
}

// Add creates an entry for the given keyword argument, or unites the value with the
// existing values if it already exists
func (v *KwargDict) Add(name string, value Value) {
	v.m.Lock()
	defer v.m.Unlock()
	v.Entries[name] = Unite(kitectx.TODO(), v.Entries[name], value)
}

// Index gets the value returned when we access an element of
// this value at the provided index, i.e the value of `x` in `x = somedict["key"]`
func (v *KwargDict) Index(index Value, allowValueMutation bool) Value {
	v.m.RLock()
	defer v.m.RUnlock()
	if key, ok := index.(StrConstant); ok {
		return v.Entries[string(key)]
	}
	return nil
}

// SetIndex returns the new value of the dict that results from setting
// the element at index to the providided value
func (v *KwargDict) SetIndex(index Value, val Value, allowValueMutation bool) Value {
	// for now kwarg dicts are not updateable except by passing parameters to functions
	return nil
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v *KwargDict) Kind() Kind { return v.base.Kind() }

// Type gets the result of calling type() on this value in python
func (v *KwargDict) Type() Value { return v.base.Type() }

// Address gets the fully qualified path to this value in the import graph
func (v *KwargDict) Address() Address { return v.base.Address() }

// Elem gets the value returned when we iterate over this value,
// i.e the value of `x` in `for x in somedict:...`
func (v *KwargDict) Elem() Value { return v.base.Elem() }

// attr looks up an attribute on this value
func (v *KwargDict) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	// here we are overriding just the dict member functions that behave differently
	// for KwargDict (as compared to DictInstance)
	switch name {
	case "copy":
		return SingleResult(BoundMethod{dictCopyAddr, v.pyCopy}, v), nil
	case "get":
		return SingleResult(BoundMethod{dictGetAddr, v.pyGet}, v), nil
	case "pop":
		return SingleResult(BoundMethod{dictPopAddr, v.pyPop}, v), nil
	case "setdefault":
		return SingleResult(BoundMethod{dictSetdefaultAddr, v.pySetdefault}, v), nil
	default:
		return attr(ctx, v.base, name)
	}
}

// equal determines whether this value is equal to another value
func (v *KwargDict) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(*KwargDict); ok {
		return u == v
	}
	return false
}

// Flatten creates a flat version of this type
func (v *KwargDict) Flatten(f *FlatValue, r *Flattener) {
	f.KwargDict = &FlatKwargDict{Entries: flattenDict(v.Entries, r)}
}

// hash gets a unique ID for this value (used during serialization)
func (v *KwargDict) hash(ctx kitectx.CallContext) FlatID {
	return FlatID(uintptr(unsafe.Pointer(v)))
}

// String provides a string representation of this value
func (v *KwargDict) String() string {
	v.m.RLock()
	defer v.m.RUnlock()
	var strs []string
	for k := range v.Entries {
		strs = append(strs, k)
	}
	return fmt.Sprintf("<kwargs: %s>", strings.Join(strs, ","))
}

// pyGet implements dict.get for KwargDict. We handle StrConstant looks in a special way.
func (v *KwargDict) pyGet(args Args) Value {
	if len(args.Positional) == 0 {
		return nil
	}

	var defaultVal Value
	if len(args.Positional) >= 2 {
		defaultVal = args.Positional[1]
	}

	key, ok := args.Positional[0].(StrConstant)
	if !ok {
		return defaultVal
	}
	v.m.RLock()
	defer v.m.RUnlock()
	return Unite(kitectx.TODO(), v.Entries[string(key)], defaultVal)
}

// pyPop implements dict.pop for KwargDict. We handle StrConstant looks in a special way.
func (v *KwargDict) pyPop(args Args) Value {
	return v.pyGet(args)
}

// pySetdefault implements dict.setdefault for KwargDict. We handle StrConstant looks in a special way.
func (v *KwargDict) pySetdefault(args Args) Value {
	return v.pyGet(args)
}

// pyCopy implements dict.copy for KwargDict
func (v *KwargDict) pyCopy(args Args) Value {
	cp := NewKwargDict()
	v.m.RLock()
	defer v.m.RUnlock()
	cp.m.Lock()
	defer cp.m.Unlock()
	for k, v := range v.Entries {
		cp.Entries[k] = v
	}
	return cp
}

// FlatKwargDict is the flat representation of KwargDict
type FlatKwargDict struct {
	Entries []FlatMember
}

// Inflate creates a value from a flat value
func (f FlatKwargDict) Inflate(ctx *Inflater) Value {
	return NewKwargDict()
}

// Link connects a value to other values during unflattening
func (f FlatKwargDict) Link(v Value, ctx *InflateContext) error {
	c := v.(*KwargDict)
	c.Entries = inflateDict(f.Entries, ctx)
	return nil
}
