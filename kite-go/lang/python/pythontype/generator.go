package pythontype

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	// Generator represents types.GeneratorType
	Generator    Value
	genCloseAddr = SplitAddress("types.GeneratorType.close")
	genNextAddr  = SplitAddress("types.GeneratorType.next")
	genSendAddr  = SplitAddress("types.GeneratorType.send")
	genThrowAddr = SplitAddress("types.GeneratorType.throw")
)

func init() {
	Generator = newRegType("types.GeneratorType", func(Args) Value { return NewGenerator(nil) }, nil, map[string]Value{
		"close": nil,
		"next":  nil,
		"send":  nil,
		"throw": nil,
	})
}

// GeneratorInstance represents a generator over the provided elem
type GeneratorInstance struct {
	Element Value
}

// NewGenerator returns a an instance of Generator
func NewGenerator(elem Value) Value {
	return GeneratorInstance{elem}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (g GeneratorInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (g GeneratorInstance) Type() Value { return Generator }

// Address returns the fully qualified path to this value
func (g GeneratorInstance) Address() Address { return Address{} }

// Elem returns the value that results from iterating over this value
func (g GeneratorInstance) Elem() Value { return g.Element }

// attr looks up an attribute on this value
func (g GeneratorInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "close":
		return SingleResult(BoundMethod{genCloseAddr, func(Args) Value { return Builtins.None }}, g), nil
	case "next":
		return SingleResult(BoundMethod{genNextAddr, func(Args) Value { return g.Elem() }}, g), nil
	case "send":
		return SingleResult(BoundMethod{genSendAddr, func(Args) Value { return Builtins.None }}, g), nil
	case "throw":
		return SingleResult(BoundMethod{genThrowAddr, func(Args) Value { return Builtins.None }}, g), nil
	default:
		return AttrResult{}, ErrNotFound
	}
}

// equal determines whether this value is equal to another value
func (g GeneratorInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(GeneratorInstance); ok {
		return equal(ctx, u.Elem(), g.Elem())
	}
	return false
}

// Flatten creates a flat (non recursive) version of this type
func (g GeneratorInstance) Flatten(f *FlatValue, r *Flattener) {
	f.Generator = &FlatGenerator{r.Flatten(g.Elem())}
}

// hash gets a unique ID for this value, used during serialization
func (g GeneratorInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltGenerator, g.Elem())
}

// String provides a string representation of this value
func (g GeneratorInstance) String() string {
	return fmt.Sprintf("generator{%v}", g.Elem())
}

// FlatGenerator is the representation of GeneratorInstance used for serialization
type FlatGenerator struct {
	Element FlatID
}

// Inflate creates a value from a flat value
func (f FlatGenerator) Inflate(r *Inflater) Value {
	return NewGenerator(r.Inflate(f.Element))
}
