package pythonimports

import "github.com/kiteco/kiteco/kite-golib/errors"

// mockBuiltins is a map of the builtins to add to mock graphs
var mockBuiltins = map[string]Kind{
	"buitins":             Module,
	"builtins.bool":       Type,
	"builtins.int":        Type,
	"builtins.float":      Type,
	"builtins.complex":    Type,
	"builtins.str":        Type,
	"builtins.list":       Type,
	"builtins.tuple":      Type,
	"builtins.dict":       Type,
	"builtins.set":        Type,
	"builtins.print":      Function,
	"builtins.type":       Type,
	"builtins.object":     Type,
	"builtins.map":        Function,
	"builtins.super":      Type,
	"builtins.all":        Function,
	"builtins.any":        Function,
	"builtins.bin":        Function,
	"builtins.memoryview": Function,
	"builtins.chr":        Function,
	"builtins.compile":    Function,
	"builtins.credits":    Function,
	"builtins.delattr":    Function,
	"builtins.dir":        Function,
	"builtins.eval":       Function,
	"builtins.exit":       Function,
	"builtins.format":     Function,
	"builtins.globals":    Function,
	"builtins.hash":       Function,
	"builtins.hex":        Function,
	"builtins.id":         Function,
	"builtins.input":      Function,
	"builtins.isinstance": Function,
	"builtins.issubclass": Function,
	"builtins.len":        Function,
	"builtins.license":    Function,
	"builtins.oct":        Function,
	"builtins.open":       Function,
	"builtins.ord":        Function,
	"builtins.quit":       Function,
	"builtins.range":      Function,
	"builtins.repr":       Function,
	"builtins.round":      Function,
	"builtins.setattr":    Function,
	"builtins.bytes":      Function,
	"builtins.vars":       Function,
	"builtins.divmod":     Function,
	"builtins.enumerate":  Function,
	"builtins.getattr":    Function,
	"builtins.max":        Function,
	"builtins.min":        Function,
	"builtins.next":       Function,
	"builtins.iter":       Function,
	"builtins.pow":        Function,
	"builtins.reversed":   Function,
	"builtins.sorted":     Function,
	"builtins.sum":        Function,
	"builtins.filter":     Function,
	"builtins.zip":        Function,
}

// BuiltinCache is a cache of commonly referenced nodes from the import graph
type BuiltinCache struct {
	BuiltinPkg *Node
	Bool       *Node
	Int        *Node
	Long       *Node
	Float      *Node
	Complex    *Node
	Str        *Node
	List       *Node
	Tuple      *Node
	Dict       *Node
	Set        *Node
	Print      *Node
	Type       *Node
	Object     *Node
	Map        *Node
	None       *Node
	TypesPkg   *Node
	Generator  *Node
	Function   *Node
	Module     *Node
}

// NewBuiltinCache constructs a BuiltinCache from an import graph and returns an error
// if any node is not found.
func NewBuiltinCache(graph *Graph) (*BuiltinCache, error) {
	var cache BuiltinCache

	if cache.BuiltinPkg = graph.PkgToNode["builtins"]; cache.BuiltinPkg == nil {
		return nil, errors.Errorf("unable to find builtins in import graph")
	}

	if cache.Bool = cache.BuiltinPkg.Members["bool"]; cache.Bool == nil {
		return nil, errors.Errorf("unable to find builtins.bool in import graph")
	}

	if cache.Int = cache.BuiltinPkg.Members["int"]; cache.Int == nil {
		return nil, errors.Errorf("unable to find builtins.int in import graph")
	}

	if cache.Float = cache.BuiltinPkg.Members["float"]; cache.Float == nil {
		return nil, errors.Errorf("unable to find builtins.float in import graph")
	}

	if cache.Complex = cache.BuiltinPkg.Members["complex"]; cache.Complex == nil {
		return nil, errors.Errorf("unable to find builtins.complex in import graph")
	}

	if cache.Str = cache.BuiltinPkg.Members["str"]; cache.Str == nil {
		return nil, errors.Errorf("unable to find builtins.str in import graph")
	}

	if cache.List = cache.BuiltinPkg.Members["list"]; cache.List == nil {
		return nil, errors.Errorf("unable to find builtins.list in import graph")
	}

	if cache.Tuple = cache.BuiltinPkg.Members["tuple"]; cache.Tuple == nil {
		return nil, errors.Errorf("unable to find builtins.tuple in import graph")
	}

	if cache.Dict = cache.BuiltinPkg.Members["dict"]; cache.Dict == nil {
		return nil, errors.Errorf("unable to find builtins.dict in import graph")
	}

	if cache.Set = cache.BuiltinPkg.Members["set"]; cache.Set == nil {
		return nil, errors.Errorf("unable to find builtins.set in import graph")
	}

	if cache.Print = cache.BuiltinPkg.Members["print"]; cache.Print == nil {
		return nil, errors.Errorf("unable to find builtins.print in import graph")
	}

	if cache.Type = cache.BuiltinPkg.Members["type"]; cache.Type == nil {
		return nil, errors.Errorf("unable to find builtins.type in import graph")
	}

	if cache.Object = cache.BuiltinPkg.Members["object"]; cache.Object == nil {
		return nil, errors.Errorf("unable to find builtins.object in import graph")
	}

	if cache.Map = cache.BuiltinPkg.Members["map"]; cache.Map == nil {
		return nil, errors.Errorf("unable to find builtins.map in import graph")
	}

	if cache.None = cache.BuiltinPkg.Members["None"]; cache.None == nil {
		return nil, errors.Errorf("unable to find builtins.None in import graph")
	}

	if cache.TypesPkg = graph.PkgToNode["types"]; cache.TypesPkg == nil {
		return nil, errors.Errorf("unable to find types in import graph")
	}

	if cache.Generator = cache.TypesPkg.Members["GeneratorType"]; cache.Generator == nil {
		return nil, errors.Errorf("unable to find types.GeneratorType in import graph")
	}

	if cache.Function = cache.TypesPkg.Members["FunctionType"]; cache.Function == nil {
		return nil, errors.Errorf("unable to find types.FunctionType in import graph")
	}

	if cache.Module = cache.TypesPkg.Members["ModuleType"]; cache.Module == nil {
		return nil, errors.Errorf("unable to find types.ModuleType in import graph")
	}

	return &cache, nil
}
