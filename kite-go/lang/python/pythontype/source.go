package pythontype

import (
	"fmt"
	"path"
	"strings"
	"unsafe"

	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// SourceValue is a Value created from direct analysis of python source files/directories
type SourceValue interface {
	Value
	source()
}

// Parameter represents a parameter to a function
type Parameter struct {
	Name        string
	Default     Value
	Symbol      *Symbol
	KeywordOnly bool
}

// ---

// SourceFunction represents a python function
type SourceFunction struct {
	Parameters       []Parameter
	Vararg           *Parameter
	Kwarg            *Parameter
	KwargDict        *KwargDict // KwargDict contains the names and values of each kwarg
	Return           *Symbol    // Return is a symbol so that we can track where it gets produced/consumed
	Locals           *SymbolTable
	Class            *SourceClass  // Class is the class in which this function was defined, or nil
	Module           *SourceModule // Module is the module in which this function was defined
	HasReceiver      bool          // HasReceiver indicates whether this function has a "self" parameter
	HasClassReceiver bool          // HasClassReceiver indicates whether this function has a "cls" parameter
}

func (v *SourceFunction) source() {}

// Address is the path for this value in the import graph
func (v *SourceFunction) Address() Address {
	return v.Locals.Name
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v *SourceFunction) Kind() Kind {
	return FunctionKind
}

// Type gets the result of calling type() on this value in python
func (v *SourceFunction) Type() Value {
	return Builtins.Function
}

// attr gets the type of the named attribute
func (v *SourceFunction) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	return AttrResult{}, ErrNotFound
}

// Call gets the result of calling this function
func (v *SourceFunction) Call(args Args) Value {
	return v.Return.Value
}

// equal determines whether this value is equal to another value
func (v *SourceFunction) equal(ctx kitectx.CallContext, other Value) bool {
	if fun, ok := other.(*SourceFunction); ok {
		// test for equality by pointer identity
		return v == fun
	}
	return false
}

// hash gets a unique ID for this value
func (v *SourceFunction) hash(ctx kitectx.CallContext) FlatID {
	// two functions are the same iff they are at the same memory location
	return FlatID(uintptr(unsafe.Pointer(v)))
}

// Flatten gets the flat version of this value
func (v *SourceFunction) Flatten(f *FlatValue, r *Flattener) {
	var fp []FlatParameter
	for _, param := range v.Parameters {
		fp = append(fp, flattenParameter(param, r))
	}
	ff := &FlatFunction{
		Params: fp,
		Return: FlatSymbol{
			Name:    v.Return.Name,
			Value:   r.Flatten(v.Return.Value),
			Private: v.Return.Private,
		},
		HasReceiver:      v.HasReceiver,
		HasClassReceiver: v.HasClassReceiver,
	}

	if v.Vararg != nil {
		fp := flattenParameter(*v.Vararg, r)
		ff.Vararg = &fp
	}

	if v.KwargDict != nil {
		ff.KwargDict = r.Flatten(v.KwargDict)
		fp := flattenParameter(*v.Kwarg, r)
		ff.Kwarg = &fp
	}

	ff.Locals = flattenSymbolTable(v.Locals, r)

	f.Function = ff
}

// String returns a string representation of a function
func (v *SourceFunction) String() string {
	return "src-func:" + v.Locals.Name.String()
}

// ---

// SourceClass represents a class created from python source
type SourceClass struct {
	Members    *SymbolTable
	Subclasses []*SourceClass // Subclasses is for proposing types for "self"
	Bases      []Value        // Bases consists of SourceClass and External structs
}

func (v *SourceClass) source() {}

// Address is the path for this value in the import graph
func (v *SourceClass) Address() Address {
	return v.Members.Name
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v *SourceClass) Kind() Kind {
	return TypeKind
}

// Type gets a value representing the result of calling type() on this class in python
func (v *SourceClass) Type() Value {
	return Builtins.Type
}

// Call instantiates the class like a python constructor
func (v *SourceClass) Call(args Args) Value {
	return SourceInstance{v}
}

// attr gets the type of the named attribute
func (v *SourceClass) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "__dict__":
		return SingleResult(NewDict(StrInstance{}, nil), v), nil
	case "__class__":
		return SingleResult(v.Type(), v), nil
	case "__weakref__":
		return SingleResult(nil, v), nil
	case "__name__", "__doc__", "__module__":
		return SingleResult(StrInstance{}, v), nil
	}

	// Search for attribute in symbol table
	if sym, found := v.Members.Get(name); found {
		return SingleResult(sym.Value, v), nil
	}

	// Search for attribute in base classes
	for _, b := range v.Bases {
		if res, _ := attr(ctx, b, name); res.Found() {
			return res, nil
		}
	}
	return AttrResult{}, ErrNotFound
}

// AttrSymbol gets the symbol for a named attribute on this value. It does not
// search base classes. Either the attribute is found on this object, or it
// is not found, in which case it is created it create is set to true.
func (v *SourceClass) AttrSymbol(ctx kitectx.Context, name string, create bool) (syms []*Symbol) {
	ctx.WithCallLimit(maxSteps, func(ctx kitectx.CallContext) error {
		return internalAttr(ctx, v, name, &syms)
	})
	if len(syms) == 0 && create {
		syms = append(syms, v.Members.Create(name))
	}
	return syms
}

// internalAttr gets the symbol and type for the named attribute. If the
// attribute is found in a parent class that is from the import graph then
// the returned symbol will be nil.
func internalAttr(ctx kitectx.CallContext, v *SourceClass, name string, syms *[]*Symbol) error {
	ctx.CheckAbort()

	// Case 2: attribute found in this class
	if sym, found := v.Members.Get(name); found {
		if sym == nil {
			panic(fmt.Sprintf("%s was found in %v but the symbol was nil", name, v))
		}
		*syms = append(*syms, sym)
		return nil
	}

	// Case 3: search base classes for attribute
	for _, b := range v.Bases {
		switch b := b.(type) {
		case *SourceClass:
			// recursive call to internalAttr: increment ctx call count
			if err := internalAttr(ctx.Call(), b, name, syms); err == nil {
				return nil
			}
		case Union:
			// unionBaseLookup internally increments ctx call count
			if err := unionBaseLookup(ctx, b, name, syms); err == nil {
				return nil
			}
		default:
			// attr internally increments ctx call count
			if _, err := attr(ctx, b, name); err == nil {
				return nil
			}
		}
	}

	// Case 4: no attribute found
	return ErrNotFound
}

// unionBaseLookup internally increments ctx call count
func unionBaseLookup(ctx kitectx.CallContext, u Union, name string, syms *[]*Symbol) error {
	ctx.CheckAbort()

	var found bool
	for _, v := range u.Constituents {
		switch v := v.(type) {
		case *SourceClass:
			// recursive call to internalAttr: increment ctx call counter
			if err := internalAttr(ctx.Call(), v, name, syms); err == nil {
				found = true
			}
		default:
			// attr internally increments ctx call count
			if _, err := attr(ctx, v, name); err == nil {
				found = true
			}
		}
	}
	if found {
		return nil
	}
	return ErrNotFound
}

// equal determines whether this value is equal to another value
func (v *SourceClass) equal(ctx kitectx.CallContext, other Value) bool {
	if cls, ok := other.(*SourceClass); ok {
		// test for equality by pointer identity
		return v == cls
	}
	return false
}

// hash gets a unique ID for this value
func (v *SourceClass) hash(ctx kitectx.CallContext) FlatID {
	// two classes are the same iff they are at the same memory location
	return FlatID(uintptr(unsafe.Pointer(v)))
}

// Flatten gets the flat version of this value
func (v *SourceClass) Flatten(f *FlatValue, r *Flattener) {
	var bases []FlatID
	for _, base := range v.Bases {
		bases = append(bases, r.Flatten(base))
	}

	var subclasses []FlatID
	for _, sub := range v.Subclasses {
		subclasses = append(subclasses, r.Flatten(sub))
	}

	f.Class = &FlatClass{
		Bases:      bases,
		Members:    flattenSymbolTable(v.Members, r),
		Subclasses: subclasses,
	}
}

// String returns a string representation of a class
func (v *SourceClass) String() string {
	return "src-class:" + v.Members.Name.String()
}

// AddSubclass adds a subclass to this class, unless it has already been added
func (v *SourceClass) AddSubclass(sub *SourceClass) {
	for _, c := range v.Subclasses {
		if c == sub {
			return
		}
	}
	v.Subclasses = append(v.Subclasses, sub)
}

// --- TODO(naman) throw out SourceInstance, StubInstance, ExternalInstance in favor of a unified Instance Value

// SourceInstance represents an instance of user-defined class
type SourceInstance struct {
	Class *SourceClass
}

func (v SourceInstance) source() {}

// Address is the path for this value in the import graph
func (v SourceInstance) Address() Address {
	return Address{}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v SourceInstance) Kind() Kind {
	return InstanceKind
}

// Type gets a value representing the result of calling type() on this class in python
func (v SourceInstance) Type() Value {
	return v.Class
}

// attr gets the type of the named attribute
func (v SourceInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "__class__":
		return SingleResult(v.Class, v), nil
	case "__dict__":
		return SingleResult(NewDict(StrInstance{}, nil), v), nil
	default:
		return attr(ctx, v.Class, name)
	}
}

// AttrSymbol gets the symbol for a named attribute
func (v SourceInstance) AttrSymbol(ctx kitectx.Context, attr string, create bool) []*Symbol {
	return v.Class.AttrSymbol(ctx, attr, create)
}

// equal determines whether this value is equal to another value
func (v SourceInstance) equal(ctx kitectx.CallContext, other Value) bool {
	if u, ok := other.(SourceInstance); ok {
		// test for equality by pointer identity between the classes
		return v.Class == u.Class
	}
	return false
}

// hash gets a unique ID for this value
func (v SourceInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, 0xF0C5341E9ACB0442, v.Class)
}

// Flatten gets the flat version of this value
func (v SourceInstance) Flatten(f *FlatValue, r *Flattener) {
	f.Instance = &FlatInstance{Class: r.Flatten(v.Class)}
}

// String returns a string representation of a class
func (v SourceInstance) String() string {
	return "src-instance:" + v.Class.Members.Name.String()
}

// ---

// SourceModule represents a python file
type SourceModule struct {
	Members *SymbolTable
}

func (v *SourceModule) source() {}

// Address is the path for this value in the import graph
func (v *SourceModule) Address() Address {
	return v.Members.Name
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v *SourceModule) Kind() Kind {
	return ModuleKind
}

// Type gets a value representing the result of calling type() on this class in python
func (v *SourceModule) Type() Value {
	return Builtins.Module
}

// attr gets the type of the named attribute
func (v *SourceModule) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	switch name {
	case "__dict__":
		return SingleResult(NewDict(StrInstance{}, nil), v), nil
	case "__loader__":
		return SingleResult(nil, v), nil
	case "__name__", "__package__", "__doc__", "__version__":
		return SingleResult(StrInstance{}, v), nil
	}

	// do not look in parent scopes here
	if sym, found := v.Members.Get(name); found {
		return SingleResult(sym.Value, v), nil
	}
	return AttrResult{}, ErrNotFound
}

// AttrSymbol gets the symbol for a named attribute
func (v *SourceModule) AttrSymbol(ctx kitectx.Context, attr string, create bool) []*Symbol {
	if sym, found := v.Members.Get(attr); found {
		return []*Symbol{sym}
	} else if create {
		return []*Symbol{v.Members.Create(attr)}
	}
	return nil
}

// equal determines whether this value is equal to another value
func (v *SourceModule) equal(ctx kitectx.CallContext, other Value) bool {
	if mod, ok := other.(*SourceModule); ok {
		// test for equality by pointer identity
		return v == mod
	}
	return false
}

// hash gets a unique ID for this value
func (v *SourceModule) hash(ctx kitectx.CallContext) FlatID {
	// two modules are the same iff they have the same address in memory
	return FlatID(uintptr(unsafe.Pointer(v)))
}

// Flatten gets the flat version of this value
func (v *SourceModule) Flatten(f *FlatValue, r *Flattener) {
	f.Module = &FlatModule{
		Members: flattenSymbolTable(v.Members, r),
	}
}

// String returns a string representation of a module
func (v *SourceModule) String() string {
	return "src-module:" + path.Base(v.Members.Name.File)
}

// ---

// SourcePackage represents a package created from python code
type SourcePackage struct {
	LowerCase  bool
	DirEntries *SymbolTable  // DirEntries contains one symbol for each file and dir inside this package
	Init       *SourceModule // Init is the __init__.py module, or nil
}

func (v *SourcePackage) source() {}

// Address is the path for this value in the import graph
func (v *SourcePackage) Address() Address {
	return v.DirEntries.Name
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v *SourcePackage) Kind() Kind {
	return ModuleKind
}

// Type gets a value representing the result of calling type() on this package in python
func (v *SourcePackage) Type() Value {
	return Builtins.Module
}

// DirAttr looks up a child module/package in v's directory
func (v *SourcePackage) DirAttr(attr string) (*Symbol, bool) {
	if v.LowerCase {
		attr = strings.ToLower(attr)
	}
	sym, ok := v.DirEntries.Get(attr)
	return sym, ok
}

// attr looks up an attribute within a package.
func (v *SourcePackage) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	// symbols in the __init__.py module take precedence
	if v.Init != nil {
		if res, _ := attr(ctx, v.Init, name); res.Found() {
			return res, nil
		}
	}
	if sym, found := v.DirAttr(name); found {
		return SingleResult(sym.Value, v), nil
	}
	return AttrResult{}, ErrNotFound
}

// equal determines whether this value is equal to another value
func (v *SourcePackage) equal(ctx kitectx.CallContext, other Value) bool {
	if q, ok := other.(*SourcePackage); ok {
		return q == v // use pointer equality
	}
	return false
}

// hash gets a unique ID for this value
func (v *SourcePackage) hash(ctx kitectx.CallContext) FlatID {
	// two packages are the same iff they have the same address in memory
	return FlatID(uintptr(unsafe.Pointer(v)))
}

// Flatten gets the flat version of this value
func (v *SourcePackage) Flatten(f *FlatValue, r *Flattener) {
	var init FlatID // the __init__.py module, or empty if no init
	if v.Init != nil {
		init = r.Flatten(v.Init)
	}
	f.Package = &FlatPackage{
		LowerCase:  v.LowerCase,
		Init:       init,
		DirEntries: flattenSymbolTable(v.DirEntries, r),
	}
}

// String returns a string representation of this package
func (v *SourcePackage) String() string {
	return "src-pkg:" + path.Base(v.DirEntries.Name.File)
}
