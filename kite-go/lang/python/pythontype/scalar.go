package pythontype

import (
	"math"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	noneAddr  = SplitAddress("builtins.None")
	trueAddr  = SplitAddress("builtins.True")
	falseAddr = SplitAddress("builtins.False")
)

// ConstantValue is a Value that represents a specific primitive value in Python,
// typically represented with a syntactic literal in Python source
type ConstantValue interface {
	Value
	constant()
}

// NoneConstant represents the None value. There is no struct called
// NoneInstance because there is only one None value in python.
type NoneConstant struct{}

func (v NoneConstant) constant() {}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v NoneConstant) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v NoneConstant) Type() Value { return Builtins.NoneType }

// Address gets the fully qualified path to this value in the import graph
func (v NoneConstant) Address() Address { return noneAddr }

// String gets a string representation of this value
func (v NoneConstant) String() string { return "None" }

// hash gets a unique ID for this value (for serialization)
func (v NoneConstant) hash(ctx kitectx.CallContext) FlatID { return saltNone }

// Flatten creates a non-recursive version of this type for serialization
func (v NoneConstant) Flatten(f *FlatValue, r *Flattener) { f.Scalar = addr(FlatNone) }

// attr gets the value of an attribute. Steps is a counter to avoid infinite loops.
func (v NoneConstant) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, nil, v.Type())
}

// equal determines whether this value is equal to another value
func (v NoneConstant) equal(ctx kitectx.CallContext, u Value) bool {
	_, eq := u.(NoneConstant)
	return eq
}

func constructNone(args Args) Value {
	return NoneConstant{}
}

// BoolConstant represents the True and False values.
type BoolConstant bool

func (v BoolConstant) constant() {}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v BoolConstant) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v BoolConstant) Type() Value { return Builtins.Bool }

// Flatten creates a non-recursive version of this type for serialization
func (v BoolConstant) Flatten(f *FlatValue, r *Flattener) {
	f.Constant = &FlatConstant{Bool: &struct{ Val BoolConstant }{v}}
}

// Address gets the fully qualified path to this value in the import graph
func (v BoolConstant) Address() Address {
	if v {
		return trueAddr
	}
	return falseAddr
}

// hash gets a unique ID for this value (for serialization)
func (v BoolConstant) hash(ctx kitectx.CallContext) FlatID {
	if v {
		return saltTrue
	}
	return saltFalse
}

// attr gets the value of an attribute. Steps is a counter to avoid infinite loops.
func (v BoolConstant) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, nil, v.Type())
}

// equal determines whether this value is equal to another value
func (v BoolConstant) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(BoolConstant); ok {
		return bool(v) == bool(u)
	}
	return false
}

// BoolInstance represents a unknown boolean value
type BoolInstance struct{}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v BoolInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v BoolInstance) Type() Value { return Builtins.Bool }

// Address gets the fully qualified path to this value in the import graph
func (v BoolInstance) Address() Address { return Address{} }

// String gets a string representation of this value
func (v BoolInstance) String() string { return "bool" }

// hash gets a unique ID for this value (for serialization)
func (v BoolInstance) hash(ctx kitectx.CallContext) FlatID { return saltBool }

// Flatten creates a non-recursive version of this type for serialization
func (v BoolInstance) Flatten(f *FlatValue, r *Flattener) { f.Scalar = addr(FlatBool) }

// attr gets the value of an attribute. Steps is a counter to avoid infinite loops.
func (v BoolInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, nil, v.Type())
}

// equal determines whether this value is equal to another value
func (v BoolInstance) equal(ctx kitectx.CallContext, u Value) bool {
	_, eq := u.(BoolInstance)
	return eq
}

func constructBool(args Args) Value {
	if len(args.Positional) == 1 {
		if v, ok := args.Positional[0].(BoolConstant); ok {
			return v
		}
	}
	return BoolInstance{}
}

// IntConstant represents an integer whose specific numeric value is known
// to static analysis.
type IntConstant int64

func (v IntConstant) constant() {}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v IntConstant) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v IntConstant) Type() Value { return Builtins.Int }

// Address gets the fully qualified path to this value in the import graph
func (v IntConstant) Address() Address { return Address{} }

// hash gets a unique ID for this value (for serialization)
func (v IntConstant) hash(ctx kitectx.CallContext) FlatID { return rehash(saltInt, FlatID(v)) }

// Flatten creates a non-recursive version of this type for serialization
func (v IntConstant) Flatten(f *FlatValue, r *Flattener) {
	f.Constant = &FlatConstant{Int: &struct{ Val IntConstant }{v}}
}

// attr gets the value of an attribute. Steps is a counter to avoid infinite loops.
func (v IntConstant) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, nil, v.Type())
}

// equal determines whether this value is equal to another value
func (v IntConstant) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(IntConstant); ok {
		return int64(v) == int64(u)
	}
	return false
}

// IntInstance represents an integer with unkonwn numeric value
type IntInstance struct{}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v IntInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v IntInstance) Type() Value { return Builtins.Int }

// Address gets the fully qualified path to this value in the import graph
func (v IntInstance) Address() Address { return Address{} }

// String gets a string representation of this value
func (v IntInstance) String() string { return "int" }

// hash gets a unique ID for this value (for serialization)
func (v IntInstance) hash(ctx kitectx.CallContext) FlatID { return saltInt }

// Flatten creates a non-recursive version of this type for serialization
func (v IntInstance) Flatten(f *FlatValue, r *Flattener) { f.Scalar = addr(FlatInt) }

// attr gets the value of an attribute. Steps is a counter to avoid infinite loops.
func (v IntInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, nil, v.Type())
}

// equal determines whether this value is equal to another value
func (v IntInstance) equal(ctx kitectx.CallContext, u Value) bool {
	_, eq := u.(IntInstance)
	return eq
}

func constructInt(args Args) Value {
	if len(args.Positional) == 1 {
		if v, ok := args.Positional[0].(IntConstant); ok {
			return v
		}
	}
	return IntInstance{}
}

// FloatConstant represents a float whose specific numeric value is known
// to static analysis.
type FloatConstant float64

func (v FloatConstant) constant() {}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v FloatConstant) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v FloatConstant) Type() Value { return Builtins.Float }

// Address gets the fully qualified path to this value in the import graph
func (v FloatConstant) Address() Address { return Address{} }

// hash gets a unique ID for this value (for serialization)
func (v FloatConstant) hash(ctx kitectx.CallContext) FlatID {
	return rehash(saltFloat, FlatID(math.Float64bits(float64(v))))
}

// Flatten creates a non-recursive version of this type for serialization
func (v FloatConstant) Flatten(f *FlatValue, r *Flattener) {
	f.Constant = &FlatConstant{Float: &struct{ Val FloatConstant }{v}}
}

// attr gets the value of an attribute. Steps is a counter to avoid infinite loops.
func (v FloatConstant) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, nil, v.Type())
}

// equal determines whether this value is equal to another value
func (v FloatConstant) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(FloatConstant); ok {
		return float64(v) == float64(u)
	}
	return false
}

// FloatInstance represents a long with unkonwn numeric value
type FloatInstance struct{}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v FloatInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v FloatInstance) Type() Value { return Builtins.Float }

// Address gets the fully qualified path to this value in the import graph
func (v FloatInstance) Address() Address { return Address{} }

// String gets a string representation of this value
func (v FloatInstance) String() string { return "float" }

// hash gets a unique ID for this value (for serialization)
func (v FloatInstance) hash(ctx kitectx.CallContext) FlatID { return saltFloat }

// Flatten creates a non-recursive version of this type for serialization
func (v FloatInstance) Flatten(f *FlatValue, r *Flattener) { f.Scalar = addr(FlatFloat) }

// attr gets the value of an attribute. Steps is a counter to avoid infinite loops.
func (v FloatInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, nil, v.Type())
}

// equal determines whether this value is equal to another value
func (v FloatInstance) equal(ctx kitectx.CallContext, u Value) bool {
	_, eq := u.(FloatInstance)
	return eq
}

func constructFloat(args Args) Value {
	if len(args.Positional) == 1 {
		if v, ok := args.Positional[0].(FloatConstant); ok {
			return v
		}
	}
	return FloatInstance{}
}

// ComplexConstant represents a complex whose specific numeric value is known
// to static analysis.
type ComplexConstant complex128

func (v ComplexConstant) constant() {}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v ComplexConstant) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v ComplexConstant) Type() Value { return Builtins.Complex }

// Address gets the fully qualified path to this value in the import graph
func (v ComplexConstant) Address() Address { return Address{} }

// hash gets a unique ID for this value (for serialization)
func (v ComplexConstant) hash(ctx kitectx.CallContext) FlatID {
	return rehash(saltComplex, FlatID(math.Float64bits(real(v))), FlatID(math.Float64bits(imag(v))))
}

// Flatten creates a non-recursive version of this type for serialization
func (v ComplexConstant) Flatten(f *FlatValue, r *Flattener) {
	f.Constant = &FlatConstant{Complex: &struct{ Val ComplexConstant }{v}}
}

// attr gets the value of an attribute. Steps is a counter to avoid infinite loops.
func (v ComplexConstant) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, nil, v.Type())
}

// equal determines whether this value is equal to another value
func (v ComplexConstant) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(ComplexConstant); ok {
		return complex128(v) == complex128(u)
	}
	return false
}

// ComplexInstance represents a complex with unkonwn numeric value
type ComplexInstance struct{}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v ComplexInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v ComplexInstance) Type() Value { return Builtins.Complex }

// Address gets the fully qualified path to this value in the import graph
func (v ComplexInstance) Address() Address { return Address{} }

// String gets a string representation of this value
func (v ComplexInstance) String() string { return "complex" }

// hash gets a unique ID for this value (for serialization)
func (v ComplexInstance) hash(ctx kitectx.CallContext) FlatID { return saltComplex }

// Flatten creates a non-recursive version of this type for serialization
func (v ComplexInstance) Flatten(f *FlatValue, r *Flattener) { f.Scalar = addr(FlatComplex) }

// attr gets the value of an attribute. Steps is a counter to avoid infinite loops.
func (v ComplexInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, nil, v.Type())
}

// equal determines whether this value is equal to another value
func (v ComplexInstance) equal(ctx kitectx.CallContext, u Value) bool {
	_, eq := u.(ComplexInstance)
	return eq
}

func constructComplex(args Args) Value {
	if len(args.Positional) == 1 {
		if v, ok := args.Positional[0].(ComplexConstant); ok {
			return v
		}
	}
	return ComplexInstance{}
}

// StrConstant represents a string whose specific value is known
// to static analysis.
type StrConstant string

func (v StrConstant) constant() {}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v StrConstant) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v StrConstant) Type() Value { return Builtins.Str }

// Address gets the fully qualified path to this value in the import graph
func (v StrConstant) Address() Address { return Address{} }

// String gets a string representation of this value
func (v StrConstant) String() string { return `"` + string(v) + `"` }

// hash gets a unique ID for this value (for serialization)
func (v StrConstant) hash(ctx kitectx.CallContext) FlatID { return rehashBytes(saltStr, []byte(v)) }

// Flatten creates a non-recursive version of this type for serialization
func (v StrConstant) Flatten(f *FlatValue, r *Flattener) {
	f.Constant = &FlatConstant{Str: &struct{ Val StrConstant }{v}}
}

// attr gets the value of an attribute. Steps is a counter to avoid infinite loops.
func (v StrConstant) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, nil, v.Type())
}

// equal determines whether this value is equal to another value
func (v StrConstant) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(StrConstant); ok {
		return string(v) == string(u)
	}
	return false
}

// Elem gets the result of indexing or iterating over this value
func (v StrConstant) Elem() Value {
	return StrInstance{}
}

// Index gets the result of indexing or iterating over this value
func (v StrConstant) Index(index Value, allowValueMutation bool) Value {
	return StrInstance{}
}

// StrInstance represents a string with unkonwn value
type StrInstance struct{}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v StrInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (v StrInstance) Type() Value { return Builtins.Str }

// Address gets the fully qualified path to this value in the import graph
func (v StrInstance) Address() Address { return Address{} }

// String gets a string representation of this value
func (v StrInstance) String() string { return "str" }

// hash gets a unique ID for this value (for serialization)
func (v StrInstance) hash(ctx kitectx.CallContext) FlatID { return saltStr }

// Flatten creates a non-recursive version of this type for serialization
func (v StrInstance) Flatten(f *FlatValue, r *Flattener) { f.Scalar = addr(FlatStr) }

// attr gets the value of an attribute. Steps is a counter to avoid infinite loops.
func (v StrInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, nil, v.Type())
}

// equal determines whether this value is equal to another value
func (v StrInstance) equal(ctx kitectx.CallContext, u Value) bool {
	_, eq := u.(StrInstance)
	return eq
}

// Elem gets the result of iterating over this value
func (v StrInstance) Elem() Value {
	return StrInstance{}
}

// Index gets the result of indexing this value
func (v StrInstance) Index(index Value, allowValueMutation bool) Value {
	return StrInstance{}
}

func constructStr(args Args) Value {
	if len(args.Positional) == 1 {
		if v, ok := args.Positional[0].(StrConstant); ok {
			return v
		}
	}
	return StrInstance{}
}
