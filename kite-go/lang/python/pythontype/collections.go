package pythontype

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	counterElementsAddr    = SplitAddress("collections.Counter.elements")
	counterMostCommonAddr  = SplitAddress("collections.Counter.most_common")
	counterSubtractAddr    = SplitAddress("collections.Counter.subtract")
	orderedDictPopItemAddr = SplitAddress("collections.OrderedDict.popitem")
	defaultDictMissingAddr = SplitAddress("collections.defaultdict.__missing__")
	dequeAppendAddr        = SplitAddress("collections.deque.append")
	dequeAppendLeftAddr    = SplitAddress("collections.deque.appendleft")
	dequeClearAddr         = SplitAddress("collections.deque.clear")
	dequeCountAddr         = SplitAddress("collections.deque.count")
	dequeExtendAddr        = SplitAddress("collections.deque.extend")
	dequeExtendLeftAddr    = SplitAddress("collections.deque.extendleft")
	dequePopAddr           = SplitAddress("collections.deque.pop")
	dequePopLeftAddr       = SplitAddress("collections.deque.popleft")
	dequeRemoveAddr        = SplitAddress("collections.deque.remove")
	dequeReverseAddr       = SplitAddress("collections.deque.reverse")
	dequeRotateAddr        = SplitAddress("collections.deque.rotate")
	dequeMaxLenAddr        = SplitAddress("collections.deque.maxlen")

	// These two are fake addresses; they are used across all namedtuple
	// instances though each namedtuple should really get its own address.
	// See NamedTupleInstance.Attr for further discussion.
	namedtupleCountAddr = SplitAddress("<namedtuple>.count")
	namedtupleIndexAddr = SplitAddress("<namedtuple>.index")
)

// Collections contains values representing the members of the collections package
var Collections struct {
	Counter     Value
	OrderedDict Value
	DefaultDict Value
	Deque       Value
	NamedTuple  Value
}

func init() {
	Collections.Counter = newRegType("collections.Counter", constructCounter, Builtins.Dict, map[string]Value{
		"elements":    nil,
		"most_common": nil,
		"subtract":    nil,
	})
	Collections.OrderedDict = newRegType("collections.OrderedDict", constructOrderedDict, Builtins.Dict, map[string]Value{
		"popitem": nil,
	})
	Collections.DefaultDict = newRegType("collections.defaultdict", constructDefaultDict, Builtins.Dict, map[string]Value{
		"__missing__":     nil,
		"default_factory": nil,
	})
	Collections.Deque = newRegType("collections.deque", constructDeque, Builtins.Object, map[string]Value{
		"append":     nil,
		"appendleft": nil,
		"clear":      nil,
		"count":      nil,
		"extend":     nil,
		"extendleft": nil,
		"pop":        nil,
		"popleft":    nil,
		"remove":     nil,
		"reverse":    nil,
		"rotate":     nil,
		"maxlen":     nil,
	})
	Collections.NamedTuple = newRegFunc("collections.namedtuple", constructNamedTupleType)
}

// CounterInstance represents an instance of a collections.Counter
type CounterInstance struct {
	Key     Value
	Element Value
}

// NewCounter creates a collections.Counter instance
func NewCounter(key, value Value) Value {
	return CounterInstance{key, value}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (c CounterInstance) Kind() Kind { return InstanceKind }

// Type gets the results of calling type() on this value in python
func (c CounterInstance) Type() Value { return Collections.Counter }

// Address gets the fully qualified path to this value in the import graph
func (c CounterInstance) Address() Address { return Address{} }

// Elem returns the value that results from iterating over this value.
func (c CounterInstance) Elem() Value { return c.Key }

// Index gets the value returned when indexing into this value
func (c CounterInstance) Index(index Value, allowValueMutation bool) Value { return c.Element }

// SetIndex returns the Counter that results from setting the element at the provided
// index to the provided value
func (c CounterInstance) SetIndex(index Value, value Value, allowValueMutation bool) Value {
	// widen constants for consistency with dict type
	return NewCounter(WidenConstants(index), WidenConstants(value))
}

// attr looks up an attribute on this value
// TODO(juan): attach rest of specialized dict attributes
func (c CounterInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "fromkeys", "update":
		// TODO(juan): these are actually attached to the Counter type in python but they throw and exception when called
		return AttrResult{}, ErrNotFound
	case "elements":
		return SingleResult(BoundMethod{counterElementsAddr, func(args Args) Value { return NewList(c.Element) }}, c), nil
	case "most_common":
		return SingleResult(BoundMethod{counterMostCommonAddr, func(args Args) Value { return NewList(c.Element) }}, c), nil
	case "subtract":
		return SingleResult(BoundMethod{counterSubtractAddr, func(args Args) Value { return Builtins.None }}, c), nil
	default:
		return resolveAttr(ctx, name, c, nil, c.Type())
	}
}

// equal determines whether this value is equal to another value
func (c CounterInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(CounterInstance); ok {
		return equal(ctx, c.Key, u.Key) && equal(ctx, c.Element, u.Element)
	}
	return false
}

// Flatten creates a flat (non recursive) version of this type
func (c CounterInstance) Flatten(f *FlatValue, r *Flattener) {
	f.Counter = &FlatCounter{r.Flatten(c.Key), r.Flatten(c.Element)}
}

// hash gets a unique ID for this value (used during serialization)
func (c CounterInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltCounter, c.Key, c.Element)
}

// String provides a string representation of this value
func (c CounterInstance) String() string {
	return fmt.Sprintf("collections.Counter{%v: %v}", c.Key, c.Element)
}

// constructCounter is the version of the Counter constructor that gets called
// directly by the analyzer
func constructCounter(args Args) Value {
	if len(args.Positional) != 1 {
		return NewCounter(nil, nil)
	}

	switch t := args.Positional[0].(type) {
	case DictInstance:
		return NewCounter(t.Key, t.Element)
	case CounterInstance:
		return NewCounter(t.Key, t.Element)
	case Iterable:
		// widen constants to make consistent with dict type
		return NewCounter(WidenConstants(t.Elem()), IntInstance{})
	default:
		return NewCounter(nil, nil)
	}
}

// FlatCounter is the representation of Counter used for serialization
type FlatCounter struct {
	Key     FlatID
	Element FlatID
}

// Inflate creates a value from a flat value
func (f FlatCounter) Inflate(r *Inflater) Value {
	return NewCounter(r.Inflate(f.Key), r.Inflate(f.Element))
}

// OrderedDictInstance represents an instance of a collections.OrderedDict
type OrderedDictInstance struct {
	delegate DictInstance
}

func (v OrderedDictInstance) hash(ctx kitectx.CallContext) FlatID {
	return v.delegate.hash(ctx)
}

// NewOrderedDict returns an instance of collections.OrderedDict
func NewOrderedDict(key, elem Value) Value {
	return OrderedDictInstance{NewDict(key, elem).(DictInstance)}
}

// NewOrderedDictWithMap returns an instance of collections.OrderedDict with the TrackedKeys maps initialized
func NewOrderedDictWithMap(key, elem Value, keymap map[ConstantValue]Value) Value {
	return OrderedDictInstance{NewDictWithMap(key, elem, keymap).(DictInstance)}
}

// GetTrackedKeys returns the list of known keys for this OrderedDict
func (v OrderedDictInstance) GetTrackedKeys() map[ConstantValue]Value {
	return v.delegate.GetTrackedKeys()
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v OrderedDictInstance) Kind() Kind { return InstanceKind }

// Type gets the results of calling type() on this value in python
func (v OrderedDictInstance) Type() Value { return Collections.OrderedDict }

// Address gets the fully qualified path to this value in the import graph
func (v OrderedDictInstance) Address() Address { return Address{} }

// Elem gets the value that results from iterating over this value
func (v OrderedDictInstance) Elem() Value { return v.delegate.Key }

// SetIndex returns the OrderedDictInstance that results from setting the element at the provided
// index to the provided value
func (v OrderedDictInstance) SetIndex(index Value, value Value, allowValueMutation bool) Value {
	updatedDict := v.delegate.SetIndex(index, value, allowValueMutation).(DictInstance)
	return NewOrderedDictWithMap(updatedDict.Key, updatedDict.Element, updatedDict.TrackedKeys)
}

// attr looks up an attribute on this value
// TODO(juan): attach rest of specialized dict attributes
func (v OrderedDictInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "popitem":
		return SingleResult(BoundMethod{orderedDictPopItemAddr, func(args Args) Value { return NewTuple(v.delegate.Key, v.delegate.Element) }}, v), nil
	default:
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

// equal determines whether this value is equal to another value
func (v OrderedDictInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(OrderedDictInstance); ok {
		return v.delegate.equal(ctx, u.delegate)
	}
	return false
}

// Flatten creates a flat (non recursive) version of this type
func (v OrderedDictInstance) Flatten(f *FlatValue, r *Flattener) {
	f.OrderedDict = &FlatOrderedDict{r.Flatten(v.delegate.Key), r.Flatten(v.delegate.Element)}
}

// String provides a string representation of this value
func (v OrderedDictInstance) String() string {
	keys := make([]string, 0, len(v.delegate.TrackedKeys))
	for k := range v.delegate.TrackedKeys {
		keys = append(keys, fmt.Sprint(k))
	}
	sort.Strings(keys)
	return fmt.Sprintf("collections.OrderedDict{%v: %v, [%s]}", v.delegate.Key, v.delegate.Element, strings.Join(keys, ","))
}

// constructOrderedDict is the version of the OrderedDict constructor that gets called
// directly by the analyzer
func constructOrderedDict(args Args) Value {
	if len(args.Positional) != 1 {
		return NewOrderedDict(nil, nil)
	}

	switch t := args.Positional[0].(type) {
	case OrderedDictInstance:
		return NewOrderedDict(t.delegate.Key, t.delegate.Element)
	case DictInstance:
		return NewOrderedDict(t.Key, t.Element)
	case Iterable:
		key, val := extractKeyVal(t.Elem())
		return NewOrderedDict(key, val)
	default:
		return NewOrderedDict(nil, nil)
	}
}

// FlatOrderedDict is the representation of OrderedDict used for serialization
type FlatOrderedDict struct {
	Key     FlatID
	Element FlatID
}

// Inflate creates a value from a flat value
func (f FlatOrderedDict) Inflate(r *Inflater) Value {
	return NewOrderedDict(r.Inflate(f.Key), r.Inflate(f.Element))
}

// DefaultDictInstance represents an instance of collections.defaultdict
type DefaultDictInstance struct {
	Key     Value
	Element Value
	Factory Value
}

// NewDefaultDict returns a collections.defualtdict instance
func NewDefaultDict(key, element, factory Value) Value {
	return DefaultDictInstance{key, element, factory}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v DefaultDictInstance) Kind() Kind { return InstanceKind }

// Type gets the results of calling type() on this value in python
func (v DefaultDictInstance) Type() Value { return Collections.DefaultDict }

// Address gets the fully qualified path to this value in the import graph
func (v DefaultDictInstance) Address() Address { return Address{} }

// Elem gets the value returned when we iterate over this value
func (v DefaultDictInstance) Elem() Value { return v.Key }

// Index gets the value returned when we index into this value
func (v DefaultDictInstance) Index(index Value, allowValueMutation bool) Value {
	factory, ok := v.Factory.(Callable)
	if !ok {
		return v.Element
	}
	return Unite(kitectx.TODO(), v.Element, factory.Call(Args{}))
}

// SetIndex returns the DefaultDictInstance that results from setting the element at the provided
// index to the provided value
func (v DefaultDictInstance) SetIndex(index Value, value Value, allowValueMutation bool) Value {
	// widen constants for consistency with dict type
	return NewDefaultDict(WidenConstants(index), WidenConstants(value), v.Factory)
}

// attr looks up an attribute on this value
// TODO(juan): attach rest of specialized dict attributes
func (v DefaultDictInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "__missing__":
		return SingleResult(BoundMethod{defaultDictMissingAddr, func(args Args) Value { return v.Element }}, v), nil
	case "default_factory":
		return SingleResult(v.Factory, v), nil
	default:
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

// equal determines whether this value is equal to another value
func (v DefaultDictInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(DefaultDictInstance); ok {
		return equal(ctx, v.Key, u.Key) && equal(ctx, v.Element, u.Element) && equal(ctx, v.Factory, u.Factory)
	}
	return false
}

// Flatten creates a flat version of this type
func (v DefaultDictInstance) Flatten(f *FlatValue, r *Flattener) {
	f.DefaultDict = &FlatDefaultDict{r.Flatten(v.Key), r.Flatten(v.Element), r.Flatten(v.Factory)}
}

// hash gets a unique ID for this value (used during serialization)
func (v DefaultDictInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltDefaultDict, v.Key, v.Element, v.Factory)
}

// String gets a string representation of the value
func (v DefaultDictInstance) String() string {
	return fmt.Sprintf("collections.defaultdict{%v: %v, with factory %v}", v.Key, v.Element, v.Factory)
}

// constructDefaultDict is the version of the collections.defaultdict constructor that
// gets acalled directly by the analyzer
func constructDefaultDict(args Args) Value {
	if len(args.Positional) != 2 {
		return NewDefaultDict(nil, nil, nil)
	}

	_, ok := args.Positional[0].(Callable)
	if !ok {
		return NewDefaultDict(nil, nil, nil)
	}

	switch t := args.Positional[1].(type) {
	case DefaultDictInstance:
		return NewDefaultDict(t.Key, t.Element, args.Positional[0])
	case DictInstance:
		return NewDefaultDict(t.Key, t.Element, args.Positional[0])
	case Iterable:
		key, val := extractKeyVal(t.Elem())
		return NewDefaultDict(key, val, args.Positional[0])
	default:
		return NewDefaultDict(nil, nil, nil)
	}
}

// FlatDefaultDict is the representation of DefaultDict used for serialization
type FlatDefaultDict struct {
	Key     FlatID
	Element FlatID
	Factory FlatID
}

// Inflate creates a value from a flat value
func (f FlatDefaultDict) Inflate(r *Inflater) Value {
	return NewDefaultDict(r.Inflate(f.Key), r.Inflate(f.Element), r.Inflate(f.Factory))
}

// DequeInstance represents an instance of a collections.deque
type DequeInstance struct {
	Element Value
}

// NewDeque creates a collections.deque instance
func NewDeque(elem Value) Value {
	return DequeInstance{elem}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v DequeInstance) Kind() Kind { return InstanceKind }

// Type gets the results of calling type() on this value in python
func (v DequeInstance) Type() Value { return Collections.Deque }

// Address gets the fully qualified pth to this value in the import graph
func (v DequeInstance) Address() Address { return Address{} }

// Elem returns the value that results from iterating over this value
func (v DequeInstance) Elem() Value { return v.Element }

// Index returns the value that results from indexing into this value
func (v DequeInstance) Index(index Value, allowValueMutation bool) Value { return v.Element }

// SetIndex returns the DequeInstance that results from setting the element at the provided
// index to the provided value
func (v DequeInstance) SetIndex(index Value, value Value, allowValueMutation bool) Value {
	return NewDeque(value)
}

// attr looks up an attribute on this value
func (v DequeInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "append":
		return SingleResult(BoundMethod{dequeAppendAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "appendleft":
		return SingleResult(BoundMethod{dequeAppendLeftAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "clear":
		return SingleResult(BoundMethod{dequeClearAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "count":
		return SingleResult(BoundMethod{dequeCountAddr, func(args Args) Value { return IntInstance{} }}, v), nil
	case "extend":
		return SingleResult(BoundMethod{dequeExtendAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "extendleft":
		return SingleResult(BoundMethod{dequeExtendLeftAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "pop":
		return SingleResult(BoundMethod{dequePopAddr, func(args Args) Value { return v.Element }}, v), nil
	case "popleft":
		return SingleResult(BoundMethod{dequePopLeftAddr, func(args Args) Value { return v.Element }}, v), nil
	case "remove":
		return SingleResult(BoundMethod{dequeRemoveAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "reverse":
		return SingleResult(BoundMethod{dequeReverseAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "rotate":
		return SingleResult(BoundMethod{dequeRotateAddr, func(args Args) Value { return Builtins.None }}, v), nil
	case "maxlen":
		return SingleResult(IntInstance{}, v), nil
	default:
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

// equal determines whether this value is equal to another value
func (v DequeInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(DequeInstance); ok {
		return equal(ctx, v.Element, u.Element)
	}
	return false
}

// Flatten creates a flat (non recursive) version of this type
func (v DequeInstance) Flatten(f *FlatValue, r *Flattener) {
	f.Deque = &FlatDeque{r.Flatten(v.Element)}
}

// hash gets a unique ID for this value (used during serialization)
func (v DequeInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltDeque, v.Element)
}

// String provides a string representation of this value
func (v DequeInstance) String() string {
	return fmt.Sprintf("collections.deque{%v}", v.Element)
}

// FlatDeque is the representation of Deque used for serialization
type FlatDeque struct {
	Elem FlatID
}

// Inflate creates a valure from a flat value
func (f FlatDeque) Inflate(r *Inflater) Value {
	return NewDeque(r.Inflate(f.Elem))
}

// constructDeque is the version of the Deque constructor that gets called
// directly by the analyzer
func constructDeque(args Args) Value {
	if len(args.Positional) != 1 {
		return NewDeque(nil)
	}

	if v, ok := args.Positional[0].(Iterable); ok {
		return NewDeque(v.Elem())
	}

	return NewDeque(nil)
}

// NamedTupleType represents a user defined type that was constructed
// via the collections.namedtuple function
type NamedTupleType struct {
	Name Value
	// need to keep ordering of members, for initializing new instances
	Fields []string
}

// NewNamedTupleType constructs a new user defined type
func NewNamedTupleType(name Value, fields []string) Value {
	return NamedTupleType{
		Name:   name,
		Fields: fields,
	}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (t NamedTupleType) Kind() Kind { return TypeKind }

// Type gets the results of calling type() on this value in python
func (t NamedTupleType) Type() Value { return Builtins.Type }

// Address gets the fully qualified path to this value in the import graph
// TODO(Juan): should we set this to be the file the type was defined in, along with the name of the class?
func (t NamedTupleType) Address() Address { return Address{} }

// Call gets the result of calling this value like a function
func (t NamedTupleType) Call(args Args) Value {
	return constructNamedTupleInstance(args, t)
}

// attr looks up an attribute on this value
func (t NamedTupleType) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	for _, field := range t.Fields {
		if field == name {
			return SingleResult(nil, t), nil
		}
	}
	return resolveAttr(ctx, name, t, nil, t.Type(), Builtins.Tuple)
}

// equal determines whether this values is equal to another value
func (t NamedTupleType) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(NamedTupleType); ok {
		if !equal(ctx, t.Name, u.Name) {
			return false
		}

		if len(t.Fields) != len(u.Fields) {
			return false
		}

		for i, field := range t.Fields {
			if field != u.Fields[i] {
				return false
			}
		}
		return true
	}
	return false
}

// Flatten creates a flat (non recursive) version of this type
func (t NamedTupleType) Flatten(f *FlatValue, r *Flattener) {
	f.NamedTupleType = &FlatNamedTupleType{
		Name:   r.Flatten(t.Name),
		Fields: t.Fields,
	}
}

// hash gets a unique ID for this value (used during serialization)
func (t NamedTupleType) hash(ctx kitectx.CallContext) FlatID {
	toHash := []Value{t.Name}
	for field := range t.Fields {
		toHash = append(toHash, StrConstant(strconv.Itoa(field)))
	}
	return rehashValues(ctx, saltNamedTupleType, toHash...)
}

// String provides a string representation of this value
func (t NamedTupleType) String() string {
	return fmt.Sprintf("NamedTupleType{%v: %s}", t.Name, strings.Join(t.Fields, ", "))
}

// FlatNamedTupleType is the representation of NamedTupleType used for serialization
type FlatNamedTupleType struct {
	Name   FlatID
	Fields []string
}

// Inflate creates a value from a flat value
func (f FlatNamedTupleType) Inflate(r *Inflater) Value {
	return NewNamedTupleType(r.Inflate(f.Name), f.Fields)
}

// constructNamedTupleType is called by Collections.NamedTuple to construct a new NamedTupleType
func constructNamedTupleType(args Args) Value {
	if len(args.Positional) < 2 {
		return nil
	}

	name := args.Positional[0]

	var fields []string
	switch f := args.Positional[1].(type) {
	case StrConstant:
		if strings.Contains(string(f), ",") {
			fields = strings.Split(string(f), ",")
		} else {
			fields = strings.Split(string(f), " ")
		}
	case TupleInstance:
		for _, elem := range f.Elements {
			elem, ok := elem.(StrConstant)
			if !ok {
				return nil
			}
			fields = append(fields, string(elem))
		}
	default:
		// TODO(juan): cant support lists since we do not
		// track list constants
		return nil
	}

	for i, field := range fields {
		fields[i] = strings.TrimSpace(field)
	}

	return NewNamedTupleType(name, fields)
}

// NamedTupleInstance represents an instance of a user defined class from collections.namedtuple
type NamedTupleInstance struct {
	Typ    NamedTupleType // make sure all instances reference a single instance of the type
	Fields []Value
}

// NewNamedTupleInstance returns an instance of a user defined type
func NewNamedTupleInstance(typ NamedTupleType, fields []Value) Value {
	if len(fields) != len(typ.Fields) {
		panic(fmt.Sprintf("instance of namedtuple %v has %d fields, need %d", typ, len(fields), len(typ.Fields)))
	}
	return NamedTupleInstance{
		Typ:    typ,
		Fields: fields,
	}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v NamedTupleInstance) Kind() Kind { return InstanceKind }

// Type gets the value representing the result of calling type() on this value in python
func (v NamedTupleInstance) Type() Value { return v.Typ }

// Address gets the fully qualified path to this value in the import graph
func (v NamedTupleInstance) Address() Address { return Address{} }

// Elem gets the value that results from iterating over this value
func (v NamedTupleInstance) Elem() Value {
	var vals []Value
	for _, field := range v.Fields {
		vals = append(vals, field)
	}
	return Unite(kitectx.TODO(), vals...)
}

// Index gets the value of the element returned when indexing into this value
func (v NamedTupleInstance) Index(index Value, allowValueMutation bool) Value {
	i := -1
	switch index := index.(type) {
	case IntConstant:
		i = int(index)
	}
	if i >= 0 && i < len(v.Fields) {
		return v.Fields[i]
	}
	return v.Elem()
}

// attr looks up an attribute on this value
func (v NamedTupleInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	// Here we use namedtupleIndexAddr and namedtupleCountAddr as
	// addresses for the bound methods. This is not quite correct
	// because each NamedTupleType is a unique, dynamically created
	// type with no address, so we should really have a unique address
	// per namedtuple. However, we must assign _some_ address or else
	// BoundMethod.hash() will panic. Giving all the bound methods the
	// same address means that they all get the same hash, which means
	// that a union will never contain bound methods from more than
	// one namedtuple.
	//
	// Note that namedtuple is unique in our type system as the only
	// dynamically constructed type that we model explicitly, so this
	// problem is also unique to namedtuple.

	switch name {
	case "count":
		return SingleResult(BoundMethod{namedtupleCountAddr, func(Args) Value { return IntConstant(len(v.Fields)) }}, v), nil
	case "index":
		return SingleResult(BoundMethod{namedtupleIndexAddr, func(Args) Value { return IntInstance{} }}, v), nil
	default:
		for i, field := range v.Typ.Fields {
			if name == field {
				return SingleResult(v.Fields[i], v), nil
			}
		}
		return resolveAttr(ctx, name, v, nil, v.Type())
	}
}

// equal determines whether this value is equal to another value
func (v NamedTupleInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(NamedTupleInstance); ok {
		if !equal(ctx, u.Typ, v.Typ) {
			return false
		}

		if len(u.Fields) != len(v.Fields) {
			return false
		}

		for i, val := range v.Fields {
			if !equal(ctx, val, u.Fields[i]) {
				return false
			}
		}
		return true
	}
	return false
}

// hash gets a unique ID for this value (used during serialization)
func (v NamedTupleInstance) hash(ctx kitectx.CallContext) FlatID {
	args := []Value{v.Typ}
	args = append(args, v.Fields...)
	return rehashValues(ctx, saltNamedTupleInstance, args...)
}

// Flatten creates a flat version of this value (used during serialization)
func (v NamedTupleInstance) Flatten(f *FlatValue, r *Flattener) {
	var ffields []FlatID
	for _, field := range v.Fields {
		ffields = append(ffields, r.Flatten(field))
	}
	f.NamedTupleInstance = &FlatNamedTupleInstance{
		Typ:    r.Flatten(v.Typ),
		Fields: ffields,
	}
}

// constructNamedTupleInstance is called when NamedTupleType is used to instantiate a new instance of
// a user defined class
func constructNamedTupleInstance(args Args, typ NamedTupleType) Value {
	switch len(typ.Fields) {
	case len(args.Positional):
		var fields []Value
		for _, val := range args.Positional {
			fields = append(fields, val)
		}
		return NewNamedTupleInstance(typ, fields)
	case len(args.Keywords):
		var fields []Value
		for _, field := range typ.Fields {
			val, found := args.Keyword(field)
			if !found {
				return nil
			}
			fields = append(fields, val)
		}
		return NewNamedTupleInstance(typ, fields)
	default:
		if args.HasVararg && args.HasKwarg {
			// ignore this case because we currently do not support
			// explicit representations of list and map literals so
			// we will not be able to extract reasonable values for
			// the names of the fields.
			return nil
		}

		// TODO(juan): if either *args or **kwargs are mappable
		// then each member will simply be a union of all types in the mappable
		var idxable Indexable
		var ok bool
		if args.HasVararg {
			idxable, ok = args.Vararg.(Indexable)
		}
		if args.HasKwarg {
			idxable, ok = args.Kwarg.(Indexable)
		}
		if !ok {
			return nil
		}

		var fields []Value
		for i := range typ.Fields {
			val := idxable.Index(IntConstant(i), false)
			if val == nil {
				return nil
			}
			fields = append(fields, val)
		}
		return NewNamedTupleInstance(typ, fields)
	}
}

// FlatNamedTupleInstance is the representation of NamedTupleInstance used for serialization
type FlatNamedTupleInstance struct {
	Typ    FlatID
	Fields []FlatID
}

// Inflate creates a value from a flat value
func (f FlatNamedTupleInstance) Inflate(r *Inflater) Value {
	typ := r.Inflate(f.Typ).(NamedTupleType)
	var fields []Value
	for _, field := range f.Fields {
		fields = append(fields, r.Inflate(field))
	}
	return NewNamedTupleInstance(typ, fields)
}
