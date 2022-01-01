package pythonanalyzer

import (
	"fmt"
	"io"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const numPasses = 2

// Options for type resolution
type Options struct {
	// User and Machine describe the user and machine for which analysis is done
	User    int64
	Machine string
	// Path is the absolute path to the file that is being resolved.
	// NOTE: must be an absolute path and it must include the .py extension.
	Path string
	// Trace is a writer to which diagnostics will be written
	Trace io.Writer
}

// ResolvedAST represents a mapping from expression nodes to their types
type ResolvedAST struct {
	// Root is the top of the syntax tree
	Root *pythonast.Module
	// References is a map from expressions to their resolved values
	References map[pythonast.Expr]pythontype.Value
	// Parent is a map from a node to its parent
	Parent map[pythonast.Node]pythonast.Node

	// ParentStmts is a map from an expression to the deepest statement that contains the expression.
	ParentStmts map[pythonast.Expr]pythonast.Stmt

	// ModuleValue is the value that the module was resolved to.
	Module *pythontype.SourceModule

	// Order that ast expressions were evaluated in
	Order map[pythonast.Expr]int

	// scopes is a map from an expression to the scope in which we should begin resolving it.
	scopes map[pythonast.Expr]pythonast.Scope

	// tables are the symbol tables associated with all of the lexical scopes in the module.
	tables map[pythonast.Scope]*pythontype.SymbolTable
}

// DeepCopy deep-copies the underlying AST and all of the ResolvedAST maps (updated to reference the copied AST nodes)
// NOTE: the symbol tables and Module value are not copied.
// Returns the copy along with a map from ast nodes in the original to ast nodes in the copy.
func (r *ResolvedAST) DeepCopy() (*ResolvedAST, map[pythonast.Node]pythonast.Node) {
	copies := pythonast.DeepCopy(r.Root)

	references := make(map[pythonast.Expr]pythontype.Value, len(r.References))
	for k, v := range r.References {
		references[copies[k].(pythonast.Expr)] = v
	}

	parent := make(map[pythonast.Node]pythonast.Node, len(r.Parent))
	for c, p := range r.Parent {
		parent[copies[c]] = copies[p]
	}

	parentStmts := make(map[pythonast.Expr]pythonast.Stmt, len(r.ParentStmts))
	for c, p := range r.ParentStmts {
		parentStmts[copies[c].(pythonast.Expr)] = copies[p].(pythonast.Stmt)
	}

	order := make(map[pythonast.Expr]int, len(r.Order))
	for n, i := range r.Order {
		order[copies[n].(pythonast.Expr)] = i
	}

	scopes := make(map[pythonast.Expr]pythonast.Scope, len(r.scopes))
	for n, s := range r.scopes {
		scopes[copies[n].(pythonast.Expr)] = copies[s].(pythonast.Scope)
	}

	tables := make(map[pythonast.Scope]*pythontype.SymbolTable, len(r.tables))
	for s, t := range r.tables {
		tables[copies[s].(pythonast.Scope)] = t
	}

	return &ResolvedAST{
		Root:        copies[r.Root].(*pythonast.Module),
		References:  references,
		Parent:      parent,
		ParentStmts: parentStmts,
		Module:      r.Module,
		Order:       order,
		scopes:      scopes,
		tables:      tables,
	}, copies
}

// UpdateScope updates the given NameExpr's scope; TODO(naman) we should instead implement a `Replace` abstraction that takes AST nodes
func (r *ResolvedAST) UpdateScope(expr pythonast.Expr, scope pythonast.Scope) {
	if scope == nil {
		delete(r.scopes, expr)
	} else {
		r.scopes[expr] = scope
	}
}

// TableFor scope
func (r *ResolvedAST) TableFor(scope pythonast.Scope) *pythontype.SymbolTable {
	return r.tables[scope]
}

// TableAndScope in which to begin resolving the provided expression.
func (r *ResolvedAST) TableAndScope(expr pythonast.Expr) (*pythontype.SymbolTable, pythonast.Scope) {
	scope := r.scopes[expr]
	return r.tables[scope], scope
}

// RefinedValue tries to return a "refined" version of the value for the specified expr,
// if no refined value is found then `r.References[expr]` is returned.
// SEE: pythonstatic/refine.go and pythonstatic/assembly.go for further details on the refinement process.
func (r *ResolvedAST) RefinedValue(expr pythonast.Expr) pythontype.Value {
	if name, ok := expr.(*pythonast.NameExpr); ok {
		if table, _ := r.TableAndScope(name); table != nil {
			if symbol := table.Find(name.Ident.Literal); symbol != nil && symbol.Value != nil {
				return symbol.Value
			}
		}
	}
	return r.References[expr]
}

// ValueForScope returns the value associated with the specified scope.
func (r *ResolvedAST) ValueForScope(scope pythonast.Scope) pythontype.Value {
	switch scope := scope.(type) {
	case *pythonast.Module:
		return r.Module
	case *pythonast.ClassDefStmt:
		return r.References[scope.Name]
	case *pythonast.FunctionDefStmt:
		return r.References[scope.Name]
	case *pythonast.LambdaExpr:
		return r.References[scope]
	default:
		return nil
	}
}

// A Resolver transforms syntax trees to resolutions
type Resolver struct {
	importer pythonstatic.Importer
	shadow   *pythontype.SourceModule
	opts     Options
}

// NewResolver construct a resolver that uses the given graph, typeinducer, and options
func NewResolver(graph pythonresource.Manager, opts Options) *Resolver {
	importer := pythonstatic.Importer{
		Path:   opts.Path,
		Global: graph,
	}
	return NewResolverUsingImporter(importer, opts)
}

// NewResolverUsingImporter construct a resolver that uses the given environment, typeinducer, and options
func NewResolverUsingImporter(importer pythonstatic.Importer, opts Options) *Resolver {
	return &Resolver{
		importer: importer,
		opts:     opts,
	}
}

// Resolve computes the types for expressions in a syntax tree
func (r *Resolver) Resolve(module *pythonast.Module) (*ResolvedAST, error) {
	return r.ResolveContext(kitectx.Background(), module, false)
}

// ResolveContext computes the types for expressions in a syntax tree with a context
func (r *Resolver) ResolveContext(ctx kitectx.Context, module *pythonast.Module, allowValueMutation bool) (rast *ResolvedAST, err error) {
	ctx.CheckAbort()

	// eval may introduce new nodes, which will cause issues with DeepCopy, so we check everywhere to refer only to nodes in the AST
	nodeSet := make(map[pythonast.Node]struct{})
	pythonast.Inspect(module, func(n pythonast.Node) bool {
		nodeSet[n] = struct{}{}
		return true
	})
	opts := pythonstatic.DefaultOptions
	opts.AllowValueMutation = allowValueMutation
	opts.UseCapabilities = true
	delegate := &collector{nodeSet: nodeSet}
	assembler := pythonstatic.NewAssembler(ctx, pythonstatic.AssemblerInputs{
		User:     r.opts.User,
		Machine:  r.opts.Machine,
		Graph:    r.importer.Global,
		Importer: &r.importer,
		Delegate: delegate,
	}, opts)

	assembler.AddSource(pythonstatic.ASTBundle{
		AST:     module,
		Path:    r.opts.Path,
		Imports: pythonstatic.FindImports(ctx, r.opts.Path, module),
	})

	asm, err := assembler.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("error analyzing file: %v", err)
	}

	// count the number of nodes in the AST
	mod := asm.Modules[module]
	nodeCount := pythonast.CountNodes(module)

	// create symbol table for the module

	tables := make(map[pythonast.Scope]*pythontype.SymbolTable)
	tables[module] = mod.Members

	addToTable := func(s pythonast.Scope, t *pythontype.SymbolTable) {
		if _, ok := nodeSet[s]; !ok {
			return
		}
		tables[s] = t
	}

	for stmt, val := range asm.Functions {
		addToTable(stmt, val.Locals)
	}

	for stmt, val := range asm.Classes {
		addToTable(stmt, val.Members)
	}

	for stmt, val := range asm.Lambdas {
		addToTable(stmt, val.Locals)
	}

	for comp, table := range asm.Comprehensions {
		addToTable(comp, table)
	}

	return &ResolvedAST{
		Root:        module,
		Parent:      pythonast.ConstructParentTable(module, nodeCount),
		ParentStmts: pythonast.ConstructStmtTable(module, nodeCount),
		References:  delegate.Exprs,
		Order:       delegate.Order,
		Module:      mod,
		tables:      tables,
		scopes:      pythonast.ConstructScopeTable(module),
	}, nil
}

// collector implements PropagatorDelegate; it receives callbacks from the propagator
// and builds up the list of references and missing expressions
type collector struct {
	Exprs map[pythonast.Expr]pythontype.Value
	Order map[pythonast.Expr]int
	count int

	nodeSet map[pythonast.Node]struct{}
}

func (c *collector) Pass(cur, total int) {
	if cur == total-1 {
		c.Exprs = make(map[pythonast.Expr]pythontype.Value, c.count)
		c.Order = make(map[pythonast.Expr]int, c.count)
	}
	c.count = 0
}

// Resolved implements pythonstatic.PropagatorDelegate
func (c *collector) Resolved(expr pythonast.Expr, val pythontype.Value) {
	if pythonast.IsNil(expr) {
		panic(fmt.Sprintf("expr was nil for val=%v", val))
	}

	// when propagating eval expressions, new Nodes may be created and passed here;
	// don't put them in the Exprs map because that causes issues (see DeepCopy above)
	if _, ok := c.nodeSet[expr]; !ok {
		return
	}

	c.count++
	if c.Exprs != nil {
		c.Exprs[expr] = val
		if _, seen := c.Order[expr]; !seen {
			c.Order[expr] = len(c.Order)
		}
	}
}
