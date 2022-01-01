package pythontype

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// It is possible to write python code that generates huge union types, but
// if the number of disjuncts is above this threshold then it is unlikely
// that we were going to get anything reasonable out of it anyway, so we
// just restrict the total union size.
const maxUnionSize = 25

// It is also possible to write python code that generates deeply nested
// union types (lists of dicts of tuples of...) so here we cap the depth
// fairly tightly (since the cost tends to grow exponentially with depth)
const maxUnionDepth = 6

// Union implements the value interface for union types
type Union struct {
	Constituents []Value
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v Union) Kind() Kind {
	return UnionKind
}

// Address gets the fully qualified path to this value in the import graph
func (v Union) Address() Address { return Address{} }

// Type gets the result of calling type() on this value in python
func (v Union) Type() Value {
	switch len(v.Constituents) {
	case 0:
		return nil
	case 1:
		return v.Constituents[0].Type()
	default:
		var types []Value
		for _, vi := range v.Constituents {
			if vi == nil {
				continue
			}
			if t := vi.Type(); t != nil {
				types = append(types, t)
			}
		}
		return Unite(kitectx.TODO(), types...)
	}
}

// Call gets the result of calling this value as though it were a value
func (v Union) Call(args Args) Value {
	switch len(v.Constituents) {
	case 0:
		return nil
	case 1:
		if f, ok := v.Constituents[0].(Callable); ok {
			return f.Call(args)
		}
		return nil
	default:
		var ret []Value
		for _, vi := range v.Constituents {
			if vi == nil {
				continue
			}
			if fun, ok := vi.(Callable); ok {
				if x := fun.Call(args); x != nil {
					ret = append(ret, x)
				}
			}
		}
		return Unite(kitectx.TODO(), ret...)
	}
}

// Elem gets the value obtained by iterating over this value
func (v Union) Elem() Value {
	switch len(v.Constituents) {
	case 0:
		return nil
	case 1:
		if f, ok := v.Constituents[0].(Iterable); ok {
			return f.Elem()
		}
		return nil
	default:
		var elts []Value
		for _, vi := range v.Constituents {
			if vi == nil {
				continue
			}
			if seq, ok := vi.(Iterable); ok {
				if x := seq.Elem(); x != nil {
					elts = append(elts, x)
				}
			}
		}
		return Unite(kitectx.TODO(), elts...)
	}
}

// Index gets the value of the element at index `index`.
func (v Union) Index(index Value, allowValueMutation bool) Value {
	switch len(v.Constituents) {
	case 0:
		return nil
	case 1:
		if f, ok := v.Constituents[0].(Indexable); ok {
			return f.Index(index, allowValueMutation)
		}
		return nil
	default:
		var idxs []Value
		for _, vi := range v.Constituents {
			if vi == nil {
				continue
			}
			if idxable, ok := vi.(Indexable); ok {
				if idx := idxable.Index(index, allowValueMutation); idx != nil {
					idxs = append(idxs, idx)
				}
			}
		}
		return Unite(kitectx.TODO(), idxs...)
	}
}

// SetIndex returns the new value that results from setting the element
// at the provided index to the provided value
func (v Union) SetIndex(index Value, value Value, allowValueMutation bool) Value {
	switch len(v.Constituents) {
	case 0:
		return nil
	case 1:
		if f, ok := v.Constituents[0].(IndexAssignable); ok {
			return f.SetIndex(index, value, allowValueMutation)
		}
		return nil
	default:
		var vs []Value
		for _, vi := range v.Constituents {
			if idxable, ok := vi.(IndexAssignable); ok {
				if v := idxable.SetIndex(index, value, allowValueMutation); v != nil {
					vs = append(vs, v)
				}
			}
		}
		return Unite(kitectx.TODO(), vs...)
	}
}

// attr gets the value of attributes on this object
func (v Union) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch len(v.Constituents) {
	case 0:
		return AttrResult{}, ErrNotFound
	case 1:
		return attr(ctx, v.Constituents[0], name)
	default:
		var vn []ValueNamespace
		for _, vi := range v.Constituents {
			if vi == nil {
				continue
			}
			if r, err := attr(ctx, vi, name); r.Found() {
				if r.ExactlyOne() {
					vn = append(vn, r.Single)
				} else {
					vn = append(vn, r.Multiple...)
				}
			} else if err == ErrTooManySteps {
				return AttrResult{}, ErrTooManySteps
			}
		}
		return UnionResult(vn), nil
	}
}

// equal determines whether this value is equal to another value
// TODO(naman) this is very slow; investigate using hash for equality at the top-level
func (v Union) equal(ctx kitectx.CallContext, u Value) bool {
	// Most unions are small but it is common to contain duplicate values
	// so just do a brute force all-vs-all comparison here. We also want
	// a union containing just one value to be equal to that value on its
	// own.

	if u, ok := u.(Union); ok {
		// unions should always consist of unique elements
		if len(u.Constituents) != len(v.Constituents) {
			return false
		}

	outer:
		for _, ui := range u.Constituents {
			for _, vi := range v.Constituents {
				if equal(ctx, vi, ui) {
					continue outer
				}
			}
			return false
		}
		return true
	}
	return false
}

// hash gets a unique ID for this value (used during serialization)
func (v Union) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltUnion, v.Constituents...)
}

// Flatten creates a flat version of this type
func (v Union) Flatten(f *FlatValue, r *Flattener) {
	ids := make([]FlatID, len(v.Constituents))
	for i, vi := range v.Constituents {
		ids[i] = r.Flatten(vi)
	}
	f.Union = &FlatUnion{ids}
}

// String provides a string representation of this value
func (v Union) String() string {
	var strs []string
	for _, x := range v.Constituents {
		strs = append(strs, fmt.Sprintf("%v", x))
	}
	return "(" + strings.Join(strs, " | ") + ")"
}

// DisjunctsNoCtx gets a list of disjuncts from a union type, none of which are themselves union types
func DisjunctsNoCtx(v Value) []Value {
	return Disjuncts(kitectx.Background(), v)
}

// Disjuncts gets a list of disjuncts from a union type, none of which are themselves union types
func Disjuncts(ctx kitectx.Context, v Value) []Value {
	if v == nil {
		return nil
	}

	u, ok := v.(Union)
	if !ok {
		return []Value{v}
	}
	return u.Constituents
}

// UniteNoCtx computes the union of several types, simplifying where possible
func UniteNoCtx(vs ...Value) Value {
	return Unite(kitectx.Background(), vs...)
}

// Unite computes the union of several types, simplifying where possible
func Unite(ctx kitectx.Context, vs ...Value) Value {
	var val Value
	ctx.WithCallLimit(maxUnionDepth, func(ctx kitectx.CallContext) error {
		val = uniteImpl(ctx, vs...)
		return nil
	})
	return val
}

// uniteImpl is takes care of restricting the depth of nested types (lists, dicts, etc)
func uniteImpl(ctx kitectx.CallContext, vs ...Value) Value {
	if ctx.AtCallLimit() {
		return nil
	}

	// this first switch block is just an optimization for some common cases
	switch len(vs) {
	case 0:
		return nil
	case 1:
		return vs[0]
	case 2:
		if vs[0] == nil {
			return vs[1]
		}
		if vs[1] == nil {
			return vs[0]
		}
	}

	// stack-allocate this array, since it's local to this function, and limited in size
	var disjunctsArr [2 * maxUnionSize]Value
	disjuncts := disjunctsArr[:0]
	addDisjunct := func(v Value) {
		if v == nil {
			return
		}
		var h1, h2 FlatID
		var err error
		vType := reflect.TypeOf(v)
		for _, other := range disjuncts {
			switch reflect.TypeOf(other) {
			case vType:
				if h1 == 0 {
					h1, err = Hash(kitectx.TODO(), v)
					if err != nil {
						// treat error hashing as not equal
						disjuncts = append(disjuncts, v)
						return
					}
				}
				h2, err = Hash(kitectx.TODO(), other)
				if err != nil {
					// treat error hashing as not equal
					break
				}
				if h1 == h2 {
					return
				}
			default:
				continue
			}
		}
		disjuncts = append(disjuncts, v)
	}

	// TODO If performance issue, we can sort vs first and then block in addDisjunct to return when maxUnionSize is reached
	for _, v := range vs {
		if union, ok := v.(Union); ok {
			for _, v := range union.Constituents {
				addDisjunct(v)
			}
		} else {
			addDisjunct(v)
		}
	}
	if len(disjuncts) > maxUnionSize {
		sort.Slice(disjuncts, func(i, j int) bool {
			h1, _ := Hash(kitectx.TODO(), disjuncts[i])
			h2, _ := Hash(kitectx.TODO(), disjuncts[j])
			return h1 < h2
		})
		disjuncts = disjuncts[:maxUnionSize]
	}

	switch len(disjuncts) {
	case 0:
		return nil
	case 1:
		return disjuncts[0]
	default:
		// group all the lists together, all the dicts together, etc
		var properties []PropertyInstance
		var lists []ListInstance
		var dicts []DictInstance
		var sets []SetInstance
		var orderedDicts []OrderedDictInstance
		var dataframes []DataFrameInstance
		var defaultDicts []DefaultDictInstance
		var counters []CounterInstance
		var deques []DequeInstance
		var querySets []QuerySetInstance
		var queues []QueueInstance
		var lifoQueues []LifoQueueInstance
		var priorityQueues []PriorityQueueInstance
		other := make([]Value, 0, len(disjuncts))
		for _, vi := range disjuncts {
			switch vi := vi.(type) {
			case ListInstance:
				lists = append(lists, vi)
			case SetInstance:
				sets = append(sets, vi)
			case OrderedDictInstance:
				orderedDicts = append(orderedDicts, vi)
			case DataFrameInstance:
				dataframes = append(dataframes, vi)
			case DictInstance:
				dicts = append(dicts, vi)
			case DefaultDictInstance:
				defaultDicts = append(defaultDicts, vi)
			case CounterInstance:
				counters = append(counters, vi)
			case DequeInstance:
				deques = append(deques, vi)
			case QuerySetInstance:
				querySets = append(querySets, vi)
			case QueueInstance:
				queues = append(queues, vi)
			case LifoQueueInstance:
				lifoQueues = append(lifoQueues, vi)
			case PriorityQueueInstance:
				priorityQueues = append(priorityQueues, vi)
			case PropertyInstance:
				properties = append(properties, vi)
			default:
				other = append(other, vi)
			}
		}
		// add the united lists, dicts, etc to the other list
		if len(lists) > 0 {
			other = append(other, uniteLists(ctx, lists...))
		}
		if len(dicts) > 0 {
			other = append(other, uniteDicts(ctx, dicts...))
		}
		if len(sets) > 0 {
			other = append(other, uniteSets(ctx, sets...))
		}
		if len(orderedDicts) > 0 {
			other = append(other, uniteOrderedDicts(ctx, orderedDicts...))
		}
		if len(dataframes) > 0 {
			other = append(other, uniteDataFrames(ctx, dataframes...))
		}
		if len(defaultDicts) > 0 {
			other = append(other, uniteDefaultDicts(ctx, defaultDicts...))
		}
		if len(counters) > 0 {
			other = append(other, uniteCounters(ctx, counters...))
		}
		if len(deques) > 0 {
			other = append(other, uniteDeques(ctx, deques...))
		}
		if len(querySets) > 0 {
			other = append(other, uniteQuerySets(ctx, querySets...))
		}
		if len(queues) > 0 {
			other = append(other, uniteQueues(ctx, queues...))
		}
		if len(lifoQueues) > 0 {
			other = append(other, uniteLifoQueues(ctx, lifoQueues...))
		}
		if len(priorityQueues) > 0 {
			other = append(other, unitePriorityQueues(ctx, priorityQueues...))
		}
		if len(properties) > 0 {
			other = append(other, uniteProperties(ctx, properties...))
		}

		if len(other) == 1 {
			return other[0]
		}
		return Union{other}
	}
}

func uniteProperties(ctx kitectx.CallContext, vs ...PropertyInstance) Value {
	var gets, sets []Value
	for _, v := range vs {
		if v.FGet != nil {
			gets = append(gets, v.FGet)
		}
		if v.FSet != nil {
			sets = append(sets, v.FSet)
		}
	}
	return PropertyInstance{
		FGet: uniteImpl(ctx.Call(), gets...),
		FSet: uniteImpl(ctx.Call(), sets...),
	}
}

func uniteLists(ctx kitectx.CallContext, vs ...ListInstance) Value {
	if len(vs) == 1 {
		return vs[0]
	}
	var elems []Value
	for _, v := range vs {
		elems = append(elems, v.Element)
	}
	return NewList(uniteImpl(ctx.Call(), elems...))
}

func uniteDicts(ctx kitectx.CallContext, vs ...DictInstance) Value {
	if len(vs) == 1 {
		return vs[0]
	}
	var keys, elems, keysList []Value
	for _, v := range vs {
		keys = append(keys, v.Key)
		elems = append(elems, v.Element)
		for k := range v.TrackedKeys {
			keysList = append(keysList, k)
		}
	}

	cstKeys := make(map[ConstantValue]Value)
	for _, k := range keysList {
		if cst, ok := k.(ConstantValue); ok {
			var vals []Value
			for _, dict := range vs {
				if v, ok := dict.TrackedKeys[cst]; ok {
					vals = append(vals, v)
				}
			}
			cstKeys[cst] = uniteImpl(ctx.Call(), vals...)
		}
	}
	return NewDictWithMap(uniteImpl(ctx.Call(), keys...), uniteImpl(ctx.Call(), elems...), cstKeys)
}

func uniteDataFrames(ctx kitectx.CallContext, vs ...DataFrameInstance) Value {
	if len(vs) == 1 {
		return vs[0]
	}
	var asDicts []DictInstance
	for _, d := range vs {
		asDicts = append(asDicts, d.delegate)
	}
	dictResult := uniteDicts(ctx, asDicts...).(DictInstance)

	return NewDataFrameInstanceWithMap(dictResult.Key, dictResult.Element, dictResult.TrackedKeys, vs[0].dfType)
}

func uniteOrderedDicts(ctx kitectx.CallContext, vs ...OrderedDictInstance) Value {
	if len(vs) == 1 {
		return vs[0]
	}
	var asDicts []DictInstance
	for _, d := range vs {
		asDicts = append(asDicts, d.delegate)
	}
	dictResult := uniteDicts(ctx, asDicts...).(DictInstance)

	return NewOrderedDictWithMap(dictResult.Key, dictResult.Element, dictResult.TrackedKeys)
}

func uniteSets(ctx kitectx.CallContext, vs ...SetInstance) Value {
	if len(vs) == 1 {
		return vs[0]
	}
	var elems []Value
	for _, v := range vs {
		elems = append(elems, v.Element)
	}
	return NewSet(uniteImpl(ctx.Call(), elems...))
}

func uniteDefaultDicts(ctx kitectx.CallContext, vs ...DefaultDictInstance) Value {
	if len(vs) == 1 {
		return vs[0]
	}
	var keys, elems, factories []Value
	for _, v := range vs {
		keys = append(keys, v.Key)
		elems = append(elems, v.Element)
		factories = append(factories, v.Factory)
	}
	return NewDefaultDict(uniteImpl(ctx.Call(), keys...), uniteImpl(ctx.Call(), elems...), uniteImpl(ctx.Call(), factories...))
}

func uniteDeques(ctx kitectx.CallContext, vs ...DequeInstance) Value {
	if len(vs) == 1 {
		return vs[0]
	}
	var elems []Value
	for _, v := range vs {
		elems = append(elems, v.Element)
	}
	return NewDeque(uniteImpl(ctx.Call(), elems...))
}

func uniteCounters(ctx kitectx.CallContext, vs ...CounterInstance) Value {
	if len(vs) == 1 {
		return vs[0]
	}
	var keys, elems []Value
	for _, v := range vs {
		keys = append(keys, v.Key)
		elems = append(elems, v.Element)
	}
	return NewCounter(uniteImpl(ctx.Call(), keys...), uniteImpl(ctx.Call(), elems...))
}

func uniteQuerySets(ctx kitectx.CallContext, vs ...QuerySetInstance) Value {
	if len(vs) == 1 {
		return vs[0]
	}
	var elems []Value
	for _, v := range vs {
		elems = append(elems, v.Element)
	}
	return NewQuerySet(uniteImpl(ctx.Call(), elems...))
}

func uniteQueues(ctx kitectx.CallContext, vs ...QueueInstance) Value {
	if len(vs) == 1 {
		return vs[0]
	}
	var elems []Value
	for _, v := range vs {
		elems = append(elems, v.Element)
	}
	return NewQueue(uniteImpl(ctx.Call(), elems...))
}

func uniteLifoQueues(ctx kitectx.CallContext, vs ...LifoQueueInstance) Value {
	if len(vs) == 1 {
		return vs[0]
	}
	var elems []Value
	for _, v := range vs {
		elems = append(elems, v.Element)
	}
	return NewLifoQueue(uniteImpl(ctx.Call(), elems...))
}

func unitePriorityQueues(ctx kitectx.CallContext, vs ...PriorityQueueInstance) Value {
	if len(vs) == 1 {
		return vs[0]
	}
	var elems []Value
	for _, v := range vs {
		elems = append(elems, v.Element)
	}
	return NewPriorityQueue(uniteImpl(ctx.Call(), elems...))
}
