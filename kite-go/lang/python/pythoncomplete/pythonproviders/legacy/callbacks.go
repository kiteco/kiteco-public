package legacy

import (
	"sort"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// Edge describes a connection between parent and child values
type Edge struct {
	Child pythontype.Value
	Edge  string
}

// Callbacks defines a set of functions that the completions engine can call in order to produce completions.
type Callbacks interface {
	licensing.ProductGetter

	// PkgMembers should return a slice of Edges representing the members of the symbol indicated by node, along with the typed prefix
	PkgMembers(ctx kitectx.Context, pkg *pythonast.DottedExpr, numDots int) (map[string]pythontype.Value, string)

	// ScoreImportEdge should return a score for an import completion
	ScoreImportEdge(ctx kitectx.Context, val pythontype.Value) float64

	// Subpackages should return a slice of Edges representing possible import subpackage completions, along with the typed prefix
	Subpackages(ctx kitectx.Context, pkg *pythonast.DottedExpr, fromDots int, name *pythonast.NameExpr) (map[string]pythontype.Value, string)

	// ImportAliases should, given the NameExpr resolving to a module and a NameExpr representing a
	// partially typed alias, return an Edge representing the package resolved from the path, as well as
	// a map from each possible alias of that symbol to the fraction of all imports of that symbol that the
	// alias represents.
	ImportAliases(ctx kitectx.Context, module, alias *pythonast.NameExpr) (Edge, map[string]float64)
	// ImportAliasesForValue skips the resolved AST lookup
	ImportAliasesForValue(ctx kitectx.Context, val pythontype.Value) map[string]float64
}

// CompletionsCallbacks implements Callbacks
type CompletionsCallbacks struct {
	Buffer []byte
	Cursor int64

	Words      []pythonscanner.Word
	Resolved   *pythonanalyzer.ResolvedAST
	LocalIndex *pythonlocal.SymbolIndex

	Importer pythonstatic.Importer

	Models *pythonmodels.Models

	IDs userids.IDs

	licensing.ProductGetter
}

// PkgMembers implements Callbacks
func (c CompletionsCallbacks) PkgMembers(ctx kitectx.Context, pkg *pythonast.DottedExpr, numDots int) (map[string]pythontype.Value, string) {
	ctx.CheckAbort()

	var typedPrefix string
	var numNames int
	if pkg != nil {
		for _, name := range pkg.Names {
			if c.Cursor < int64(name.Begin()) {
				break
			}
			numNames++
		}
		if numNames > 0 {
			name := pkg.Names[numNames-1]
			lenTypedPrefix := int(c.Cursor - int64(name.Begin()))
			if lenTypedPrefix < 0 || len(name.Ident.Literal) < lenTypedPrefix {
				rollbar.Error(errors.Errorf("cursor out of bounds of name in dotted expr"))
				return nil, ""
			}
			typedPrefix = name.Ident.Literal[:lenTypedPrefix]
		}
	}

	var parent pythontype.Value

	if numNames > 1 {
		parent = c.Resolved.References[pkg.Names[numNames-2]]
	} else if numDots == 0 {
		parent = pythontype.ExternalRoot{Graph: c.Importer.Global}
	} else if c.Importer.Local != nil {
		parent = c.Importer.Local.ImportRel(c.Importer.Path, numDots)
	}

	members := pythontype.Members(ctx, c.Importer.Global, parent)
	for attr, val := range members {
		var modules []pythontype.Value
		for _, v := range pythontype.Disjuncts(ctx, val) {
			if v.Kind() == pythontype.ModuleKind {
				modules = append(modules, v)
			}
		}
		if len(modules) == 0 {
			delete(members, attr)
			continue
		}
		members[attr] = pythontype.Unite(ctx, modules...)
	}

	// Get local top-level imports as well
	if numDots == 0 && numNames <= 1 {
		members = c.localImportEdges(ctx, members)
	}

	return members, typedPrefix
}

// ScoreImportEdge implements Callbacks
func (c CompletionsCallbacks) ScoreImportEdge(ctx kitectx.Context, val pythontype.Value) float64 {
	return float64(scoreValueCompletion(ctx, val, c.Importer.Global, c.LocalIndex, func(counts symbolcounts.Counts) int { return counts.Import }))
}

// Subpackages implements Callbacks
func (c CompletionsCallbacks) Subpackages(ctx kitectx.Context, pkg *pythonast.DottedExpr, fromDots int, name *pythonast.NameExpr) (map[string]pythontype.Value, string) {
	ctx.CheckAbort()

	var val pythontype.Value

	if pkg == nil {
		// If there's no package, we think it may be a relative import of the form: from . import foo
		if fromDots == 0 {
			return nil, ""
		}
		val, _ = c.Importer.ImportRel(fromDots)
	} else {
		ref := c.Resolved.References[pkg]
		if ref == nil {
			return nil, ""
		}
		val = ref
	}

	var typedPrefix string
	if name != nil {
		lenTypedPrefix := int(c.Cursor - int64(name.Begin()))
		if lenTypedPrefix < 0 || len(name.Ident.Literal) < lenTypedPrefix {
			rollbar.Error(errors.Errorf("cursor out of bounds of name in dotted expr"))
			return nil, ""
		}
		typedPrefix = name.Ident.Literal[:lenTypedPrefix]
	}

	return pythontype.Members(ctx, c.Importer.Global, val), typedPrefix
}

// ImportAliases implements Callbacks
func (c CompletionsCallbacks) ImportAliases(ctx kitectx.Context, module, alias *pythonast.NameExpr) (Edge, map[string]float64) {
	ctx.CheckAbort()

	val := c.Resolved.References[module]
	sym := getSingleGlobalSymbolFromValue(ctx, val)
	if sym.Nil() {
		return Edge{}, nil
	}
	fractions := c.importAliasesForSymbol(sym, alias)

	symPath := sym.Path()

	edge := Edge{
		Child: val,
		Edge:  symPath.Last(),
	}

	return edge, fractions
}

// ImportAliasesForValue implements Callbacks
func (c CompletionsCallbacks) ImportAliasesForValue(ctx kitectx.Context, val pythontype.Value) map[string]float64 {
	sym := getSingleGlobalSymbolFromValue(ctx, val)
	if sym.Nil() {
		return nil
	}
	return c.importAliasesForSymbol(sym, nil)
}

// local "absolute" importable edges (i.e. with no dots)
func (c CompletionsCallbacks) localImportEdges(ctx kitectx.Context, out map[string]pythontype.Value) map[string]pythontype.Value {
	ctx.CheckAbort()

	if c.Importer.Local == nil {
		return out
	}

	if out == nil {
		out = make(map[string]pythontype.Value)
	}

	// note that currently, ListAbs handles the "implicit relative import" of Py2, since the heuristic walks every ancestor of the srcpath
	// however, if we made that behavior more precise, we might consider using ImportRel here to additionally add the relatively importable modules.
	absImports := c.Importer.Local.ListAbs(c.Importer.Path)
	names := make([]string, 0, len(absImports))
	for k := range absImports {
		names = append(names, k)
	}
	sort.Strings(names)

	for _, name := range names {
		sym := absImports[name]
		srcVal, ok := sym.Value.(pythontype.SourceValue)
		if !ok {
			rollbar.Error(errors.New("non-SourceValue returned by SourceTree.ListAbs"))
			continue
		}

		edgeName := valueName(srcVal)
		if edgeName == "" {
			edgeName = name
		}

		out[edgeName] = sym.Value
	}

	return out
}

func getSingleGlobalSymbolFromValue(ctx kitectx.Context, val pythontype.Value) pythonresource.Symbol {
	// find the least symbol in the disjuncts, matching the PathSymbol behavior (called by ImportAliases below)
	var leastSym pythonresource.Symbol
	for _, val := range pythontype.Disjuncts(ctx, val) {
		if val, ok := val.(pythontype.External); ok {
			sym := val.Symbol()
			if leastSym.Nil() || sym.Less(leastSym) {
				leastSym = sym
			}
		}
	}
	return leastSym
}
func (c CompletionsCallbacks) importAliasesForSymbol(sym pythonresource.Symbol, alias *pythonast.NameExpr) map[string]float64 {
	counts := c.Importer.Global.SymbolCounts(sym)
	if counts == nil {
		return nil
	}

	fractions := make(map[string]float64)
	for alias, count := range counts.ImportAliases {
		fractions[alias] = float64(count) / float64(counts.ImportThis)
	}
	return fractions
}
func scoreValueCompletion(ctx kitectx.Context, child pythontype.Value, rm pythonresource.Manager, idx *pythonlocal.SymbolIndex, scorer symbolcounts.Scorer) int {
	ctx.CheckAbort()

	if child == nil {
		return 0
	}

	choice := pythontype.MostSpecific(ctx, child)
	// First see if we are looking at global values, and use global scores
	switch choice.(type) {
	case pythontype.ExternalRoot:
		return scoreExternal(ctx, choice, rm, scorer)
	case pythontype.External:
		return scoreExternal(ctx, choice, rm, scorer)
	}

	// Second try ranking using the entire local index
	if idx != nil {
		if score, err := idx.ValueCount(ctx, child); err == nil {
			return score
		}
	}

	return 0
}
func scoreExternal(ctx kitectx.Context, val pythontype.Value, rm pythonresource.Manager, scorer symbolcounts.Scorer) int {
	ctx.CheckAbort()

	path := pythoncode.ValuePath(ctx, val, rm)
	if path.Empty() {
		return 0
	}

	sym, err := rm.PathSymbol(path)
	if err != nil {
		return 0
	}

	counts := rm.SymbolCounts(sym)
	if counts == nil {
		return 0
	}

	return scorer(*counts)
}
