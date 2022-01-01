package pythonbatch

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonindex"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/pkg/errors"
)

var (
	// DefaultOptions is a set of reasonable defaults for analysis
	DefaultOptions = Options{
		BuildTimeout: 30 * time.Second,
		Options:      pythonstatic.DefaultOptions,
		PathSelection: SelectionOptions{
			Parse: pythonparser.Options{
				ErrorMode: pythonparser.Recover,
			},
			// determined by the speed and memory requirements of static analysis
			ProjectFileLimit: 500,
			LibraryFileLimit: 100,
			SizeLimit:        1000000,
		},
	}

	// DefaultLocalOptions is a set of reasonable defaults for analysis for kite local
	DefaultLocalOptions = Options{
		BuildTimeout: 30 * time.Second,
		Local:        true,
		Options:      pythonstatic.DefaultOptions,
		PathSelection: SelectionOptions{
			Parse: pythonparser.Options{
				ErrorMode: pythonparser.Recover,
			},
			// determined by the speed and memory requirements of static analysis
			ProjectFileLimit: 500,
			LibraryFileLimit: 100,
			SizeLimit:        1000000,
		},
	}

	// ErrTooLarge is returned to indicate that a file is too large to analyze
	ErrTooLarge = errors.New("file too large")
)

// Options represents options for a batch manager
type Options struct {
	// BuildTimeout for the analysis portion of the build; 0 means no timeout.
	BuildTimeout time.Duration
	// TraceWriter is the writer to which to write the trace
	TraceWriter io.Writer
	// Options for static analysis
	Options pythonstatic.Options
	// PathSelection options
	PathSelection SelectionOptions
	// Local mode for kite local
	Local bool
}

// SourceUnit represents the result of analyzing a single python file
type SourceUnit struct {
	ASTBundle pythonstatic.ASTBundle
	Hash      string          // Hash is the hash from the files DB
	Contents  []byte          // Contents of the file
	Lines     *linenumber.Map // Lines contains the offset for each line
}

// Batch represents the results of analyzing a set of python files
type Batch struct {
	Assembly *pythonstatic.Assembly
	// Docs is a map from canonical name to documentation
	Docs map[string]pythonlocal.Documentation
	// Definitions is a map from canonical names to definitions
	Definitions map[string]pythonlocal.Definition
	// ArgSpecs is a map from canonical names to arg spec
	ArgSpecs map[string]pythonimports.ArgSpec
	// ValuesCount is a map from a value's locator to a counter used for ranking
	ValuesCount map[string]int
	// Methods is a map from a value's locator to its method pattners
	Methods map[string]*pythoncode.MethodPatterns
	// InvertedIndex is used for active search
	// TODO(naman) unused: rm unless we decide to turn local code search back on
	InvertedIndex map[string][]*pythonindex.IdentCount
}

// BatchManager manages a batch of files
type BatchManager struct {
	opts           Options
	user           int64
	machine        string
	delegate       *usageDelegate
	assembler      *pythonstatic.Assembler
	sources        map[string]*SourceUnit // sources is keyed by paths on the user's filesystem
	buildDurations map[string]time.Duration
}

// BatchInputs to build a batch of files.
type BatchInputs = pythonstatic.AssemblerInputs

// NewBatchManager returns a new batch manager
func NewBatchManager(ctx kitectx.Context, bi BatchInputs, opts Options, buildDurations map[string]time.Duration) *BatchManager {
	ctx.CheckAbort()

	if buildDurations == nil {
		buildDurations = make(map[string]time.Duration)
	}

	delegate := new(usageDelegate)
	bi.Delegate = delegate
	opts.Options.AllowValueMutation = true
	assembler := pythonstatic.NewAssembler(ctx, bi, opts.Options)
	assembler.SetTrace(opts.TraceWriter)
	return &BatchManager{
		opts:           opts,
		user:           bi.User,
		machine:        bi.Machine,
		delegate:       delegate,
		assembler:      assembler,
		sources:        make(map[string]*SourceUnit),
		buildDurations: buildDurations,
	}
}

// Add puts a new file into the batch. It also adds all ancestor directories.
// source.Hash, source.Contents, source.ASTBundle.Path, and source.ASTBundle.LibraryPath must be set;
// everything else will be computed if necessary.
func (b *BatchManager) Add(source *SourceUnit) error {
	if source.Lines == nil {
		source.Lines = linenumber.NewMap(source.Contents)
	}

	b.sources[source.ASTBundle.Path] = source
	b.assembler.AddSource(source.ASTBundle)
	return nil
}

// Build runs the resolver on the files in a batch
func (b *BatchManager) Build(ctx kitectx.Context) (batch *Batch, err error) {
	ctx.CheckAbort()

	ts := time.Now()
	assembly, err := b.assembler.Build(ctx)
	if err != nil {
		return nil, err
	}
	b.buildDurations["assembler_build"] = time.Since(ts)

	// Initialize docs, definitions, usages and argument specs
	docs := make(map[string]pythonlocal.Documentation, len(assembly.Modules)+len(assembly.Classes)+len(assembly.Functions))
	definitions := make(map[string]pythonlocal.Definition, len(assembly.Classes)+len(assembly.Functions))
	argspecs := make(map[string]pythonimports.ArgSpec, len(assembly.Functions))
	valuesCount := make(map[string]int)

	ts = time.Now()
	// Construct documentation for modules
	for syntax, module := range assembly.Modules {
		path := module.Members.Name.File

		id := pythonlocal.LookupID(module)
		if doc := pythonlocal.BuildDocumentation(path, module.Address().Path.Last(), module.Address().Path.String(), syntax.Body); doc != nil {
			docs[id] = *doc
		}
	}
	b.buildDurations["construct_modules"] = time.Since(ts)

	ts = time.Now()
	// Construct documentation and definitions for classes
	for syntax, class := range assembly.Classes {
		path := class.Members.Name.File
		source, found := b.sources[path]
		if !found {
			log.Printf("source not found for %s", path)
			continue
		}

		ident := syntax.Name.Ident.Literal

		id := pythonlocal.LookupID(class)
		doc := pythonlocal.BuildDocumentation(path, ident, class.Address().Path.String(), syntax.Body)
		if doc != nil {
			docs[id] = *doc
		}

		def := pythonlocal.BuildDefinition(
			path, syntax, source.Lines)
		if def != nil {
			definitions[id] = *def
		}
	}
	b.buildDurations["construct_classes"] = time.Since(ts)

	ts = time.Now()
	// Construct documentation, definitions and argument specs for functions
	for syntax, function := range assembly.Functions {
		path := function.Locals.Name.File
		source, found := b.sources[path]
		if !found {
			log.Printf("source not found for %s", path)
			continue
		}

		ident := syntax.Name.Ident.Literal
		id := pythonlocal.LookupID(function)
		if doc := pythonlocal.BuildDocumentation(path, ident, function.Address().Path.String(), syntax.Body); doc != nil {
			docs[id] = *doc
		}
		if def := pythonlocal.BuildDefinition(
			path, syntax, source.Lines); def != nil {
			definitions[id] = *def
		}

		argspec := pythonlocal.ArgSpecFromFunctionDef(source.Contents, syntax, -1)
		argspecs[id] = *argspec
	}
	b.buildDurations["construct_functions"] = time.Since(ts)

	ts = time.Now()
	// Store call specs
	type callSpec struct {
		val   pythontype.Value
		specs []*pythoncode.CallSpec
	}
	calls := make(map[pythontype.FlatID]*callSpec)
	funcParams := make(map[pythontype.FlatID][]string)

	// count values for ranking completions
	for path, file := range assembly.Files {
		if file.ASTBundle.AST == nil {
			continue
		}

		src := b.sources[path]
		if src == nil || src.Contents == nil {
			continue
		}

		pythonast.Inspect(file.ASTBundle.AST, func(node pythonast.Node) bool {
			ctx.CheckAbort()

			if pythonast.IsNil(node) {
				return false
			}

			switch node := node.(type) {
			case *pythonast.FunctionDefStmt:
				// Store the parameter names so we can use them later when computing
				// method patterns
				val, ok := b.delegate.Exprs[node.Name]
				if !ok {
					return true
				}
				h, err := pythontype.Hash(ctx, val)
				if err != nil {
					return true
				}
				funcParams[h] = functionParameters(node, src)

			case *pythonast.AttributeExpr:
				val, ok := b.delegate.Exprs[node]
				if !ok {
					return true
				}

				countValues(ctx, valuesCount, val, node, src)
			case *pythonast.CallExpr:
				val, ok := b.delegate.Exprs[node.Func]
				if !ok {
					return true
				}

				if spec := pythoncode.NewCallSpec(ctx, node, src.Contents, b.delegate.Exprs); spec != nil {
					for _, elem := range pythontype.Disjuncts(ctx, val) {
						hsh, err := pythontype.Hash(ctx, elem)
						if err != nil {
							continue
						}
						if c, exists := calls[hsh]; exists {
							c.specs = append(c.specs, spec)
						} else {
							calls[hsh] = &callSpec{
								val:   elem,
								specs: []*pythoncode.CallSpec{spec},
							}
						}
					}
				}

				// here we only deal with the case where the base
				// is a NameExpr since the case where the base
				// is an AttributeExpr will be handled when we recurse
				// into the AttributeExpr
				_, isName := node.Func.(*pythonast.NameExpr)
				if !isName {
					return true
				}

				countValues(ctx, valuesCount, val, node, src)
			}
			return true
		})
	}
	b.buildDurations["store_call_specs"] = time.Since(ts)

	ts = time.Now()
	// Compute signature patterns
	methods := make(map[string]*pythoncode.MethodPatterns)
	for hsh, cs := range calls {
		val := cs.val
		specs := cs.specs
		if id := pythonlocal.LookupID(val); id != "" {
			pats := pythoncode.MethodPatternsFromCallSpecs(val, specs, funcParams[hsh])
			pythoncode.ProcessPatterns(pats)
			methods[id] = pats
		}
	}
	b.buildDurations["compute_sig_patterns"] = time.Since(ts)

	return &Batch{
		Assembly:    assembly,
		Docs:        docs,
		Definitions: definitions,
		ArgSpecs:    argspecs,
		ValuesCount: valuesCount,
		Methods:     methods,
	}, nil
}

func countValues(ctx kitectx.Context, counts map[string]int, val pythontype.Value, expr pythonast.Expr, src *SourceUnit) {
	ctx.CheckAbort()

	for _, elem := range pythontype.Disjuncts(ctx, val) {
		if elem.Kind() != pythontype.TypeKind && elem.Kind() != pythontype.FunctionKind && elem.Kind() != pythontype.ModuleKind {
			continue
		}

		id := pythonlocal.LookupID(elem)
		counts[id]++
	}
}

func functionParameters(def *pythonast.FunctionDefStmt, src *SourceUnit) []string {
	var params []string
	for _, p := range def.Parameters {
		if int(p.Name.Begin()) >= len(src.Contents) || int(p.Name.End()) > len(src.Contents) {
			rollbar.Debug(fmt.Errorf("function parameter out of bounds, batch.go"),
				fmt.Sprintf("begin: %d, end: %d, len: %d, src: %s", int(p.Name.Begin()), int(p.Name.End()), len(src.Contents), string(src.Contents)))
			continue
		}
		n := string(src.Contents[p.Name.Begin():p.Name.End()])
		// TODO(Daniel): Find a better way to filter out `self` and `cls`
		if n != "self" && n != "cls" {
			params = append(params, n)
		}
	}
	return params
}

type usageDelegate struct {
	Exprs map[pythonast.Expr]pythontype.Value
	count uint
}

func (u *usageDelegate) Pass(cur, total int) {
	if cur == total-1 {
		u.Exprs = make(map[pythonast.Expr]pythontype.Value, u.count)
	}
	u.count = 0
}

func (u *usageDelegate) Resolved(expr pythonast.Expr, value pythontype.Value) {
	u.count++
	if u.Exprs != nil {
		u.Exprs[expr] = value
	}
}
