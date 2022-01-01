package pythontype

import "github.com/kiteco/kiteco/kite-golib/kitectx"

// WidenConstants converts constants to their broader instance representations, and leaves
// other values unchanged.
func WidenConstants(v Value) Value {

	// TODO(naman) we should either accept a kitectx.Context here, or, preferably,
	// Unite should automatically widen multiple constants of the same type:
	// i.e. Unite(IntConstant(1), IntConstant(2)) should become just IntInstance.

	// The latter approach guarantees that widening all the constituents of a Union
	// results in another Union of the same "shape," which would allow us to process
	// the Union case here non-recursively, with no need for a kitectx.Context.

	switch v := v.(type) {
	case BoolConstant:
		return BoolInstance{}
	case IntConstant:
		return IntInstance{}
	case FloatConstant:
		return FloatInstance{}
	case ComplexConstant:
		return ComplexInstance{}
	case StrConstant:
		return StrInstance{}
	case Union:
		var wide []Value
		for _, vi := range Disjuncts(kitectx.TODO(), v) {
			wide = append(wide, WidenConstants(vi))
		}
		return Unite(kitectx.TODO(), wide...)
	default:
		return v
	}
}

// pyEnumerate implements the python builtins.enumerate function
func pyEnumerate(args Args) Value {
	// the result of enumerate(x) is list<tuple<int, elem(x)>>.
	var elem Value
	if len(args.Positional) == 1 {
		if seq, ok := args.Positional[0].(Iterable); ok {
			elem = seq.Elem()
		}
	}

	// normally the index is an integer but it can also be specified manually with
	// the "start" keyword
	var index Value
	index = IntInstance{}
	if start, found := args.Keyword("start"); found {
		index = Unite(kitectx.TODO(), index, WidenConstants(start))
	}

	return NewList(NewTuple(index, elem))
}

// pyMap implements the python builtins.map function
// TODO: (hrysoula) map now returns an iterator in py3, should anything change here?
func pyMap(args Args) Value {
	if len(args.Positional) < 2 {
		return NewList(nil)
	}

	fun, ok := args.Positional[0].(Callable)
	if !ok {
		return NewList(nil)
	}

	if len(args.Positional) == 2 {
		seq, ok := args.Positional[1].(Iterable)
		if !ok {
			return NewList(nil)
		}
		return NewList(fun.Call(Positional(seq.Elem())))
	}
	var elems []Value
	for _, arg := range args.Positional[1:] {
		if seq, ok := arg.(Iterable); ok {
			elems = append(elems, seq.Elem())
		} else {
			elems = append(elems, nil)
		}
	}
	return NewList(fun.Call(Positional(NewTuple(elems...))))
}

// pySum implements the python builtins.sum function
func pySum(args Args) Value {
	// The result of sum(x1, ..., xn) can be any of the types of the parameters,
	// except for constants, which should be replaced by their widened counterparts.
	var vs []Value
	for _, v := range args.Positional {
		vs = append(vs, WidenConstants(v))
	}
	if start, found := args.Keyword("start"); found {
		vs = append(vs, WidenConstants(start))
	}
	if args.HasVararg {
		if seq, ok := args.Vararg.(Iterable); ok {
			vs = append(vs, WidenConstants(seq.Elem()))
		}
	}
	return Unite(kitectx.TODO(), vs...)
}

// pyZip implements the python builtins.map function
// TODO: (hrysoula) zip now returns an iterator in py3, should anything change here?
func pyZip(args Args) Value {
	// The result of zip(x1, ..., xn) is a list of tuples that look like
	// (elem(x1), ..., elem(xn)) where elem(xi) is the element type of xi.
	//
	// If we have a vararg then we need a whole different code path do not know
	// how many elements will be in the inner tuples, so instead we should
	// represent them as lists:
	//    list<list<elem(elem(vararg))>>
	var elemTypes []Value
	for _, v := range args.Positional {
		if seq, ok := v.(Iterable); ok {
			elemTypes = append(elemTypes, WidenConstants(seq.Elem()))
		} else {
			// must add nil entries since otherwise the tuple indices will be wrong
			elemTypes = append(elemTypes, nil)
		}
	}

	if args.HasVararg {
		if outer, ok := args.Vararg.(Iterable); ok {
			if inner, ok := outer.(Iterable); ok {
				elemTypes = append(elemTypes, inner.Elem())
			}
		}

		// if we resolved the element types for any positional args then we will get:
		//     list<list< elem(x1) | ... | elem(xn) >>
		// if we also had a vararg and we got its inner element type then we will get:
		//     list<list< elem(x1) | ... | elem(xn) | elem(elem(vararg)) >>
		// if we _only_ had a vararg and we got its inner element type then we will get:
		//     list<list< elem(elem(vararg)) >>
		// if we got none then we will fall back to:
		//      list<list< unknown >>
		return NewList(NewList(Unite(kitectx.TODO(), elemTypes...)))
	}

	// in this code path we had no varargs so we know the size of the tuple:
	//     list<tuple< elem(x1) , ... , elem(xn) >>
	return NewList(NewTuple(elemTypes...))
}

// pyDivmod implements builtins.divmod
func pyDivmod(args Args) Value {
	if len(args.Positional) != 2 {
		return NewTuple(FloatInstance{}, FloatInstance{})
	}
	return NewTuple(WidenConstants(args.Positional[0]), WidenConstants(args.Positional[1]))
}

func pyGetattr(args Args) Value {
	if len(args.Positional) < 2 {
		return nil
	}
	obj := args.Positional[0]
	if obj == nil {
		return nil
	}

	var vals []Value
	attr := args.Positional[1]
	for _, vi := range Disjuncts(kitectx.TODO(), attr) {
		if vi, ok := vi.(StrConstant); ok {
			if res, _ := Attr(kitectx.TODO(), obj, string(vi)); res.Found() {
				vals = append(vals, res.Value())
			}
		}
	}

	if defaultVal, ok := args.Keyword("default"); ok {
		vals = append(vals, defaultVal)
	} else if len(args.Positional) >= 3 {
		vals = append(vals, args.Positional[2])
	}
	return Unite(kitectx.TODO(), vals...)
}

func pyMax(args Args) Value {
	var vs []Value
	for _, v := range args.Positional {
		vs = append(vs, WidenConstants(v))
	}
	return Unite(kitectx.TODO(), vs...)
}

func pyMin(args Args) Value {
	var vs []Value
	for _, v := range args.Positional {
		vs = append(vs, WidenConstants(v))
	}
	return Unite(kitectx.TODO(), vs...)
}

func pyNext(args Args) Value {
	if len(args.Positional) == 0 {
		return nil
	}

	seq, ok := args.Positional[0].(Iterable)
	if !ok {
		return nil
	}

	v := seq.Elem()
	if defaultVal, ok := args.Keyword("default"); ok {
		v = Unite(kitectx.TODO(), v, defaultVal)
	} else if len(args.Positional) >= 2 {
		v = Unite(kitectx.TODO(), v, args.Positional[1])
	}
	return v
}

func pyIter(args Args) Value {
	if len(args.Positional) == 0 {
		return nil
	}

	seq, ok := args.Positional[0].(Iterable)
	if !ok {
		return NewList(nil)
	}
	return NewList(seq.Elem())
}

func pyPow(args Args) Value {
	if len(args.Positional) == 0 {
		return FloatInstance{}
	}

	if args.Positional[0] == nil {
		return FloatInstance{}
	}
	return args.Positional[0]
}

func pyReversed(args Args) Value {
	if len(args.Positional) == 0 {
		return nil
	}

	seq, ok := args.Positional[0].(Iterable)
	if !ok {
		return NewList(nil)
	}
	return NewList(seq.Elem())
}

func pySuper(args Args) Value {
	// In python 3, super cannot be implemented as an ordinary function because it
	// recieves no arguments. Instead super is a special case in evaluator.go, which
	// uses the enclosing type to simulate super.
	return nil
}

func pySorted(args Args) Value {
	if len(args.Positional) == 0 {
		return nil
	}

	seq, ok := args.Positional[0].(Iterable)
	if !ok {
		return NewList(nil)
	}
	return NewList(seq.Elem())
}

// TODO: (hrysoula) filter now returns an iterator in py3, should anything change here?
func pyFilter(args Args) Value {
	if len(args.Positional) < 2 {
		return NewList(nil)
	}

	seq, ok := args.Positional[1].(Iterable)
	if !ok {
		return NewList(nil)
	}
	return NewList(seq.Elem())
}
