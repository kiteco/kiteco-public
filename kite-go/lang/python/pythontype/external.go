package pythontype

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/pkg/errors"
)

// GlobalValue is an (instances of a) Value that lives in the global symbol graph.
// It must be one of an External, ExternalInstance, or ExternalRoot
type GlobalValue interface {
	Value
	Canonical() GlobalValue
	Dist() keytypes.Distribution
}

// ExternalRoot is a dummy value representing the "root" of the symbol graph
type ExternalRoot struct {
	Graph pythonresource.Manager
}

// Canonical implements GlobalValue interface
func (v ExternalRoot) Canonical() GlobalValue {
	return v
}

// Dist implements GlobalValue
func (v ExternalRoot) Dist() keytypes.Distribution {
	return keytypes.Distribution{}
}

// Kind implements Value
func (v ExternalRoot) Kind() Kind {
	return UnknownKind
}

// Type implements Value
func (v ExternalRoot) Type() Value {
	return nil
}

// Address implements Value
func (v ExternalRoot) Address() Address {
	return Address{IsExternalRoot: true}
}

// attr implements Value
func (v ExternalRoot) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	pkgPath := pythonimports.NewDottedPath(name)
	var vals []ValueNamespace
	for _, dist := range v.Graph.DistsForPkg(name) {
		if sym, err := v.Graph.NewSymbol(dist, pkgPath); err == nil {
			vals = append(vals, ValueNamespace{
				Value:     TranslateExternal(sym, v.Graph),
				Namespace: v,
			})
		}
	}
	return UnionResult(vals), nil
}

// equal implements Value
func (v ExternalRoot) equal(ctx kitectx.CallContext, val Value) bool {
	_, ok := val.(ExternalRoot)
	return ok
}

// Flatten implements Value
func (v ExternalRoot) Flatten(f *FlatValue, r *Flattener) {
	f.ExternalRoot = &FlatExternalRoot{}
}

// hash implements Value
func (v ExternalRoot) hash(ctx kitectx.CallContext) FlatID {
	return saltExternal
}

// String returns a string representation of the value
func (v ExternalRoot) String() string {
	return "external-root"
}

// External represents a value from the symbol graph. The contained symbol is always valid by construction.
type External struct {
	symbol pythonresource.Symbol
	graph  pythonresource.Manager
}

// Canonical implements GlobalValue interface
func (v External) Canonical() GlobalValue {
	v.symbol = v.symbol.Canonical()
	return v
}

// Dist implements GlobalValue
func (v External) Dist() keytypes.Distribution {
	return v.symbol.Dist()
}

// NewExternal constructs a new External value given a Symbol and graph (resource Manager)
func NewExternal(sym pythonresource.Symbol, graph pythonresource.Manager) External {
	return External{
		symbol: sym,
		graph:  graph,
	}
}

// Symbol returns the Symbol corresponding to this External
func (v External) Symbol() pythonresource.Symbol {
	return v.symbol
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v External) Kind() Kind {
	switch v.graph.Kind(v.symbol) {
	case keytypes.ModuleKind:
		return ModuleKind
	case keytypes.FunctionKind:
		return FunctionKind
	case keytypes.TypeKind:
		return TypeKind
	case keytypes.ObjectKind:
		return InstanceKind
	default:
		// Descriptors will resolve to UnknownKind, which is formally correct,
		// although perhaps we should approximate them at Instances?
		return UnknownKind
	}
}

func (v External) extType() (External, error) {
	sym, err := v.graph.Type(v.symbol)
	if err != nil {
		return External{}, errors.New("type not navigable")
	}
	return NewExternal(sym, v.graph), nil
}

// Type gets the result of calling type() on this value in python
func (v External) Type() Value {
	ext, err := v.extType()
	if err != nil {
		return nil
	}
	return TranslateExternal(ext.symbol, ext.graph)
}

// Call is the result of calling this function. It returns nil except for types,
// for which it instantiates the type.
func (v External) Call(args Args) Value {
	var typeSyms []pythonresource.Symbol
	if v.Kind() == TypeKind {
		typeSyms = append(typeSyms, v.symbol)
	} else {
		typeSyms = v.graph.ReturnTypes(v.symbol)
	}

	var typeVals []Value
	for _, typeSym := range typeSyms {
		typeVals = append(typeVals, TranslateExternalInstance(typeSym, v.graph))
	}

	if len(typeVals) == 0 {
		typeVals = append(typeVals, ExternalReturnValue{
			fn:    v.symbol,
			graph: v.graph,
		})
	}

	return Unite(kitectx.TODO(), typeVals...)
}

// Address is the address for this object, or nil for non-addressable values
func (v External) Address() Address {
	return Address{Path: v.symbol.Path()}
}

// attr gets the named attribute, checking the symbol, its base classes, and its type.
func (v External) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	symAttrs := resolveExternalSymAttrs(ctx, v, name)
	var pairs []ValueNamespace
	for _, symAttr := range symAttrs {
		pairs = append(pairs, ValueNamespace{
			Value:     TranslateExternal(symAttr.child, v.graph),
			Namespace: TranslateExternal(symAttr.parent, v.graph),
		})
	}

	return UnionResult(pairs), nil
}

// equal determines whether this value is equal to another value
func (v External) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(External); ok {
		return v.symbol.Equals(u.symbol)
	}
	return false
}

// Flatten gets the representation of an external instance suitable for serialization
func (v External) Flatten(f *FlatValue, r *Flattener) {
	f.External = &FlatExternal{v.symbol.Path()}
}

// String provides a string representation of this value
func (v External) String() string {
	return fmt.Sprintf("external:%s", v.symbol.String())
}

// hash gets a unique ID for this value (used during serialization)
func (v External) hash(ctx kitectx.CallContext) FlatID {
	dist := v.symbol.Dist()
	return rehashBytes(rehashBytes(rehash(saltExternal, FlatID(v.symbol.PathHash())), []byte(dist.Name)), []byte(dist.Version))
}

// --- TODO(naman) throw out SourceInstance, StubInstance, ExternalInstance in favor of a unified Instance Value

// ExternalInstance represents an instance of a value from the import
// graph. The node contained within an ExternalInstance is never nil.
type ExternalInstance struct {
	TypeExternal External
}

// Canonical implements GlobalValue interface
func (v ExternalInstance) Canonical() GlobalValue {
	v.TypeExternal = v.TypeExternal.Canonical().(External)
	return v
}

// Dist implements GlobalValue
func (v ExternalInstance) Dist() keytypes.Distribution {
	return v.TypeExternal.Dist()
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v ExternalInstance) Kind() Kind {
	return InstanceKind
}

// Type gets the result of calling type() on this value in python
func (v ExternalInstance) Type() Value {
	return v.TypeExternal
}

// Address is the address for this object, or nil for non-addressable values
func (v ExternalInstance) Address() Address {
	return Address{}
}

// attr gets the named attribute. Steps is a counter to avoid infinite loops.
func (v ExternalInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return attr(ctx, v.TypeExternal, name)
}

// equal determines whether this value is equal to another value
func (v ExternalInstance) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(ExternalInstance); ok {
		return equal(ctx, v.TypeExternal, u.TypeExternal)
	}
	return false
}

// hash gets a unique ID for this value (used during serialization)
func (v ExternalInstance) hash(ctx kitectx.CallContext) FlatID {
	return rehashValues(ctx, saltExternalInstance, v.TypeExternal)
}

// Flatten gets the representation of an external instance suitable for serialization
func (v ExternalInstance) Flatten(f *FlatValue, r *Flattener) {
	f.ExternalInstance = &FlatExternalInstance{v.TypeExternal.symbol.Path()}
}

// String provides a string representation of this value
func (v ExternalInstance) String() string {
	return fmt.Sprintf("externalinstance:%s", v.TypeExternal.symbol.PathString())
}

// ExternalReturnValue is a placeholder
// for a global function that we do not know the return value of
// TODO: this is pretty hacky and is meant for consumption on the ggnn end
type ExternalReturnValue struct {
	fn    pythonresource.Symbol
	graph pythonresource.Manager
}

// Canonical implements GlobalValue interface
func (v ExternalReturnValue) Canonical() GlobalValue {
	v.fn = v.fn.Canonical()
	return v
}

// Dist implements GlobalValue
func (v ExternalReturnValue) Dist() keytypes.Distribution {
	return v.fn.Dist()
}

// NewExternalReturnValue ...
func NewExternalReturnValue(fn pythonresource.Symbol, graph pythonresource.Manager) ExternalReturnValue {
	return ExternalReturnValue{
		fn:    fn,
		graph: graph,
	}
}

// Func returns the Symbol corresponding to the function this value was returned from
func (v ExternalReturnValue) Func() pythonresource.Symbol {
	return v.fn
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (v ExternalReturnValue) Kind() Kind {
	// TODO: use instance instead?
	return UnknownKind
}

// Type gets the result of calling type() on this value in python
func (v ExternalReturnValue) Type() Value {
	// TODO: do better
	return nil
}

// Address is the address for this object, or nil for non-addressable values
func (v ExternalReturnValue) Address() Address {
	// TODO: unclear what address should be...
	return Address{}
}

// attr gets the named attribute. Steps is a counter to avoid infinite loops.
func (v ExternalReturnValue) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	return AttrResult{}, ErrNotFound
}

// equal determines whether this value is equal to another value
func (v ExternalReturnValue) equal(ctx kitectx.CallContext, u Value) bool {
	if u, ok := u.(ExternalReturnValue); ok {
		return v.fn.Equals(u.fn)
	}
	return false
}

// Flatten gets the representation of an external instance suitable for serialization
func (v ExternalReturnValue) Flatten(f *FlatValue, r *Flattener) {
	f.ExternalReturnValue = &FlatExternalReturnValue{v.fn.Path()}
}

// String provides a string representation of this value
func (v ExternalReturnValue) String() string {
	return fmt.Sprintf("external-return-value:%s", v.fn.String())
}

// hash gets a unique ID for this value (used during serialization)
func (v ExternalReturnValue) hash(ctx kitectx.CallContext) FlatID {
	dist := v.fn.Dist()
	return rehashBytes(rehashBytes(rehash(saltExternalReturnValue, FlatID(v.fn.PathHash())), []byte(dist.Name)), []byte(dist.Version))
}

// TranslateExternal translates a Symbol into a Value. If the value has a special implementation within the type system,
// that is returned; otherwise the value simply wraps the provided Symbol.
func TranslateExternal(symbol pythonresource.Symbol, graph pythonresource.Manager) Value { // TODO[external]
	// TODO: inline the hash switch below (they are constants but Go does not know)
	// TODO: for the specialized values, we may want to parametrize them by the (not necessarily canonical) symbol so
	// that we can access it later.
	switch symbol.Canonical().PathHash() {
	// special values
	case Builtins.None.Address().Path.Hash:
		return Builtins.None
	case Builtins.True.Address().Path.Hash:
		return Builtins.True
	case Builtins.False.Address().Path.Hash:
		return Builtins.False

		// scalar types
	case Builtins.NoneType.Address().Path.Hash:
		return Builtins.NoneType
	case Builtins.Bool.Address().Path.Hash:
		return Builtins.Bool
	case Builtins.Int.Address().Path.Hash:
		return Builtins.Int
	case Builtins.Float.Address().Path.Hash:
		return Builtins.Float
	case Builtins.Complex.Address().Path.Hash:
		return Builtins.Complex
	case Builtins.Str.Address().Path.Hash:
		return Builtins.Str

		// structured types
	case Builtins.List.Address().Path.Hash:
		return Builtins.List
	case Builtins.Dict.Address().Path.Hash:
		return Builtins.Dict
	case Builtins.Set.Address().Path.Hash:
		return Builtins.Set
	case Builtins.Tuple.Address().Path.Hash:
		return Builtins.Tuple

		// other low-level types
	case Builtins.Object.Address().Path.Hash:
		return Builtins.Object
	case Builtins.Type.Address().Path.Hash:
		return Builtins.Type
	case Builtins.Function.Address().Path.Hash:
		return Builtins.Function
	case Builtins.Method.Address().Path.Hash:
		return Builtins.Method

	case Builtins.Property.Address().Path.Hash:
		return Builtins.Property

		// collections
	case Collections.Counter.Address().Path.Hash:
		return Collections.Counter
	case Collections.DefaultDict.Address().Path.Hash:
		return Collections.DefaultDict
	case Collections.Deque.Address().Path.Hash:
		return Collections.Deque
	case Collections.OrderedDict.Address().Path.Hash:
		return Collections.OrderedDict
	case Collections.NamedTuple.Address().Path.Hash:
		return Collections.NamedTuple
	case dataFrameAddress.Path.Hash:
		return NewDataFrame(graph)

		// Queue
	case Queue.LifoQueue.Address().Path.Hash:
		return Queue.LifoQueue
	case Queue.PriorityQueue.Address().Path.Hash:
		return Queue.PriorityQueue
	case Queue.Queue.Address().Path.Hash:
		return Queue.Queue

		// django
	case Django.DB.Models.Manager.Address().Path.Hash:
		return Django.DB.Models.Manager
	case Django.DB.Models.QuerySet.Address().Path.Hash:
		return Django.DB.Models.QuerySet
	case Django.DB.Models.Options.Options.Address().Path.Hash:
		return Django.DB.Models.Options.Options
	case Django.Shortcuts.GetObjectOr404.Address().Path.Hash:
		return Django.Shortcuts.GetObjectOr404
	case Django.Shortcuts.GetListOr404.Address().Path.Hash:
		return Django.Shortcuts.GetListOr404
	}

	if tySym, err := graph.Type(symbol); err == nil {
		if v := translateInstance(tySym, graph); v != nil {
			return v
		}
	}

	return External{
		symbol: symbol,
		graph:  graph,
	}
}

// TranslateExternalInstance translates a Symbol into a Value representing an instance of the Value identified by the Symbol
func TranslateExternalInstance(tySymbol pythonresource.Symbol, graph pythonresource.Manager) Value {
	if v := translateInstance(tySymbol, graph); v != nil {
		return v
	}
	// TODO(naman) we should not canonicalize here if possible
	return ExternalInstance{NewExternal(tySymbol.Canonical(), graph)}
}

// translateInstance checks for a special implementation of an instance of the provided type symbol
// if none exists, it returns nil
func translateInstance(typeSymbol pythonresource.Symbol, graph pythonresource.Manager) Value {
	// TODO: inline the rehash below (they are constants but Go does not know)
	switch typeSymbol.Canonical().PathHash() {
	case Builtins.NoneType.Address().Path.Hash:
		return NoneConstant{}
	case Builtins.Bool.Address().Path.Hash:
		return BoolInstance{}
	case Builtins.Int.Address().Path.Hash:
		return IntInstance{}
	case Builtins.Float.Address().Path.Hash:
		return FloatInstance{}
	case Builtins.Complex.Address().Path.Hash:
		return ComplexInstance{}
	case Builtins.Str.Address().Path.Hash:
		return StrInstance{}
	case Builtins.List.Address().Path.Hash:
		return NewList(nil)
	case Builtins.Dict.Address().Path.Hash:
		return NewDict(nil, nil)
	case Builtins.Set.Address().Path.Hash:
		return NewSet(nil)
	case Builtins.Tuple.Address().Path.Hash:
		return NewTuple()
	case Builtins.Property.Address().Path.Hash:
		return NewPropertyInstance(nil, nil)

		// collections
	case Collections.Counter.Address().Path.Hash:
		return NewCounter(nil, nil)
	case Collections.DefaultDict.Address().Path.Hash:
		return NewDefaultDict(nil, nil, nil)
	case Collections.Deque.Address().Path.Hash:
		return NewDeque(nil)
	case Collections.OrderedDict.Address().Path.Hash:
		return NewOrderedDict(nil, nil)
	case dataFrameAddress.Path.Hash:
		return NewDataFrameInstanceFromGraph(nil, nil, graph)

		// Queue
	case Queue.LifoQueue.Address().Path.Hash:
		return NewLifoQueue(nil)
	case Queue.PriorityQueue.Address().Path.Hash:
		return NewPriorityQueue(nil)
	case Queue.Queue.Address().Path.Hash:
		return NewQueue(nil)

		// django
	case Django.DB.Models.Manager.Address().Path.Hash:
		return NewManager(nil)
	case Django.DB.Models.QuerySet.Address().Path.Hash:
		return NewQuerySet(nil)
	case Django.DB.Models.Options.Options.Address().Path.Hash:
		return NewOptions(nil)

	default:
		return nil
	}
}
