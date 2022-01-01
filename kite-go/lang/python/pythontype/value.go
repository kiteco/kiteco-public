package pythontype

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// the maximum number of recursive attribute evaluations before giving up
const maxSteps = 50

// Kind is the most meta of all the meta levels we use to describe values in
// python. It reveals the ultimate essence of all things, including things that
// are not directly representable as python values, such as union types.
type Kind int

const (
	// UnknownKind indicates a python value about which we know nothing
	UnknownKind Kind = iota
	// FunctionKind indicates a python function or method
	FunctionKind
	// TypeKind indicates a python type
	TypeKind
	// ModuleKind indicates a python module
	ModuleKind
	// InstanceKind indicates an instance of some type
	InstanceKind
	// UnionKind indicates a value that is a union of several possible values
	UnionKind
	// DescriptorKind indicates a value that ascribes to the descriptor protocol
	DescriptorKind
)

// NodeKind translates this kind to an import graph kind. If this kind has no
// corresponding import graph kind then pythonimports.Unknown will be returned.
func (k Kind) NodeKind() pythonimports.Kind {
	switch k {
	case UnknownKind:
		return pythonimports.None
	case FunctionKind:
		return pythonimports.Function
	case TypeKind:
		return pythonimports.Type
	case ModuleKind:
		return pythonimports.Module
	case InstanceKind:
		return pythonimports.Object
	case DescriptorKind:
		return pythonimports.Descriptor
	default:
		return pythonimports.None
	}
}

// TranslateNodeKind converts a import graph kind into a type kind. It is the
// inverse of NodeKind above
func TranslateNodeKind(k pythonimports.Kind) Kind {
	switch k {
	case pythonimports.None:
		return UnknownKind
	case pythonimports.Function:
		return FunctionKind
	case pythonimports.Type:
		return TypeKind
	case pythonimports.Module:
		return ModuleKind
	case pythonimports.Object:
		return InstanceKind
	case pythonimports.Descriptor:
		return DescriptorKind
	default:
		return UnknownKind
	}
}

// String gets a string representation of this kind
func (k Kind) String() string {
	switch k {
	case UnknownKind:
		return "unknown"
	case FunctionKind:
		return "function"
	case TypeKind:
		return "type"
	case ModuleKind:
		return "module"
	case InstanceKind:
		return "instance"
	case UnionKind:
		return "union"
	case DescriptorKind:
		return "descriptor"
	default:
		return fmt.Sprintf("invalid(%d)", k)
	}
}

// Value represents a set of values that a python expression might have. It
// can represent a specific value, such as a class or function with a known
// name, or it can represent sets of values that correspond to python types,
// such as any integer or any sequence of tuples, or it can represent sets of
// values that do not correspond to python types, such as disjunctions of the
// above.
type Value interface {
	// Kind categorizes this value as function/type/module/instance/union/etc
	Kind() Kind

	// Type gets the result of calling type() on this value in python
	Type() Value

	// Address is the path for this value in the import graph
	Address() Address

	// Flatten creates a non-recursive version of this type for serialization
	// TODO(naman) this should be private
	Flatten(*FlatValue, *Flattener)

	hash(ctx kitectx.CallContext) FlatID
	equal(kitectx.CallContext, Value) bool
	attr(kitectx.CallContext, string) (AttrResult, error)
}

// Callable is the interface for values that can be used as functions
type Callable interface {
	// Call gets the value resulting from calling this object with the given parameters.
	Call(args Args) Value
}

// CallableValue combines Callable and Value
type CallableValue interface {
	Callable
	Value
}

// Iterable is the interface for values that support iteration
type Iterable interface {
	// Elem gets the value for the values returned when we iterate over an Iterable,
	// i.e the value of `x` in `for x in someiterable:...`
	Elem() Value
}

// Indexable is the interface for values that support indexing
type Indexable interface {
	// Index gets the value for an element at the provided index,
	// i.e the value of `x` in `x = someindexable[0]`
	Index(index Value, allowValueMutation bool) Value
}

// IndexAssignable is the interface for valeus that can be assigned
// to via index expressions such as "value[i] = rhs"
type IndexAssignable interface {
	// SetIndex returns the new value that results from
	// assigning to the specified index.
	// e.g
	// l = list(1,2) -> l = list<int>
	// l[0] = "hello" -> l = list<int|str>
	SetIndex(index Value, val Value, allowValueMutation bool) Value
}

// Mutable represents a value with attributes that can be updated and on which
// new attributes can be created.
type Mutable interface {
	// AttrSymbol gets the symbol for the named attribute. If the attribute
	// does not exist and create is true then a symbol is created.
	AttrSymbol(ctx kitectx.Context, attr string, create bool) []*Symbol
}

// AttrNoCtx is equivalent to Attr
func AttrNoCtx(v Value, name string) (AttrResult, error) {
	return Attr(kitectx.Background(), v, name)
}

// Attr gets the named attribute Value of v, representing `{v}.{name}` in Python
// WARNING:
//  - The semantics are currently unclear around the return values of this function (SEE https://github.com/kiteco/kiteco/issues/9393),
//    until this is resolved the best way to check if an attribute was sucessfully found is to check `res.Found()`.
func Attr(ctx kitectx.Context, v Value, name string) (res AttrResult, err error) {
	err = ctx.WithCallLimit(maxSteps, func(ctx kitectx.CallContext) (err error) {
		res, err = attr(ctx, v, name)
		return
	})
	return
}

// attr internally increments ctx's call count
func attr(ctx kitectx.CallContext, v Value, name string) (AttrResult, error) {
	ctx.CheckAbort()
	if v == nil {
		return AttrResult{}, ErrNotFound
	}
	return v.attr(ctx.Call(), name)
}

// EqualNoCtx is equivalent to Equal
func EqualNoCtx(u, v Value) bool {
	return Equal(kitectx.Background(), u, v)
}

// Equal determines whether two Value objects represent the same set of values.
// This is not the same as testing those values for equality from within python
// because a Value represents a set of possible python values, and this function
// implements set equality, whereas in python each expression only has one value
// at runtime, and the == operator simply compares them. For example:
//   - two integer instances are considered equal by this function but could easily
//     not be equal in python
//   - this function considers the Unknown value to be equal to the Unknown value but
//     of course those values could be anything, which means they could easily not be
//     equal in python
//   - this function does not consider the union value "int|str" to be equal to the
//     "int" value, yet in python those two values could in fact be equal (because they
//     could both be the same integer)
func Equal(ctx kitectx.Context, u, v Value) bool {
	var res bool
	err := ctx.WithCallLimit(maxSteps, func(ctx kitectx.CallContext) error {
		res = equal(ctx, u, v)
		return nil
	})
	if err != nil {
		return false
	}
	return res
}

func equal(ctx kitectx.CallContext, u, v Value) bool {
	ctx.CheckAbort()
	if u == nil && v == nil {
		return true
	}
	if u == nil || v == nil {
		return false
	}
	return u.equal(ctx.Call(), v)
}

func hash(ctx kitectx.CallContext, v Value) FlatID {
	ctx.CheckAbort()
	if v == nil {
		return 0
	}
	return v.hash(ctx.Call())
}

// Hash gets the hash for a value
func Hash(ctx kitectx.Context, v Value) (FlatID, error) {
	if v == nil {
		return 0, nil
	}

	var h FlatID
	err := ctx.WithCallLimit(50, func(ctx kitectx.CallContext) error {
		h = hash(ctx, v)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return h, nil
}

// MustHash gets the hash for a value, panicking on error
func MustHash(v Value) FlatID {
	h, err := Hash(kitectx.Background(), v)
	if err != nil {
		panic(err)
	}
	return h
}

// String representation of a value,
// this is guaranteed to not be recursive.
func String(v Value) string {
	if v == nil {
		return "<nil>"
	}

	switch v := v.(type) {
	case TupleInstance:
		return "tuple"
	case SetInstance:
		return "set"
	case DictInstance:
		return "dict"
	case ListInstance:
		return "list"
	case Union:
		return "union"
	case SuperInstance:
		return "super"
	case ExplicitFunc, ExplicitModule, ExplicitType, BoundMethod:
		// just prints address
		return fmt.Sprintf("%v", v)
	case *SourceClass, *SourceFunction, SourceInstance, *SourceModule, *SourcePackage:
		// just prints address
		return fmt.Sprintf("%v", v)
	case PriorityQueueInstance:
		return "PriorityQueue"
	case LifoQueueInstance:
		return "LifoQueue"
	case QueueInstance:
		return "Queue"
	case IntConstant, IntInstance, FloatConstant, FloatInstance,
		StrConstant, StrInstance, NoneConstant, BoolConstant, BoolInstance,
		ComplexConstant, ComplexInstance:
		return fmt.Sprintf("%v", v)
	case *KwargDict:
		return fmt.Sprintf("%v", v)
	case GeneratorInstance:
		return "generator"
	case External:
		return v.symbol.PathString()
	case ExternalInstance:
		return v.TypeExternal.symbol.PathString()
	case ExternalRoot:
		return v.String()
	case ManagerInstance:
		return "django.db.models.Manager"
	case QuerySetInstance:
		return "django.db.models.QuerySet"
	case OptionsInstance:
		return "django.db.models.Options"
	case CounterInstance:
		return "collections.Counter"
	case OrderedDictInstance:
		return "collections.OrderedDict"
	case DefaultDictInstance:
		return "collections.DefaultDict"
	case DequeInstance:
		return "collections.Deque"
	case NamedTupleInstance, NamedTupleType:
		return "collections.NamedTuple"
	default:
		return "<unknown value>"
	}
}
