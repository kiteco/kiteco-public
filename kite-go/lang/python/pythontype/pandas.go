package pythontype

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	dataFrameAddress    = SplitAddress(dataFrameStringAddr)
	dataFrameStringAddr = "pandas.core.frame.DataFrame"
	seriesStringAddr    = "pandas.core.series.Series"
)

// DataFrame represents the DataFrame type, it is backed up by the default DF representation stored in the resource manager
type DataFrame struct {
	defaultDFType  Value
	seriesInstance Value
}

// Kind implements Value
func (d DataFrame) Kind() Kind {
	return TypeKind
}

// Type implements Value
func (d DataFrame) Type() Value {
	return Builtins.Type
}

// Address implements Value
func (d DataFrame) Address() Address {
	return dataFrameAddress
}

// Flatten implements Value (used for serialization)
func (d DataFrame) Flatten(fv *FlatValue, flattener *Flattener) {
	fv.DataFrame = &FlatDataFrame{}
}

func (d DataFrame) hash(ctx kitectx.CallContext) FlatID {
	return rehash(saltType, FlatID(dataFrameAddress.Path.Hash))
}

func (d DataFrame) equal(ctx kitectx.CallContext, v Value) bool {
	if _, ok := v.(DataFrame); ok {
		return true
	}
	return false
}

// Call is the method used to create a DataFrameInstance from the DataFrame type
func (d DataFrame) Call(args Args) Value {
	return NewDataFrameInstance(nil, nil, d)
}

func (d DataFrame) attr(ctx kitectx.CallContext, s string) (AttrResult, error) {
	return d.defaultDFType.attr(ctx, s)
}

// NewDataFrame returns a representation of the DataFrame type
// It needs an access to the manager to get the default representation of DataFrame (used to access all the attributes)
func NewDataFrame(rm pythonresource.Manager) Value {
	dfExt, err := getDataFrameExternal(rm)
	if err != nil {
		// If we can't even build a plain external it means the RM doesn't know pandas
		return nil
	}
	seriesInstance, err := getSeriesInstance(rm)
	if err != nil {
		return nil
	}
	return DataFrame{
		defaultDFType:  dfExt,
		seriesInstance: seriesInstance,
	}
}

// DataFrameInstance holds information about the columns of a DataFrame to provide completion for them
// For all the attributes, the default DataFrame representation (from the RM) is used
type DataFrameInstance struct {
	dfType   DataFrame
	delegate DictInstance
}

// GetTrackedKeys returns the list of known keys for this DataFrame
func (df DataFrameInstance) GetTrackedKeys() map[ConstantValue]Value {
	return df.delegate.GetTrackedKeys()
}

func (df DataFrameInstance) hash(ctx kitectx.CallContext) FlatID {
	return df.delegate.hash(ctx)
}

func (df DataFrameInstance) equal(ctx kitectx.CallContext, val Value) bool {
	if df2, ok := val.(DataFrameInstance); ok {
		return df.delegate.equal(ctx, df2.delegate)
	}
	return false
}

// String provides a string representation of this value
func (df DataFrameInstance) String() string {
	return fmt.Sprintf("DataFrame %s", df.delegate.String())
}

func getDataFrameExternal(rm pythonresource.Manager) (External, error) {
	dataframeSymbol, err := rm.NewSymbol(keytypes.PandasDistribution, pythonimports.NewDottedPath(dataFrameStringAddr))
	if err != nil {
		return External{}, err
	}

	return External{
		symbol: dataframeSymbol,
		graph:  rm,
	}, nil
}

func getDefaultDFInstance(rm pythonresource.Manager) (ExternalInstance, error) {
	dataframeSymbol, err := rm.NewSymbol(keytypes.PandasDistribution, pythonimports.NewDottedPath(dataFrameStringAddr))
	if err != nil {
		return ExternalInstance{}, err
	}
	return ExternalInstance{NewExternal(dataframeSymbol.Canonical(), rm)}, nil
}

func getSeriesInstance(rm pythonresource.Manager) (Value, error) {
	seriesSymbol, err := rm.NewSymbol(keytypes.PandasDistribution, pythonimports.NewDottedPath(seriesStringAddr))
	if err != nil {
		return ExternalInstance{}, err
	}
	return TranslateExternalInstance(seriesSymbol, rm), nil
}

// NewDataFrameInstanceFromGraph builds a DataFrameInstance without requiring access to a DataFrame value.
// instead it get everything it needs from the manager to have access to the default representation of DataFrame
func NewDataFrameInstanceFromGraph(key, element Value, rm pythonresource.Manager) Value {
	ref, ok := NewDataFrame(rm).(DataFrame)
	if !ok {
		return nil
	}
	return NewDataFrameInstance(key, element, ref)
}

// NewDataFrameInstanceWithMapFromGraph is the same than NewDataFrameInstanceFromGraph but also initialize the map
// for the tracked keys
func NewDataFrameInstanceWithMapFromGraph(key, element Value, keyMap map[ConstantValue]Value, rm pythonresource.Manager) Value {
	ref, ok := NewDataFrame(rm).(DataFrame)
	if !ok {
		return nil
	}
	return NewDataFrameInstanceWithMap(key, element, keyMap, ref)
}

// NewDataFrameInstance build a new instance of a DataFrame (without trackedKey map)
func NewDataFrameInstance(key, element Value, ref DataFrame) Value {
	return DataFrameInstance{ref, NewDict(key, element).(DictInstance)}
}

// NewDataFrameInstanceWithMap build a new instance of a DataFrame with a map to hold the known keys
func NewDataFrameInstanceWithMap(key, element Value, keyMap map[ConstantValue]Value, ref DataFrame) Value {
	return DataFrameInstance{ref, NewDictWithMap(key, element, keyMap).(DictInstance)}
}

// Kind categorizes this value as function/type/module/instance/union/etc
func (DataFrameInstance) Kind() Kind { return InstanceKind }

// Type gets the result of calling type() on this value in python
func (df DataFrameInstance) Type() Value {
	return df.dfType.defaultDFType
}

// Address is the path for this value in the import graph
func (DataFrameInstance) Address() Address { return dataFrameAddress }

// Index allows DataFrameInstance to implement Indexable interface
func (df DataFrameInstance) Index(index Value, allowValueMutation bool) Value {
	if allowValueMutation {
		// To store the key if allowMutation is on
		df.delegate.Index(index, allowValueMutation)
	}
	switch index.(type) {
	case ListInstance:
		// We don't know what are the columns selected in the list, we return a default DF
		return NewDataFrameInstance(nil, nil, df.dfType)
	case BoolInstance:
		// It's just a filter, columns are the same we can return the same DF (only row are filtered)
		return df
	default:
		// It's not a list, big chance that it is just the name of a column, we return a Series
		return df.dfType.seriesInstance
	}
}

// SetIndex implements IndexAssignable for DataFrameInstance
func (df DataFrameInstance) SetIndex(index Value, value Value, allowValueMutation bool) Value {
	// A DataFrame always return a pandas.Series object when a key is accessed, so we store that directly in the map
	// instead of the actual value associated with the key
	updatedDict := df.delegate.SetIndex(index, df.dfType.seriesInstance, allowValueMutation).(DictInstance)
	return NewDataFrameInstanceWithMap(updatedDict.Key, updatedDict.Element, updatedDict.TrackedKeys, df.dfType)
}

// Flatten creates a flat version of this value
func (df DataFrameInstance) Flatten(f *FlatValue, r *Flattener) {
	f.DataFrameInstance = &FlatDataFrameInstance{FlatDict{
		Key:     r.Flatten(df.delegate.Key),
		Element: r.Flatten(df.delegate.Element),
	}}
}

func (df DataFrameInstance) attr(ctx kitectx.CallContext, name string) (AttrResult, error) {
	ctx.CheckAbort()
	return resolveAttr(ctx, name, df, nil, df.dfType.defaultDFType)
}
