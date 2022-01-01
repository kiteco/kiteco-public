package pythontype

// Builtins contains values representing each python builtin
var Builtins struct {
	// Constants
	None  Value // None represents the python value None
	True  Value // True represents the python value True
	False Value // False represents the python value False

	// Low-level types
	Type     Value // Type represents builtins.type
	Object   Value // Object represents builtins.object
	Function Value // Function represents types.FunctionType
	Method   Value // Method represents types.MethodType
	Module   Value // Module represents types.ModuleType

	// Scalar types
	NoneType Value // NoneType represents builtins.None.__class__
	Bool     Value // Bool represents builtins.bool
	Int      Value // Int represents builtins.int
	Float    Value // Float represents builtins.float
	Complex  Value // Complex represents builtins.complex
	Str      Value // Str represents builtins.str

	// Structured types
	Tuple Value // Tuple represents builtins.tuple
	List  Value // List represents builtins.list
	Dict  Value // Dict represents builtins.dict
	Set   Value // Set represents builtins.set
	Super Value // Super represents builtins.super

	// Other builtin types
	Slice Value // Slice represents builtins.slice

	// Functions that have special meaning to the propagator
	IsInstance Value // IsInstance represents builtins.isinstance
	IsSubclass Value // IsSubclass represents builtins.issubclass
	Eval       Value // Eval represents builtins.eval

	Property Value
}

// BuiltinSymbols contain the same elements as Builtins but represented as a map
// from python identifiers to values. This is useful when resolving python identifiers
// against building
var BuiltinSymbols map[string]Value

// BuiltinModule contains the same elements as BuiltinSymbols but represented as a
// module object
var BuiltinModule Value

func init() {
	// -- constants
	Builtins.None = NoneConstant{}
	Builtins.True = BoolConstant(true)
	Builtins.False = BoolConstant(false)

	// -- low-level types

	// builtins.object
	// NOTE: __class__ and __doc__ are automatically added in `NewType`
	Builtins.Object = newRegType("builtins.object", nil, nil, map[string]Value{
		"__delattr__":      nil,
		"__format__":       nil,
		"__getattribute__": nil,
		"__hash__":         nil,
		"__init__":         nil,
		"__new__":          nil,
		"__reduce__":       nil,
		"__reduce_ex__":    nil,
		"__repr__":         nil,
		"__setattr__":      nil,
		"__sizeof__":       nil,
		"__str__":          nil,
		"__subclasshook__": nil,
	})

	// builtins.type
	Builtins.Type = newRegType("builtins.type", nil, Builtins.Object, map[string]Value{
		"__abstractmethods__": nil,
		"__basicsize__":       nil,
		"__delattr__":         nil,
		"__dictoffset__":      nil,
		"__eq__":              nil,
		"__flags__":           nil,
		"__ge__":              nil,
		"__getattribute__":    nil,
		"__gt__":              nil,
		"__hash__":            nil,
		"__init__":            nil,
		"__instancecheck__":   nil,
		"__itemsize__":        nil,
		"__le__":              nil,
		"__lt__":              nil,
		"__module__":          nil,
		"__mro__":             nil,
		"__ne__":              nil,
		"__new__":             nil,
		"__repr__":            nil,
		"__setattr__":         nil,
		"__subclasscheck__":   nil,
		"__subclasses__":      nil,
		"__weakrefoffset__":   nil,
		"mro":                 nil,
	})

	// function
	Builtins.Function = newRegType("types.FunctionType", nil, Builtins.Object, map[string]Value{
		"__closure__":      nil,
		"__code__":         nil,
		"__defaults__":     NewTuple(),
		"__delattr__":      nil,
		"__get__":          nil,
		"__getattribute__": nil,
		"__globals__":      NewDict(StrInstance{}, nil),
		"__module__":       nil,
		"__name__":         nil,
		"__new__":          nil,
		"__repr__":         nil,
		"__setattr__":      nil,
		"__subclasshook__": nil,
		"func_closure":     nil,
		"func_code":        nil,
		"func_defaults":    NewTuple(),
		"func_dict":        NewDict(StrInstance{}, nil),
		"func_doc":         StrInstance{},
		"func_globals":     NewDict(StrInstance{}, nil),
		"func_name":        nil,
	})

	Builtins.Method = newRegType("types.MethodType", nil, Builtins.Object, map[string]Value{
		"__cmp__":          nil,
		"__delattr__":      nil,
		"__func__":         nil,
		"__get__":          nil,
		"__getattribute__": nil,
		"__hash__":         nil,
		"__new__":          nil,
		"__repr__":         nil,
		"__self__":         nil,
		"__setattr__":      nil,
		"__subclasshook__": nil,
		"im_class":         nil,
		"im_func":          nil,
		"im_self":          nil,
	})

	Builtins.Module = newRegType("types.ModuleType", nil, Builtins.Object, map[string]Value{
		"__delattr__":      nil,
		"__getattribute__": nil,
		"__init__":         nil,
		"__new__":          nil,
		"__repr__":         nil,
		"__setattr__":      nil,
		"__subclasshook__": nil,
	})

	Builtins.NoneType = newRegType("builtins.None.__class__", constructNone, Builtins.Object, map[string]Value{
		"__hash__":         nil,
		"__repr__":         nil,
		"__subclasshook__": nil,
	})

	Builtins.Bool = newNumericType("builtins.bool", constructBool, Builtins.Object, map[string]Value{
		"real":        FloatInstance{},
		"imag":        FloatInstance{},
		"numerator":   IntInstance{},
		"denominator": IntInstance{},
		"conjugate":   newRegFunc("builtins.bool.conjugate", func(args Args) Value { return IntInstance{} }),
		"bit_length":  newRegFunc("builtins.bool.bit_length", func(args Args) Value { return IntInstance{} }),
	})

	Builtins.Int = newNumericType("builtins.int", constructInt, Builtins.Object, map[string]Value{
		"real":        FloatInstance{},
		"imag":        FloatInstance{},
		"numerator":   IntInstance{},
		"denominator": IntInstance{},
		"conjugate":   newRegFunc("builtins.int.conjugate", func(args Args) Value { return IntInstance{} }),
		"bit_length":  newRegFunc("builtins.int.bit_length", func(args Args) Value { return IntInstance{} }),
	})

	Builtins.Float = newNumericType("builtins.float", constructFloat, Builtins.Object, map[string]Value{
		"real":             FloatInstance{},
		"imag":             FloatInstance{},
		"as_integer_ratio": newRegFunc("builtins.float.as_integer_ratio", func(args Args) Value { return NewTuple(IntInstance{}, IntInstance{}) }),
		"conjugate":        newRegFunc("builtins.float.conjugate", func(args Args) Value { return FloatInstance{} }),
		"fromhex":          newRegFunc("builtins.float.fromhex", func(args Args) Value { return FloatInstance{} }),
		"hex":              newRegFunc("builtins.float.hex", func(args Args) Value { return StrInstance{} }),
		"is_integer":       newRegFunc("builtins.float.is_integer", func(args Args) Value { return BoolInstance{} }),
	})

	Builtins.Complex = newNumericType("builtins.complex", constructComplex, Builtins.Object, map[string]Value{
		"real":      FloatInstance{},
		"imag":      FloatInstance{},
		"conjugate": newRegFunc("builtins.complex.conjugate", func(args Args) Value { return ComplexInstance{} }),
	})

	Builtins.Str = newRegType("builtins.str", constructStr, Builtins.Object, map[string]Value{
		"__add__":                     nil,
		"__contains__":                nil,
		"__eq__":                      nil,
		"__format__":                  nil,
		"__ge__":                      nil,
		"__getattribute__":            nil,
		"__getitem__":                 nil,
		"__getnewargs__":              nil,
		"__getslice__":                nil,
		"__gt__":                      nil,
		"__hash__":                    nil,
		"__le__":                      nil,
		"__len__":                     nil,
		"__lt__":                      nil,
		"__mod__":                     nil,
		"__mul__":                     nil,
		"__ne__":                      nil,
		"__new__":                     nil,
		"__repr__":                    nil,
		"__rmod__":                    nil,
		"__rmul__":                    nil,
		"__sizeof__":                  nil,
		"__str__":                     nil,
		"__subclasshook__":            nil,
		"_formatter_field_name_split": nil,
		"_formatter_parser":           nil,
		"capitalize":                  newRegFunc("builtins.str.capitalize", func(args Args) Value { return StrInstance{} }),
		"center":                      newRegFunc("builtins.str.center", func(args Args) Value { return StrInstance{} }),
		"count":                       newRegFunc("builtins.str.count", func(args Args) Value { return IntInstance{} }),
		"decode":                      newRegFunc("builtins.str.decode", func(args Args) Value { return StrInstance{} }),
		"encode":                      newRegFunc("builtins.str.encode", func(args Args) Value { return nil }),
		"endswith":                    newRegFunc("builtins.str.endswith", func(args Args) Value { return BoolInstance{} }),
		"expandtabs":                  newRegFunc("builtins.str.expandtabs", func(args Args) Value { return StrInstance{} }),
		"find":                        newRegFunc("builtins.str.find", func(args Args) Value { return IntInstance{} }),
		"format":                      newRegFunc("builtins.str.format", func(args Args) Value { return StrInstance{} }),
		"index":                       newRegFunc("builtins.str.index", func(args Args) Value { return IntInstance{} }),
		"isalnum":                     newRegFunc("builtins.str.isalnum", func(args Args) Value { return BoolInstance{} }),
		"isalpha":                     newRegFunc("builtins.str.isalpha", func(args Args) Value { return BoolInstance{} }),
		"isdigit":                     newRegFunc("builtins.str.isdigit", func(args Args) Value { return BoolInstance{} }),
		"islower":                     newRegFunc("builtins.str.islower", func(args Args) Value { return BoolInstance{} }),
		"isspace":                     newRegFunc("builtins.str.isspace", func(args Args) Value { return BoolInstance{} }),
		"istitle":                     newRegFunc("builtins.str.istitle", func(args Args) Value { return BoolInstance{} }),
		"isupper":                     newRegFunc("builtins.str.isupper", func(args Args) Value { return BoolInstance{} }),
		"join":                        newRegFunc("builtins.str.join", func(args Args) Value { return StrInstance{} }),
		"ljust":                       newRegFunc("builtins.str.ljust", func(args Args) Value { return StrInstance{} }),
		"lower":                       newRegFunc("builtins.str.lower", func(args Args) Value { return StrInstance{} }),
		"lstrip":                      newRegFunc("builtins.str.lstrip", func(args Args) Value { return StrInstance{} }),
		"partition":                   newRegFunc("builtins.str.partition", func(args Args) Value { return NewTuple(StrInstance{}, StrInstance{}, StrInstance{}) }),
		"replace":                     newRegFunc("builtins.str.replace", func(args Args) Value { return StrInstance{} }),
		"rfind":                       newRegFunc("builtins.str.rfind", func(args Args) Value { return IntInstance{} }),
		"rindex":                      newRegFunc("builtins.str.rindex", func(args Args) Value { return IntInstance{} }),
		"rjust":                       newRegFunc("builtins.str.rjust", func(args Args) Value { return StrInstance{} }),
		"rpartition":                  newRegFunc("builtins.str.rpartition", func(args Args) Value { return NewTuple(StrInstance{}, StrInstance{}, StrInstance{}) }),
		"rsplit":                      newRegFunc("builtins.str.rsplit", func(args Args) Value { return NewList(StrInstance{}) }),
		"rstrip":                      newRegFunc("builtins.str.rstrip", func(args Args) Value { return StrInstance{} }),
		"split":                       newRegFunc("builtins.str.split", func(args Args) Value { return NewList(StrInstance{}) }),
		"splitlines":                  newRegFunc("builtins.str.splitlines", func(args Args) Value { return NewList(StrInstance{}) }),
		"startswith":                  newRegFunc("builtins.str.startswith", func(args Args) Value { return BoolInstance{} }),
		"strip":                       newRegFunc("builtins.str.strip", func(args Args) Value { return StrInstance{} }),
		"swapcase":                    newRegFunc("builtins.str.swapcase", func(args Args) Value { return StrInstance{} }),
		"title":                       newRegFunc("builtins.str.title", func(args Args) Value { return StrInstance{} }),
		"translate":                   newRegFunc("builtins.str.translate", func(args Args) Value { return nil }),
		"upper":                       newRegFunc("builtins.str.upper", func(args Args) Value { return StrInstance{} }),
		"zfill":                       newRegFunc("builtins.str.zfill", func(args Args) Value { return StrInstance{} }),
	})

	Builtins.List = newRegType("builtins.list", constructList, Builtins.Object, map[string]Value{
		"__add__":          nil,
		"__contains__":     nil,
		"__delitem__":      nil,
		"__delslice__":     nil,
		"__eq__":           nil,
		"__ge__":           nil,
		"__getattribute__": nil,
		"__getitem__":      nil,
		"__getslice__":     nil,
		"__gt__":           nil,
		"__hash__":         nil,
		"__iadd__":         nil,
		"__imul__":         nil,
		"__init__":         nil,
		"__iter__":         nil,
		"__le__":           nil,
		"__len__":          nil,
		"__lt__":           nil,
		"__mul__":          nil,
		"__ne__":           nil,
		"__new__":          nil,
		"__repr__":         nil,
		"__reversed__":     nil,
		"__rmul__":         nil,
		"__setitem__":      nil,
		"__setslice__":     nil,
		"__sizeof__":       nil,
		"__subclasshook__": nil,
		"append":           nil,
		"count":            nil,
		"extend":           nil,
		"index":            nil,
		"insert":           nil,
		"pop":              nil,
		"remove":           nil,
		"reverse":          nil,
		"sort":             nil,
	})

	Builtins.Dict = newRegType("builtins.dict", constructDict, Builtins.Object, map[string]Value{
		"__cmp__":          nil,
		"__contains__":     nil,
		"__delitem__":      nil,
		"__eq__":           nil,
		"__ge__":           nil,
		"__getattribute__": nil,
		"__getitem__":      nil,
		"__gt__":           nil,
		"__hash__":         nil,
		"__init__":         nil,
		"__iter__":         nil,
		"__le__":           nil,
		"__len__":          nil,
		"__lt__":           nil,
		"__ne__":           nil,
		"__new__":          nil,
		"__repr__":         nil,
		"__setitem__":      nil,
		"__sizeof__":       nil,
		"__subclasshook__": nil,
		"clear":            nil,
		"copy":             nil,
		"fromkeys":         nil,
		"get":              nil,
		"has_key":          nil,
		"items":            nil,
		"keys":             nil,
		"pop":              nil,
		"popitem":          nil,
		"setdefault":       nil,
		"update":           nil,
		"values":           nil,
		"viewitems":        nil,
		"viewkeys":         nil,
		"viewvalues":       nil,
	})

	Builtins.Set = newRegType("builtins.set", constructSet, Builtins.Object, map[string]Value{
		"__and__":                     nil,
		"__cmp__":                     nil,
		"__contains__":                nil,
		"__eq__":                      nil,
		"__ge__":                      nil,
		"__getattribute__":            nil,
		"__gt__":                      nil,
		"__hash__":                    nil,
		"__iand__":                    nil,
		"__init__":                    nil,
		"__ior__":                     nil,
		"__isub__":                    nil,
		"__iter__":                    nil,
		"__ixor__":                    nil,
		"__le__":                      nil,
		"__len__":                     nil,
		"__lt__":                      nil,
		"__ne__":                      nil,
		"__new__":                     nil,
		"__or__":                      nil,
		"__rand__":                    nil,
		"__reduce__":                  nil,
		"__repr__":                    nil,
		"__ror__":                     nil,
		"__rsub__":                    nil,
		"__rxor__":                    nil,
		"__sizeof__":                  nil,
		"__sub__":                     nil,
		"__subclasshook__":            nil,
		"__xor__":                     nil,
		"add":                         nil,
		"clear":                       nil,
		"copy":                        nil,
		"difference":                  nil,
		"difference_update":           nil,
		"discard":                     nil,
		"intersection":                nil,
		"intersection_update":         nil,
		"isdisjoint":                  nil,
		"issubset":                    nil,
		"issuperset":                  nil,
		"pop":                         nil,
		"remove":                      nil,
		"symmetric_difference":        nil,
		"symmetric_difference_update": nil,
		"union":                       nil,
		"update":                      nil,
	})

	Builtins.Tuple = newRegType("builtins.tuple", constructTuple, Builtins.Object, map[string]Value{
		"__add__":          nil,
		"__contains__":     nil,
		"__eq__":           nil,
		"__ge__":           nil,
		"__getattribute__": nil,
		"__getitem__":      nil,
		"__getnewargs__":   nil,
		"__getslice__":     nil,
		"__gt__":           nil,
		"__hash__":         nil,
		"__iter__":         nil,
		"__le__":           nil,
		"__len__":          nil,
		"__lt__":           nil,
		"__mul__":          nil,
		"__ne__":           nil,
		"__new__":          nil,
		"__repr__":         nil,
		"__rmul__":         nil,
		"__subclasshook__": nil,
		"count":            nil,
		"index":            nil,
	})

	// There is no constructor for super because in python 3 it takes no arguments so
	// cannot be implemented as a regular function. Instead pythonstatic.Evaluator
	// checks for super as a special case and calls NewSuper.
	Builtins.Super = newRegType("builtins.super", nil, Builtins.Object, map[string]Value{
		"__delattr__":      nil,
		"__doc__":          nil,
		"__format__":       nil,
		"__get__":          nil,
		"__getattribute__": nil,
		"__hash__":         nil,
		"__init__":         nil,
		"__new__":          nil,
		"__reduce__":       nil,
		"__reduce_ex__":    nil,
		"__repr__":         nil,
		"__self__":         nil,
		"__self_class__":   nil,
		"__setattr__":      nil,
		"__sizeof__":       nil,
		"__str__":          nil,
		"__subclasshook__": nil,
		"__thisclass__":    nil,
	})

	Builtins.Eval = newRegFunc("builtins.eval", func(args Args) Value { return nil })
	Builtins.IsInstance = newRegFunc("builtins.isinstance", func(args Args) Value { return BoolInstance{} })
	Builtins.IsSubclass = newRegFunc("builtins.issubclass", func(args Args) Value { return BoolInstance{} })

	Builtins.Property = newRegType("builtins.property", constructProperty, Builtins.Object, map[string]Value{
		"__delete__": nil,
		"__get__":    nil,
		"__set__":    nil,
		"__format__": nil,
		"fget":       nil,
		"fset":       nil,
		"fdel":       nil,
		"getter":     nil,
		"setter":     nil,
		"deleter":    nil,
	})

	// BuiltinSymbols contains only those builtins that are accessible as members of the
	// builtins module. (So there is no entry for "True" or "None" or "function")
	BuiltinSymbols = map[string]Value{
		"object": Builtins.Object,
		"type":   Builtins.Type,

		"NoneType": Builtins.NoneType,
		"bool":     Builtins.Bool,
		"int":      Builtins.Int,
		"float":    Builtins.Float,
		"complex":  Builtins.Complex,
		"str":      Builtins.Str,

		"None":  Builtins.None,
		"True":  Builtins.True,
		"False": Builtins.False,

		"list":  Builtins.List,
		"dict":  Builtins.Dict,
		"set":   Builtins.Set,
		"tuple": Builtins.Tuple,
		"super": Builtins.Super,

		"property": Builtins.Property,

		"eval":       Builtins.Eval,
		"isinstance": Builtins.IsInstance,
		"issubclass": Builtins.IsSubclass,
		"abs": newRegFunc("builtins.abs", func(args Args) Value {
			return Union{[]Value{FloatInstance{}, IntInstance{}}}
		}),
		"dir":     newRegFunc("builtins.dir", func(args Args) Value { return NewList(StrInstance{}) }),
		"globals": newRegFunc("builtins.globals", func(args Args) Value { return NewDict(StrInstance{}, nil) }),
		"range":   newRegFunc("builtins.range", func(args Args) Value { return NewList(IntInstance{}) }),
		"vars":    newRegFunc("builtins.vars", func(args Args) Value { return NewDict(StrInstance{}, nil) }),
		"len":     newRegFunc("builtins.len", func(_ Args) Value { return IntInstance{} }),

		// these functions have values that depend on their parameters
		"divmod":    newRegFunc("builtins.divmod", pyDivmod),
		"enumerate": newRegFunc("builtins.enumerate", pyEnumerate),
		"getattr":   newRegFunc("builtins.getattr", pyGetattr),
		"map":       newRegFunc("builtins.map", pyMap),
		"max":       newRegFunc("builtins.max", pyMax),
		"min":       newRegFunc("builtins.min", pyMin),
		"next":      newRegFunc("builtins.next", pyNext),
		"iter":      newRegFunc("builtins.iter", pyIter),
		"pow":       newRegFunc("builtins.pow", pyPow),
		"reversed":  newRegFunc("builtins.reversed", pyReversed),
		"sorted":    newRegFunc("builtins.sorted", pySorted),
		"sum":       newRegFunc("builtins.sum", pySum),
		"filter":    newRegFunc("builtins.filter", pyFilter),
		"zip":       newRegFunc("builtins.zip", pyZip),
	}

	// builtins module
	BuiltinModule = newRegModule("builtins", BuiltinSymbols)
}

func newNumericType(addr string, ctor func(Args) Value, base Value, dict map[string]Value) ExplicitType {
	// add members that are common to all numeric types
	dict["__abs__"] = nil
	dict["__add__"] = nil
	dict["__coerce__"] = nil
	dict["__div__"] = nil
	dict["__divmod__"] = nil
	dict["__eq__"] = nil
	dict["__float__"] = nil
	dict["__floordiv__"] = nil
	dict["__format__"] = nil
	dict["__ge__"] = nil
	dict["__getattribute__"] = nil
	dict["__getnewargs__"] = nil
	dict["__gt__"] = nil
	dict["__hash__"] = nil
	dict["__int__"] = nil
	dict["__le__"] = nil
	dict["__lt__"] = nil
	dict["__mod__"] = nil
	dict["__mul__"] = nil
	dict["__ne__"] = nil
	dict["__neg__"] = nil
	dict["__new__"] = nil
	dict["__nonzero__"] = nil
	dict["__pos__"] = nil
	dict["__pow__"] = nil
	dict["__radd__"] = nil
	dict["__rdiv__"] = nil
	dict["__rdivmod__"] = nil
	dict["__repr__"] = nil
	dict["__rfloordiv__"] = nil
	dict["__rmod__"] = nil
	dict["__rmul__"] = nil
	dict["__rpow__"] = nil
	dict["__rsub__"] = nil
	dict["__rtruediv__"] = nil
	dict["__str__"] = nil
	dict["__sub__"] = nil
	dict["__subclasshook__"] = nil
	dict["__truediv__"] = nil
	return newRegType(addr, ctor, Builtins.Object, dict)
}
