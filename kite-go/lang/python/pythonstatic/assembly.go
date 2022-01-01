package pythonstatic

import (
	"fmt"
	"io"
	"log"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// DefaultOptions are the default options for the Assembler
var DefaultOptions = Options{
	PrivateImports:  false,
	Passes:          3,
	UseCapabilities: true,
}

// Options represents the options for the Assembler
type Options struct {
	// PrivateImports excepting `from .. import *` & `from .. import .. as ..`
	// only relevant for stub analysis: see typeshed/CONTRIBUTING.md
	PrivateImports bool
	// UseCapabilities to perform typechecking and type inference
	UseCapabilities bool
	// Passes is the number of propagation passes to make over the data
	Passes int
	// allowValueMutation allow/block updating the key map in dict values
	// The are only allowed when building local index to avoid any race condition
	AllowValueMutation bool
}

// ASTBundle represents a source file's AST bundled with some precomputed metadata
type ASTBundle struct {
	AST *pythonast.Module
	// Windows if the file is from a Windows machine
	Windows bool
	// Path to the file for the AST
	Path string
	// Imports is a slice of imports collected from AST
	Imports []ImportPath
	// LibraryPath is true if the AST should be considered a library
	LibraryPath string
}

// File represents a source file
type File struct {
	ASTBundle ASTBundle
	Module    *pythontype.SourceModule
}

// Assembly represents the functions, classes, and modules defined in one or more
// source files
type Assembly struct {
	PythonPaths    map[string]struct{} // paths to consider import "roots" for e.g. libraries
	Files          map[string]*File    // Files is keyed by absolute path, and includes the ".py"
	Modules        map[*pythonast.Module]*pythontype.SourceModule
	Classes        map[*pythonast.ClassDefStmt]*pythontype.SourceClass
	Functions      map[*pythonast.FunctionDefStmt]*pythontype.SourceFunction
	Lambdas        map[*pythonast.LambdaExpr]*pythontype.SourceFunction
	Comprehensions map[pythonast.Comprehension]*pythontype.SymbolTable
	Sources        *pythonenv.SourceTree
	PropagateOrder []pythonast.Node
}

// NewAssembly creates an empty assembly.
func NewAssembly() *Assembly {
	assembly := &Assembly{
		PythonPaths:    make(map[string]struct{}),
		Files:          make(map[string]*File),
		Modules:        make(map[*pythonast.Module]*pythontype.SourceModule),
		Classes:        make(map[*pythonast.ClassDefStmt]*pythontype.SourceClass),
		Functions:      make(map[*pythonast.FunctionDefStmt]*pythontype.SourceFunction),
		Lambdas:        make(map[*pythonast.LambdaExpr]*pythontype.SourceFunction),
		Comprehensions: make(map[pythonast.Comprehension]*pythontype.SymbolTable),
		Sources:        pythonenv.NewSourceTree(),
	}
	return assembly
}

// WalkSymbols calls the provided function for each symbol in the assembly
func (a *Assembly) WalkSymbols(f func(*pythontype.Symbol)) {
	visit := func(syms *pythontype.SymbolTable) {
		for _, sym := range syms.Table {
			f(sym)
		}
	}
	for _, pkg := range a.Sources.Dirs {
		visit(pkg.DirEntries)
	}
	for _, mod := range a.Sources.Files {
		visit(mod.Members)
	}
	for _, class := range a.Classes {
		visit(class.Members)
	}
	for _, fun := range a.Functions {
		visit(fun.Locals)
		f(fun.Return)
	}
	for _, lambda := range a.Lambdas {
		visit(lambda.Locals)
		f(lambda.Return)
	}
}

// Assembler is responsible for analyzing python source files and outputting an Assembly
type Assembler struct {
	user         int64
	machine      string
	assembly     *Assembly
	builtinScope *pythontype.SymbolTable
	helpers      *helpers
	extImporter  *Importer
}

// BuiltinScope creates a symbol table containing builtins
func BuiltinScope(ctx kitectx.Context, graph pythonresource.Manager) *pythontype.SymbolTable {
	ctx.CheckAbort()

	builtin, err := graph.PathSymbol(pythonimports.NewDottedPath("builtins"))
	if err != nil {
		panic(errors.Errorf("unable to find python 3 builtins: %v", err))
	}

	// Create a SymbolTable to mirror the __builtins__ package
	builtinScope := pythontype.NewSymbolTable(pythontype.Address{Path: pythonimports.NewPath("builtins")}, nil)

	attrs, err := graph.Children(builtin)
	if err != nil {
		panic(errors.Errorf("unable to find children for `%s`: %v", builtin, err))
	}

	for _, attr := range attrs {
		child, err := graph.ChildSymbol(builtin, attr)
		if err != nil {
			rollbar.Error(errors.Wrapf(err, "attribute missing from builtins"), fmt.Sprintf("attr: `%s`", attr))
			continue
		}
		s := builtinScope.LocalOrCreate(attr)
		s.Value = pythontype.Unite(ctx, s.Value, pythontype.TranslateExternal(child, graph))
	}

	// Override with symbols from pythontype.BuiltinSymbols
	for attr, val := range pythontype.BuiltinSymbols {
		builtinScope.Put(attr, val)
	}
	return builtinScope
}

// AssemblerInputs bundles the inputs for the assmebler.
type AssemblerInputs struct {
	User     int64
	Machine  string
	Graph    pythonresource.Manager
	Importer *Importer
	Delegate PropagatorDelegate
}

// NewAssembler constructs an empty assembler. Graph must not be nil.
func NewAssembler(ctx kitectx.Context, ai AssemblerInputs, opts Options) *Assembler {
	assembly := NewAssembly()
	return &Assembler{
		user:         ai.User,
		machine:      ai.Machine,
		assembly:     assembly,
		builtinScope: BuiltinScope(ctx, ai.Graph),
		helpers: &helpers{
			ResourceManager: ai.Graph,
			Assembly:        assembly,
			Delegate:        ai.Delegate,
			Opts:            opts,
		},
		extImporter: ai.Importer,
	}
}

// SetTrace starts writing trace output to the given writer for the data flow analysis
func (b *Assembler) SetTrace(w io.Writer) {
	b.helpers.TraceWriter = w
}

func (b *Assembler) trace(format string, objs ...interface{}) {
	if b.helpers.TraceWriter != nil {
		fmt.Fprintf(b.helpers.TraceWriter, format+"\n", objs...)
	}
}

// AddSource adds a source file to the assembly
func (b *Assembler) AddSource(bundle ASTBundle) {
	if bundle.LibraryPath != "" {
		b.assembly.PythonPaths[bundle.LibraryPath] = struct{}{}
	}

	if _, seen := b.assembly.Sources.Files[bundle.Path]; seen {
		log.Printf("Assembler.AddSource received %s twice", bundle.Path)
		return
	}

	mod := CreateModule(pythontype.Address{User: b.user, Machine: b.machine, File: bundle.Path}, b.builtinScope)
	b.assembly.Sources.AddFile(bundle.Path, mod, bundle.Windows)
	b.assembly.Files[bundle.Path] = &File{
		ASTBundle: bundle,
		Module:    mod,
	}

	b.assembly.Modules[bundle.AST] = mod
}

func (b *Assembler) importer(path string) Importer {
	if b.extImporter != nil {
		return *b.extImporter
	}

	return Importer{
		Path:        path,
		PythonPaths: b.assembly.PythonPaths,
		Global:      b.helpers.ResourceManager,
		Local:       b.assembly.Sources,
	}
}

// Build propagates types through all functions, classes, and modules, for a fixed
// number of iterations, the provided delegate is attached on the final pass.
// NOTE: delegate can be nil.
// NOTE: the number of iterations is determined by Options.Passes.
func (b *Assembler) Build(ctx kitectx.Context) (assembly *Assembly, err error) {
	ctx.CheckAbort()

	b.bootstrap()
	a := Analyzer{
		helpers: b.helpers,
	}

	for i := 0; i < b.helpers.Opts.Passes; i++ {
		b.trace("\n### PROPAGATION PASS %d", i)
		if a.Delegate != nil {
			a.Delegate.Pass(i, b.helpers.Opts.Passes)
		}

		if b.helpers.Opts.UseCapabilities {
			b.helpers.CapabilityDelegate = newCapabilityDelegate()
		}

		// this must be an explicit for loop because the list grows during the loop
		for j := 0; j < len(b.assembly.PropagateOrder); j++ {
			switch n := b.assembly.PropagateOrder[j].(type) {
			case *pythonast.Module:
				module := b.assembly.Modules[n]
				imp := b.importer(module.Members.Name.File)
				b.trace("\n### PROPAGATING MODULE %s ###", module)
				a.Module(ctx, module, n, imp)
			case *pythonast.FunctionDefStmt:
				fun := b.assembly.Functions[n]
				imp := b.importer(fun.Locals.Name.File)
				b.trace("\n### PROPAGATING FUNCTION %s ###", fun)
				a.Function(ctx, fun, n, imp)
			case *pythonast.LambdaExpr:
				lambda := b.assembly.Lambdas[n]
				imp := b.importer(lambda.Locals.Name.File)
				b.trace("\n### PROPAGATING LAMBDA ###")
				a.Lambda(ctx, lambda, n, imp)
			default:
				panic(fmt.Sprintf("encountered %T in propagate order", b.assembly.PropagateOrder[j]))
			}
		}

		if i == b.helpers.Opts.Passes-1 && b.helpers.Opts.UseCapabilities {
			refineUnions(ctx, b.helpers.CapabilityDelegate.capabilities)
		}
	}
	return b.assembly, nil
}

// GetCapabilitiesRecord returns the list of capabilities collected
// It returns nil if CapabilityDelegate hasn't been enable
func (b *Assembler) GetCapabilitiesRecord() map[*pythontype.Symbol][]Capability {
	if b.helpers.CapabilityDelegate != nil {
		return b.helpers.CapabilityDelegate.capabilities
	}
	return nil
}
