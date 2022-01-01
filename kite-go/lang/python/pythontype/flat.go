package pythontype

import (
	"fmt"
	"log"
	"reflect"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

const (
	// FlatUnknown represents a value about which we know nothing. It
	// corresponds to the nil Value. This enum must start at 1 not 0
	// because gob encodes a pointer to the integer zero as a nil pointer.
	FlatUnknown FlatScalar = 1 + iota
	// FlatNone represents the None value
	FlatNone
	// FlatBool represents a boolean value
	FlatBool
	// FlatInt represents an integer value
	FlatInt
	// FlatFloat represents a floating point value
	FlatFloat
	// FlatComplex represents a complex value
	FlatComplex
	// FlatStr represents a string value
	FlatStr
)

// FlatID is the type for ids on flat values
type FlatID uint64

// FlatValue is the flat representation for values. Its ID is always set, and
// exactly one other field is also set
type FlatValue struct {
	ID FlatID

	Scalar   *FlatScalar
	Constant *FlatConstant
	Union    *FlatUnion

	List            *FlatList
	Dict            *FlatDict
	Set             *FlatSet
	Tuple           *FlatTuple
	Super           *FlatSuper
	Property        *FlatProperty
	PropertyUpdater *FlatPropertyUpdater
	Generator       *FlatGenerator

	Function *FlatFunction
	Class    *FlatClass
	Instance *FlatInstance
	Module   *FlatModule
	Package  *FlatPackage

	KwargDict *FlatKwargDict

	Explicit *FlatExplicit

	ExternalRoot        *FlatExternalRoot
	External            *FlatExternal
	ExternalInstance    *FlatExternalInstance
	ExternalReturnValue *FlatExternalReturnValue

	// collections
	Counter            *FlatCounter
	OrderedDict        *FlatOrderedDict
	DefaultDict        *FlatDefaultDict
	Deque              *FlatDeque
	NamedTupleType     *FlatNamedTupleType
	NamedTupleInstance *FlatNamedTupleInstance

	// Queue
	Queue         *FlatQueue
	LifoQueue     *FlatLifoQueue
	PriorityQueue *FlatPriorityQueue

	// Django
	Manager  *FlatManager
	QuerySet *FlatQuerySet
	Options  *FlatOptions

	// Pandas
	DataFrame         *FlatDataFrame
	DataFrameInstance *FlatDataFrameInstance
}

// Inflate converts a flat value to a value
func (f FlatValue) Inflate(i *Inflater) Value {
	switch {
	case f.Scalar != nil:
		return f.Scalar.Inflate(i)
	case f.Constant != nil:
		return f.Constant.Inflate(i)
	case f.Union != nil:
		return f.Union.Inflate(i)
	case f.List != nil:
		return f.List.Inflate(i)
	case f.Dict != nil:
		return f.Dict.Inflate(i)
	case f.Set != nil:
		return f.Set.Inflate(i)
	case f.Tuple != nil:
		return f.Tuple.Inflate(i)
	case f.Super != nil:
		return f.Super.Inflate(i)
	case f.Property != nil:
		return f.Property.Inflate(i)
	case f.PropertyUpdater != nil:
		return f.PropertyUpdater.Inflate(i)
	case f.Generator != nil:
		return f.Generator.Inflate(i)
	case f.Function != nil:
		return f.Function.Inflate(i)
	case f.Class != nil:
		return f.Class.Inflate(i)
	case f.Instance != nil:
		return f.Instance.Inflate(i)
	case f.Module != nil:
		return f.Module.Inflate(i)
	case f.Package != nil:
		return f.Package.Inflate(i)
	case f.KwargDict != nil:
		return f.KwargDict.Inflate(i)
	case f.Explicit != nil:
		return f.Explicit.Inflate(i)
	case f.ExternalRoot != nil:
		return f.ExternalRoot.Inflate(i)
	case f.External != nil:
		return f.External.Inflate(i)
	case f.ExternalInstance != nil:
		return f.ExternalInstance.Inflate(i)
	case f.ExternalReturnValue != nil:
		return f.ExternalReturnValue.Inflate(i)
	case f.Counter != nil:
		return f.Counter.Inflate(i)
	case f.OrderedDict != nil:
		return f.OrderedDict.Inflate(i)
	case f.DefaultDict != nil:
		return f.DefaultDict.Inflate(i)
	case f.Deque != nil:
		return f.Deque.Inflate(i)
	case f.NamedTupleType != nil:
		return f.NamedTupleType.Inflate(i)
	case f.NamedTupleInstance != nil:
		return f.NamedTupleInstance.Inflate(i)
	case f.Queue != nil:
		return f.Queue.Inflate(i)
	case f.LifoQueue != nil:
		return f.LifoQueue.Inflate(i)
	case f.PriorityQueue != nil:
		return f.PriorityQueue.Inflate(i)
	case f.Manager != nil:
		return f.Manager.Inflate(i)
	case f.QuerySet != nil:
		return f.QuerySet.Inflate(i)
	case f.Options != nil:
		return f.Options.Inflate(i)
	default:
		rollbar.Error(fmt.Errorf("invalid FlatValue: all fields were nil"))
		return nil
	}
}

// Check returns an error unless this value contains exactly one non-nil field.
func (f FlatValue) Check() error {
	// This for debugging only so we use reflection. TODO(alex): delete.
	var count int
	v := reflect.ValueOf(f)
	// start at 1 to skip the ID field
	for i := 1; i < v.NumField(); i++ {
		if !v.Field(i).IsNil() {
			count++
		}
	}
	if count == 0 {
		return fmt.Errorf("all fields were nil")
	} else if count > 1 {
		return fmt.Errorf("%d fields were non-nil", count)
	}
	return nil
}

// Link connects a value to other values during unflattening. This is necessary to
// avoid circular dependencies when values mutually reference each other.
func (f FlatValue) Link(v Value, ctx *InflateContext) error {
	switch {
	case f.Function != nil:
		return f.Function.Link(v, ctx)
	case f.Class != nil:
		return f.Class.Link(v, ctx)
	case f.Module != nil:
		return f.Module.Link(v, ctx)
	case f.Package != nil:
		return f.Package.Link(v, ctx)
	case f.KwargDict != nil:
		return f.KwargDict.Link(v, ctx)
	default:
		return nil
	}
}

// FlatScalar is the representation of the scalar instance types used during
// serialization. It is an enumeration.
type FlatScalar int

// Inflate converts a flat scalar into a Value
func (f FlatScalar) Inflate(i *Inflater) Value {
	switch f {
	case FlatUnknown:
		return nil
	case FlatNone:
		return Builtins.None
	case FlatBool:
		return BoolInstance{}
	case FlatInt:
		return IntInstance{}
	case FlatFloat:
		return FloatInstance{}
	case FlatComplex:
		return ComplexInstance{}
	case FlatStr:
		return StrInstance{}
	default:
		rollbar.Error(fmt.Errorf("invalid FlatScalar"), f)
		return nil
	}
}

// FlatConstant is the representation for the scalar constant types used for
// serialization. Exactly one of its members is non-nil.
// We wrap all the constant values in structs since pointers to Go primitives
// don't play nicely with gob ser/des
type FlatConstant struct {
	Bool    *struct{ Val BoolConstant }
	Int     *struct{ Val IntConstant }
	Float   *struct{ Val FloatConstant }
	Complex *struct{ Val ComplexConstant }
	Str     *struct{ Val StrConstant }
}

// Inflate converts a flat scalar into a Value
func (f FlatConstant) Inflate(i *Inflater) Value {
	switch {
	case f.Bool != nil:
		return f.Bool.Val
	case f.Int != nil:
		return f.Int.Val
	case f.Float != nil:
		return f.Float.Val
	case f.Complex != nil:
		return f.Complex.Val
	case f.Str != nil:
		return f.Str.Val
	default:
		rollbar.Error(fmt.Errorf("invalid FlatConstant: all fields were nil"))
		return nil
	}
}

// FlatUnion is the representation of unions used for serialization
type FlatUnion struct {
	Constituents []FlatID
}

// Inflate creates a value from a flat value
func (f FlatUnion) Inflate(r *Inflater) Value {
	vs := make([]Value, len(f.Constituents))
	for i, id := range f.Constituents {
		vs[i] = r.Inflate(id)
	}
	return Union{vs}
}

// FlatList is the representation of unions used for serialization
type FlatList struct {
	Element FlatID
}

// Inflate creates a value from a flat value
func (f FlatList) Inflate(r *Inflater) Value {
	return NewList(r.Inflate(f.Element))
}

// FlatDict is the representation of unions used for serialization
type FlatDict struct {
	Key     FlatID
	Element FlatID
}

// Inflate creates a value from a flat value
func (f FlatDict) Inflate(r *Inflater) Value {
	return NewDict(r.Inflate(f.Key), r.Inflate(f.Element))
}

// FlatDataFrameInstance is the representation of pandas.DataFrame used for serialization
type FlatDataFrameInstance struct {
	FlatDict
}

// Inflate creates a DataFrameInstance value from a flat representation
func (f FlatDataFrameInstance) Inflate(r *Inflater) Value {
	return NewDataFrameInstanceFromGraph(r.Inflate(f.Key), r.Inflate(f.Element), r.Graph)
}

// FlatDataFrame is a Flat representation of the DataFrame type
// It doesn't contains anything as it just need an access to the resource manager at runtime
// to access the default representation of a DataFrame
type FlatDataFrame struct {
}

// Inflate creates a DataFrame from its flat representation, it used the resource manager available in the inflater
// for that
func (df FlatDataFrame) Inflate(r *Inflater) Value {
	return NewDataFrame(r.Graph)
}

// FlatSet is the representation of unions used for serialization
type FlatSet struct {
	Element FlatID
}

// Inflate creates a value from a flat value
func (f FlatSet) Inflate(r *Inflater) Value {
	return NewSet(r.Inflate(f.Element))
}

// FlatTuple is the representation of tuples used for serialization
type FlatTuple struct {
	Elements []FlatID
}

// Inflate creates a value from a flat value
func (f FlatTuple) Inflate(r *Inflater) Value {
	var vs []Value
	for _, id := range f.Elements {
		vs = append(vs, r.Inflate(id))
	}
	return NewTuple(vs...)
}

// FlatSuper is the representation of super instances used for serialization
type FlatSuper struct {
	Bases    []FlatID
	Instance FlatID
}

// Inflate creates a value from a flat value
func (f FlatSuper) Inflate(r *Inflater) Value {
	var bases []Value
	for _, b := range f.Bases {
		bases = append(bases, r.Inflate(b))
	}
	return SuperInstance{Bases: bases, Instance: r.Inflate(f.Instance)}
}

// FlatProperty is the representation of PropertyInstance for serialization
type FlatProperty struct {
	FGet FlatID
	FSet FlatID
}

// Inflate creates a value from a flat value
func (f FlatProperty) Inflate(r *Inflater) Value {
	return NewPropertyInstance(r.Inflate(f.FGet), r.Inflate(f.FSet))
}

// FlatPropertyUpdater is the representation of PropertyUpdater for serialization
type FlatPropertyUpdater struct {
	Which    string
	Instance FlatID
}

// Inflate creates a value from a flat value
func (f FlatPropertyUpdater) Inflate(r *Inflater) Value {
	inst, ok := r.Inflate(f.Instance).(PropertyInstance)
	if !ok {
		rollbar.Error(fmt.Errorf("expected FlatPropertyUpdater.Instance to inflate to PropertyInstance"))
		return nil
	}
	return PropertyUpdater{
		Which:    f.Which,
		Instance: inst,
	}
}

// FlatParameter is the representation of function parameters used for serialization,
// we do not store the symbol because this is already serialized
// as part of the symbol table for the function.
type FlatParameter struct {
	Name        string
	Default     FlatID
	KeywordOnly bool
}

// FlatFunction is the representation of functions used for serialization
type FlatFunction struct {
	Params           []FlatParameter
	Vararg           *FlatParameter
	Kwarg            *FlatParameter
	KwargDict        FlatID
	Return           FlatSymbol
	HasReceiver      bool
	HasClassReceiver bool
	Locals           FlatSymbolTable
}

// Inflate creates a value from a flat value
func (f FlatFunction) Inflate(i *Inflater) Value {
	// TODO(juan): kind of nasty
	sf := &SourceFunction{
		Locals: NewSymbolTable(f.Locals.Name, nil),
	}
	i.tables[f.Locals.Name.String()] = sf.Locals

	// leave rest to the link step to avoid circular dependencies
	return sf
}

// Link connects a value to other values during unflattening
func (f FlatFunction) Link(v Value, ctx *InflateContext) error {
	c := v.(*SourceFunction)
	c.HasReceiver = f.HasReceiver
	c.HasClassReceiver = f.HasClassReceiver

	// TODO(juan): kind of nasty
	c.Locals.Parent = ctx.Tables[f.Locals.Parent.String()]
	for _, fs := range f.Locals.Table {
		c.Locals.Table[fs.Name.Path.Last()] = &Symbol{
			Name:    fs.Name,
			Value:   ctx.Local[fs.Value],
			Private: fs.Private,
		}
	}

	c.Return = &Symbol{
		Name:    f.Return.Name,
		Value:   ctx.Local[f.Return.Value],
		Private: f.Return.Private,
	}
	for _, param := range f.Params {
		c.Parameters = append(c.Parameters, Parameter{
			Name:        param.Name,
			Default:     ctx.Local[param.Default],
			Symbol:      c.Locals.Table[param.Name],
			KeywordOnly: param.KeywordOnly,
		})
	}
	if f.Vararg != nil {
		c.Vararg = &Parameter{
			Name:        f.Vararg.Name,
			Symbol:      c.Locals.Table[f.Vararg.Name],
			KeywordOnly: f.Vararg.KeywordOnly,
		}
	}
	if f.Kwarg != nil {
		val := ctx.Local[f.KwargDict]
		dict, ok := val.(*KwargDict)
		if !ok {
			return fmt.Errorf("Kwarg for %v was %T but expected KwargDict", c.Locals.Name, val)
		}
		c.Kwarg = &Parameter{
			Name:    f.Kwarg.Name,
			Default: ctx.Local[f.Kwarg.Default],
			Symbol:  c.Locals.Table[f.Kwarg.Name],
		}
		c.KwargDict = dict
	}

	return nil
}

// FlatMember is the representation of attributes used for serialization
type FlatMember struct {
	Name  string
	Value FlatID
}

// FlatClass is the representation of classes used for serialization
type FlatClass struct {
	Bases      []FlatID
	Subclasses []FlatID
	Members    FlatSymbolTable
}

// Inflate creates a value from a flat value
func (f FlatClass) Inflate(i *Inflater) Value {
	// TODO(juan): nasty
	sc := &SourceClass{
		Members: NewSymbolTable(f.Members.Name, nil),
	}
	i.tables[f.Members.Name.String()] = sc.Members

	// leave rest to the link step to avoid circular dependencies
	return sc
}

// Link connects a value to other values during unflattening
func (f FlatClass) Link(v Value, ctx *InflateContext) error {
	c := v.(*SourceClass)
	for _, subID := range f.Subclasses {
		if sub := ctx.Local[subID]; sub != nil {
			if cls, ok := sub.(*SourceClass); ok {
				c.Subclasses = append(c.Subclasses, cls)
			} else {
				return fmt.Errorf("expected only subclasses of type SourceClass, got type %T (%v)", sub, sub)
			}
		}
	}

	// TODO(juan): kind of nasty
	c.Members.Parent = ctx.Tables[f.Members.Parent.String()]
	for _, fs := range f.Members.Table {
		c.Members.Table[fs.Name.Path.Last()] = &Symbol{
			Name:    fs.Name,
			Value:   ctx.Local[fs.Value],
			Private: fs.Private,
		}
	}

	for _, baseID := range f.Bases {
		if base := ctx.Local[baseID]; base != nil {
			c.Bases = append(c.Bases, base)
		}
	}
	return nil
}

// FlatInstance is the representation of class instances used for serialization
type FlatInstance struct {
	Class FlatID
}

// Inflate creates a value from a flat value
func (f FlatInstance) Inflate(i *Inflater) Value {
	// we don't need to delay this inflation, since circular dependencies are broken at the SourceClass members
	v := i.Inflate(f.Class)
	class, ok := v.(*SourceClass)
	if !ok {
		rollbar.Error(fmt.Errorf("FlatInstance Class not SourceClass"), fmt.Sprintf("%T", v))
		return nil
	}
	return SourceInstance{class}
}

// FlatModule is the representation of modules used for serialization
type FlatModule struct {
	Members FlatSymbolTable
}

// Inflate creates a value from a flat value
func (f FlatModule) Inflate(i *Inflater) Value {
	// TODO(juan): nasty
	sm := &SourceModule{
		Members: NewSymbolTable(f.Members.Name, nil),
	}
	i.tables[sm.Members.Name.String()] = sm.Members

	// leave rest to the link step to avoid circular dependencies
	return sm
}

// Link connects a value to other values during unflattening
func (f FlatModule) Link(v Value, ctx *InflateContext) error {
	c := v.(*SourceModule)

	// TODO(juan): nasty
	c.Members.Parent = ctx.Tables[f.Members.Parent.String()]
	for _, fs := range f.Members.Table {
		c.Members.Table[fs.Name.Path.Last()] = &Symbol{
			Name:    fs.Name,
			Value:   ctx.Local[fs.Value],
			Private: fs.Private,
		}
	}

	return nil
}

// FlatPackage is the representation of packages used for serialization
type FlatPackage struct {
	// LowerCase if attribute lookups should be lowercased before checking DirEntries
	LowerCase  bool
	DirEntries FlatSymbolTable
	Init       FlatID // Init is the ID of the __init__.py module for this dir
}

// Inflate creates a value from a flat value
func (f FlatPackage) Inflate(i *Inflater) Value {
	// TODO(juan): nasty
	sp := &SourcePackage{
		LowerCase:  f.LowerCase,
		DirEntries: NewSymbolTable(f.DirEntries.Name, nil),
	}
	i.tables[sp.DirEntries.Name.String()] = sp.DirEntries

	// leave rest to the link step to avoid circular dependencies
	return sp
}

// Link connects a value to other values during unflattening
func (f FlatPackage) Link(v Value, ctx *InflateContext) error {
	c := v.(*SourcePackage)
	initval := ctx.Local[f.Init]
	if initval != nil {
		init, ok := initval.(*SourceModule)
		if !ok {
			return fmt.Errorf("FlatPackage.Init was a %T: %v", initval, initval)
		}
		c.Init = init
	}

	// TODO(juan): nasty
	c.DirEntries.Parent = ctx.Tables[f.DirEntries.Parent.String()]
	for _, fs := range f.DirEntries.Table {
		c.DirEntries.Table[fs.Name.Path.Last()] = &Symbol{
			Name:    fs.Name,
			Value:   ctx.Local[fs.Value],
			Private: fs.Private,
		}
	}

	return nil
}

// FlatExplicit is the flat representation of ExplicitType and ExplicitFunc
type FlatExplicit struct {
	Path pythonimports.DottedPath
}

// Inflate creates a value from a flat value
func (f FlatExplicit) Inflate(r *Inflater) Value {
	v, found := singletons[f.Path.Hash]
	if !found {
		rollbar.Error(fmt.Errorf("no singleton value registered for path"), fmt.Sprintf("%s", f.Path))
		return nil
	}
	return v
}

// FlatExternalRoot represents an external root
type FlatExternalRoot struct{}

// Inflate creates a value from a flat value
func (f FlatExternalRoot) Inflate(r *Inflater) Value {
	return ExternalRoot{Graph: r.Graph}
}

// FlatExternal provides a link between the local and global graph
type FlatExternal struct {
	Path pythonimports.DottedPath
}

// Inflate creates a value from a flat value
func (f FlatExternal) Inflate(r *Inflater) Value {
	sym, err := r.Graph.PathSymbol(f.Path)
	if err != nil {
		log.Printf(err.Error())
		return nil
	}
	return TranslateExternal(sym, r.Graph)
}

// FlatExternalReturnValue ...
type FlatExternalReturnValue struct {
	Fn pythonimports.DottedPath
}

// Inflate creates a value from a flat value
func (f FlatExternalReturnValue) Inflate(r *Inflater) Value {
	sym, err := r.Graph.PathSymbol(f.Fn)
	if err != nil {
		log.Printf(err.Error())
		return nil
	}
	return ExternalReturnValue{fn: sym, graph: r.Graph}
}

// FlatExternalInstance provides a link between the local and global graph
type FlatExternalInstance struct {
	Path pythonimports.DottedPath
}

// Inflate creates a value from a flat value
func (f FlatExternalInstance) Inflate(r *Inflater) Value {
	sym, err := r.Graph.PathSymbol(f.Path)
	if err != nil {
		log.Printf(err.Error())
		return nil
	}
	return TranslateExternalInstance(sym, r.Graph)
}

func addr(x FlatScalar) *FlatScalar {
	return &x
}

// A Flattener recursively flattens values to FlatValues
// TODO(naman) this should be private
type Flattener struct {
	seen   map[FlatID]struct{}
	values []*FlatValue
	ctx    kitectx.Context
}

// Flatten gets the ID for the given value if is has already been flattened,
// or else flattens it and returns a newly allocated ID.
func (r *Flattener) Flatten(v Value) FlatID {
	// do not include a FlatValue for the nil Value
	if v == nil {
		return 0
	}

	r.ctx.CheckAbort()
	id := MustHash(v)
	if _, done := r.seen[id]; done {
		return id
	}

	f := FlatValue{ID: id}
	r.values = append(r.values, &f)
	r.seen[id] = struct{}{}

	v.Flatten(&f, r)
	if err := f.Check(); err != nil {
		panic(fmt.Sprintf("flattening %v (%T): %v", v, v, err))
	}

	return id
}

// FlattenValues flattens the given values
func FlattenValues(ctx kitectx.Context, vs []Value) (fs []*FlatValue, err error) {
	ctx.CheckAbort()
	f := Flattener{
		seen: make(map[FlatID]struct{}),
		ctx:  ctx,
	}
	for _, v := range vs {
		f.Flatten(v)
	}
	return f.values, nil
}

// InflateContext represents the external ctx that values get linked to
type InflateContext struct {
	Local  map[FlatID]Value
	Tables map[string]*SymbolTable
}

type link struct {
	v Value
	f *FlatValue
}

// Inflater is responsible for converting flat value into values
type Inflater struct {
	flat    map[FlatID]*FlatValue
	values  map[FlatID]Value
	seen    map[FlatID]struct{}
	missing []FlatID
	Graph   pythonresource.Manager
	tables  map[string]*SymbolTable
}

// Inflate gets the value for the given ID
func (i *Inflater) Inflate(id FlatID) Value {
	// there is no FlatValue for the nil value so do not look for it
	if id == 0 {
		return nil
	}

	if v, found := i.values[id]; found {
		return v
	}
	if _, isseen := i.seen[id]; isseen {
		panic("inflate loop")
	}
	f, ok := i.flat[id]
	if !ok {
		i.missing = append(i.missing, id)
		return nil
	}

	i.seen[id] = struct{}{}
	v := f.Inflate(i)
	i.values[id] = v
	return v
}

// InflateValues converts flat values to values
func InflateValues(fs []*FlatValue, graph pythonresource.Manager) (map[FlatID]Value, error) {
	inflater := Inflater{
		flat:   make(map[FlatID]*FlatValue),
		values: make(map[FlatID]Value),
		seen:   make(map[FlatID]struct{}),
		Graph:  graph,
		tables: make(map[string]*SymbolTable),
	}
	for _, f := range fs {
		inflater.flat[f.ID] = f
	}
	for _, f := range fs {
		inflater.Inflate(f.ID)
	}
	ctx := InflateContext{
		Local:  inflater.values,
		Tables: inflater.tables,
	}
	for _, f := range fs {
		f.Link(inflater.values[f.ID], &ctx)
	}
	if len(inflater.missing) > 0 {
		return nil, fmt.Errorf("%d IDs missing in InflateValues: %v", len(inflater.missing), inflater.missing)
	}
	return inflater.values, nil
}

func flattenDict(dict map[string]Value, r *Flattener) []FlatMember {
	var members []FlatMember
	for attr, val := range dict {
		members = append(members, FlatMember{Name: attr, Value: r.Flatten(val)})
	}
	return members
}

func inflateDict(members []FlatMember, ctx *InflateContext) map[string]Value {
	dict := make(map[string]Value)
	for _, member := range members {
		dict[member.Name] = ctx.Local[member.Value]
	}
	return dict
}

func flattenSymbolTable(table *SymbolTable, r *Flattener) FlatSymbolTable {
	var symbols []FlatSymbol
	for _, sym := range table.Table {
		symbols = append(symbols, FlatSymbol{
			Name:    sym.Name,
			Value:   r.Flatten(sym.Value),
			Private: sym.Private,
		})
	}

	var parent Address
	if table.Parent != nil {
		parent = table.Parent.Name
	}

	return FlatSymbolTable{
		Parent: parent,
		Name:   table.Name,
		Table:  symbols,
	}
}

func flattenParameter(p Parameter, r *Flattener) FlatParameter {
	return FlatParameter{
		Name:        p.Name,
		Default:     r.Flatten(p.Default),
		KeywordOnly: p.KeywordOnly,
	}
}
