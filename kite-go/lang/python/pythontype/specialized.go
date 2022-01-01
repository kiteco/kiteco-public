package pythontype

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// ExplicitModule implements the value interface for modules
type ExplicitModule struct {
	Addr    Address
	Members map[string]Value
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v ExplicitModule) Kind() Kind { return ModuleKind }

// Type gets the result of calling type() on this value in python
func (v ExplicitModule) Type() Value { return Builtins.Module }

// Address gets the fully qualified path to this value in the import graph
func (v ExplicitModule) Address() Address { return v.Addr }

// attr looks up an attribute on this values
func (v ExplicitModule) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, v.Members, nil)
}

// equal determines whether this value is equal to another value
func (v ExplicitModule) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(ExplicitModule); ok {
		return v.Addr.Equals(u.Addr)
	}
	return false
}

// Flatten gets the flat version of this value
func (v ExplicitModule) Flatten(f *FlatValue, r *Flattener) {
	f.Explicit = &FlatExplicit{Path: v.Addr.Path}
}

// hash gets a hash for this value, which is used during serialization
func (v ExplicitModule) hash(ctx kitectx.CallContext) FlatID {
	// TODO(alex): this could collide if two funcs have the same path but one
	// is from the global graph and the other is from the local graph
	return rehash(saltModule, FlatID(v.Addr.Path.Hash))
}

// String provides a string representation of this value
func (v ExplicitModule) String() string {
	return fmt.Sprintf("generic:%s", v.Addr.Path.String())
}

// NewModule creates a new module with the give name and members
func NewModule(addr string, dict map[string]Value) Value {
	return ExplicitModule{
		Addr:    SplitAddress(addr),
		Members: dict,
	}
}

// ExplicitType implements the value interface for non-callable, non-iterable objects
type ExplicitType struct {
	Addr        Address
	Constructor func(Args) Value
	Base        Value
	Members     map[string]Value
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v ExplicitType) Kind() Kind { return TypeKind }

// Type gets the result of calling type() on this value in python
func (v ExplicitType) Type() Value { return Builtins.Type }

// Address gets the fully qualified path to this value in the import graph
func (v ExplicitType) Address() Address { return v.Addr }

// Call gets the result of calling this value like a function
func (v ExplicitType) Call(args Args) Value {
	if v.Constructor == nil {
		return nil
	}
	return v.Constructor(args)
}

// attr looks up an attribute on this values
func (v ExplicitType) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, v.Members, v.Base)
}

// equal determines whether this value is equal to another value
func (v ExplicitType) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(ExplicitType); ok {
		return v.Addr.Equals(u.Addr)
	}
	return false
}

// hash gets a hash for this value, which is used during serialization
func (v ExplicitType) hash(ctx kitectx.CallContext) FlatID {
	// TODO(alex): this could collide if two funcs have the same path but one
	// is from the global graph and the other is from the local graph
	return rehash(saltType, FlatID(v.Addr.Path.Hash))
}

// Flatten gets the flat version of this value
func (v ExplicitType) Flatten(f *FlatValue, r *Flattener) {
	f.Explicit = &FlatExplicit{Path: v.Addr.Path}
}

// String provides a string representation of this value
func (v ExplicitType) String() string {
	return fmt.Sprintf("generic:%s", v.Addr.Path.String())
}

// NewType creates a new type with the give name, base class, and dictionary
func NewType(addr string, ctor func(Args) Value, base Value, dict map[string]Value) ExplicitType {
	t := ExplicitType{
		Addr:        SplitAddress(addr),
		Constructor: ctor,
		Base:        base,
		Members:     dict,
	}
	dict["__doc__"] = StrInstance{}
	dict["__base__"] = base
	dict["__bases__"] = NewTuple(base)
	dict["__class__"] = t
	dict["__dict__"] = NewDict(StrInstance{}, nil)
	dict["__name__"] = StrConstant(t.Addr.Path.Last())
	if ctor == nil {
		ctor = func(Args) Value { return nil }
	}
	dict["__call__"] = newRegFunc(addr+".__init__", ctor)
	return t
}

// ExplicitFunc implements the value interface for functions
type ExplicitFunc struct {
	Addr Address
	F    func(args Args) Value
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v ExplicitFunc) Kind() Kind { return FunctionKind }

// Type gets the result of calling type() on this value in python
func (v ExplicitFunc) Type() Value { return Builtins.Function }

// Address gets the fully qualified path to this value in the import graph
func (v ExplicitFunc) Address() Address { return v.Addr }

// Call gets the result of calling this value like a function
func (v ExplicitFunc) Call(args Args) Value { return v.F(args) }

// attr looks up an attribute on this values
func (v ExplicitFunc) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, nil, v.Type())
}

// equal determines whether this value is equal to another value
func (v ExplicitFunc) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(ExplicitFunc); ok {
		return v.Addr.Equals(u.Addr)
	}
	return false
}

// hash gets a hash for this value, which is used during serialization
func (v ExplicitFunc) hash(ctx kitectx.CallContext) FlatID {
	// TODO(alex): this could collide if two funcs have the same path but one
	// is from the global graph and the other is from the local graph
	return rehash(saltFunc, FlatID(v.Addr.Path.Hash))
}

// Flatten gets the flat version of this value
func (v ExplicitFunc) Flatten(f *FlatValue, r *Flattener) {
	f.Explicit = &FlatExplicit{Path: v.Addr.Path}
}

// String provides a string representation of this value
func (v ExplicitFunc) String() string {
	return fmt.Sprintf("generic:%s", v.Addr.Path.String())
}

// NewFunc creates a new function
func NewFunc(addr string, f func(args Args) Value) Value {
	if f == nil {
		// this will cause panics later and will then be harder to diagnose, so
		// it makes sense to panic here
		panic(fmt.Sprintf("NewFunc call with nil function for address %s", addr))
	}
	return ExplicitFunc{
		Addr: SplitAddress(addr),
		F:    f,
	}
}

// BoundMethod implements the value interface for member functions. Note that F
// will be nil after flattening/unflattening because we don't yet have a way to
// flatten unnamed go functions.
type BoundMethod struct {
	Addr Address
	F    func(args Args) Value
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v BoundMethod) Kind() Kind { return FunctionKind }

// Type gets the result of calling type() on this value in python
func (v BoundMethod) Type() Value { return Builtins.Function }

// Address gets the fully qualified path to this value in the import graph
func (v BoundMethod) Address() Address { return v.Addr }

// Call gets the result of calling this value like a function
func (v BoundMethod) Call(args Args) Value { return v.F(args) }

// attr looks up an attribute on this values
func (v BoundMethod) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return resolveAttr(ctx, name, v, nil, v.Type())
}

// equal determines whether this value is equal to another value
func (v BoundMethod) equal(ctx kitectx.CallContext, u Value) bool {
	// Do we really need equality tests for bound methods? If so we will
	// need to store a reference to "self" and also some unique name for
	// the function itself.
	return false
}

// hash gets a hash for this object used during serialization
func (v BoundMethod) hash(ctx kitectx.CallContext) FlatID {
	return rehash(saltBoundMethod, FlatID(v.Addr.Path.Hash))
}

// Flatten gets the flat version of this value
func (v BoundMethod) Flatten(f *FlatValue, r *Flattener) {
	// TODO(alex): figure out how to find the function again on the other side
	f.Scalar = addr(FlatNone)
}

// String provides a string representation of this value
func (v BoundMethod) String() string {
	return fmt.Sprintf("boundmethod:%s", v.Addr.Path.String())
}
