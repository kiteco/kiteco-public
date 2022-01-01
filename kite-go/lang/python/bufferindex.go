package python

import (
	"errors"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
)

var errNotFound = errors.New("node was not found in buffer index")
var errFailed = errors.New("node was found but entity could not be constructed")

// bufferIndex constructs definitions, usages, and documentation from AST
// nodes
type bufferIndex struct {
	tree        *pythonenv.SourceTree
	refs        map[pythonast.Expr]pythontype.Value
	classes     map[string]*pythonast.ClassDefStmt
	funcs       map[string]*pythonast.FunctionDefStmt
	valuesCount map[string]int
	filepath    string
	buf         []byte
	hash        string // localfiles.ComputeHash of buf
	lines       *linenumber.Map
}

// newBufferIndex builds a bufferIndex from a resolved AST
func newBufferIndex(ctx kitectx.Context, resolved *pythonanalyzer.ResolvedAST, buf []byte, filepath string) *bufferIndex {
	ctx.CheckAbort()

	classes := make(map[string]*pythonast.ClassDefStmt)
	funcs := make(map[string]*pythonast.FunctionDefStmt)
	valuesCount := make(map[string]int)

	pythonast.Inspect(resolved.Root, func(node pythonast.Node) bool {
		ctx.CheckAbort()

		if node == nil {
			return false
		}
		switch node := node.(type) {
		case *pythonast.ClassDefStmt:
			ref := resolved.References[node.Name]
			if ref == nil {
				break
			}

			id := pythonlocal.LookupID(ref)
			if id != "" {
				classes[id] = node
			}
		case *pythonast.FunctionDefStmt:
			ref := resolved.References[node.Name]
			if ref == nil {
				break
			}

			id := pythonlocal.LookupID(ref)
			if id != "" {
				funcs[id] = node
			}
		case pythonast.Expr:
			ref := resolved.References[node]
			if ref == nil {
				break
			}

			id := pythonlocal.LookupID(ref)
			if id == "" {
				break
			}

			switch ref.Kind() {
			case pythontype.TypeKind, pythontype.FunctionKind, pythontype.ModuleKind:
				valuesCount[id]++
			}
		}
		return true
	})

	tree := pythonenv.NewSourceTree()
	tree.AddFile(filepath, resolved.Module, strings.HasPrefix(filepath, "/windows/"))

	return &bufferIndex{
		tree:        tree,
		refs:        resolved.References,
		classes:     classes,
		funcs:       funcs,
		valuesCount: valuesCount,
		filepath:    filepath,
		buf:         buf,
		hash:        localfiles.ComputeHash(buf),
		lines:       linenumber.NewMap(buf),
	}
}

// documentation extracts documentation for the given node from the current file.
func (b *bufferIndex) Documentation(v pythontype.Value) (*pythonlocal.Documentation, error) {
	id := pythonlocal.LookupID(v)
	if v == nil || id == "" {
		return nil, errNotFound
	}

	switch v.Kind() {
	case pythontype.FunctionKind:
		stmt, found := b.funcs[id]
		if !found {
			return nil, errNotFound
		}

		doc := pythonlocal.BuildDocumentation(b.filepath, stmt.Name.Ident.Literal, "", stmt.Body)
		if doc == nil {
			return doc, errFailed
		}
		return doc, nil

	case pythontype.TypeKind:
		stmt, found := b.classes[id]
		if !found {
			return nil, errNotFound
		}

		doc := pythonlocal.BuildDocumentation(b.filepath, stmt.Name.Ident.Literal, "", stmt.Body)
		if doc == nil {
			return nil, errFailed
		}
		return doc, nil
	}

	return nil, errNotFound
}

func (b *bufferIndex) FindValue(ctx kitectx.Context, file string, path []string) (pythontype.Value, error) {
	if file == b.filepath {
		return b.tree.FindValue(ctx, file, path)
	}
	return nil, errNotFound
}

func (b *bufferIndex) FindSymbol(ctx kitectx.Context, file string, path []string, attr string) (pythontype.Value, pythontype.Value, error) {
	if file == b.filepath {
		return b.tree.FindSymbol(ctx, file, path, attr)
	}
	return nil, nil, errNotFound
}

// definition gets the definition of the given value
func (b *bufferIndex) Definition(v pythontype.Value) (*pythonlocal.Definition, error) {
	id := pythonlocal.LookupID(v)
	if v == nil || id == "" {
		return nil, errNotFound
	}

	switch v.Kind() {
	case pythontype.FunctionKind:
		stmt, found := b.funcs[id]
		if !found {
			return nil, errNotFound
		}

		def := pythonlocal.BuildDefinition(b.filepath, stmt, b.lines)
		if def == nil {
			return nil, errFailed
		}
		return def, nil

	case pythontype.TypeKind:
		stmt, found := b.classes[id]
		if !found {
			return nil, errNotFound
		}

		def := pythonlocal.BuildDefinition(b.filepath, stmt, b.lines)
		if def == nil {
			return nil, errFailed
		}
		return def, nil
	}

	return nil, errNotFound
}

// ValueCount gets the count of a value in this index
func (b *bufferIndex) ValueCount(val pythontype.Value) (int, error) {
	count, found := b.valuesCount[pythonlocal.LookupID(val)]
	if !found {
		return 0, errNotFound
	}
	return count, nil
}

// ArgSpec gets the arg spec for the given node
// TODO(naman) unused; remove or re-enable?
func (b *bufferIndex) ArgSpec(ctx kitectx.Context, v pythontype.Value) (*pythonimports.ArgSpec, error) {
	ctx.CheckAbort()
	id := pythonlocal.LookupID(v)
	if v == nil || id == "" {
		return nil, errNotFound
	}

	switch v.Kind() {
	case pythontype.FunctionKind:
		stmt, found := b.funcs[id]
		if !found {
			return nil, errNotFound
		}

		argspec := pythonlocal.ArgSpecFromFunctionDef(b.buf, stmt, 0)
		if argspec == nil {
			return nil, errFailed
		}
		return argspec, nil

	case pythontype.TypeKind:
		// look for an explicit constructor
		attr, _ := pythontype.Attr(ctx, v, "__init__")
		if val := attr.Value(); val != nil && val.Kind() == pythontype.FunctionKind {
			if as, err := b.ArgSpec(ctx, val); err == nil {
				return as, nil
			}
		}

		// fall back to argspec with "self" only (since all classes have default constructors)
		return &pythonimports.ArgSpec{
			Args: []pythonimports.Arg{pythonimports.Arg{Name: "self"}},
		}, nil
	}

	return nil, errNotFound
}
