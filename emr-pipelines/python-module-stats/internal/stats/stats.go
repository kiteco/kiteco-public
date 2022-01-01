package stats

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	parseOpts = pythonparser.Options{
		ScanOptions: pythonscanner.Options{
			ScanComments: false,
			ScanNewLines: false,
		},
		ErrorMode: pythonparser.Recover,
	}

	resolveOpts = pythonanalyzer.Options{}
)

// Params are required to run `Extract`
type Params struct {
	Client  *typeinduction.Client
	Manager pythonresource.Manager
}

// PathCounts describes a set of counts of a symbol, keyed by its path.
type PathCounts map[string]*symbolcounts.Counts

func (p PathCounts) get(path string) *symbolcounts.Counts {
	if counts := p[path]; counts != nil {
		return counts
	}

	counts := symbolcounts.NewCounts()
	p[path] = &counts
	return &counts
}

func addImport(c *symbolcounts.Counts, alias string) {
	c.Import++
	c.ImportThis++
	if alias != "" {
		c.ImportAliases[alias]++
	}
}

// Extract the stats for a python file. The stats are keyed by a potential path to a symbol; a given key
// may or may not represent a valid path.
func Extract(params Params, src []byte) (PathCounts, error) {
	// parse
	mod, err := pythonparser.Parse(kitectx.Background(), src, parseOpts)
	if mod == nil {
		return nil, err
	}

	// resolve
	resolver := pythonanalyzer.NewResolver(params.Manager, params.Client, resolveOpts)

	resolved, err := resolver.Resolve(mod)
	if err != nil {
		return nil, err
	}

	// count stats
	c := make(PathCounts)

	pythonast.InspectEdges(mod, func(parent, child pythonast.Node, field string) bool {
		if pythonast.IsNil(child) || pythonast.IsNil(parent) {
			return true
		}

		if _, isBadStmt := parent.(*pythonast.BadStmt); isBadStmt {
			return false
		}

		if _, isAssign := parent.(*pythonast.AssignStmt); isAssign && field == "Targets" {
			// do not recurse into LHS of assignment statements to avoid double counting
			return false
		}

		switch child := child.(type) {
		case *pythonast.ImportFromStmt:
			// ignore relative imports
			if child.Package != nil && len(child.Dots) == 0 {
				base := child.Package.Join()
				for _, clause := range child.Names {
					name := base + "." + clause.External.Ident.Literal
					var alias string
					if clause.Internal != nil {
						alias = clause.Internal.Ident.Literal
					}
					addImport(c.get(name), alias)
				}
			}
			return false

		case *pythonast.ImportNameStmt:
			for _, clause := range child.Names {
				var alias string
				if clause.Internal != nil {
					alias = clause.Internal.Ident.Literal
				}
				addImport(c.get(clause.External.Join()), alias)
			}
			return false

		case *pythonast.AttributeExpr:
			if ref := resolved.References[child]; ref != nil && ref.Value != nil {
				if path := pythoncode.ValuePath(ref.Value, params.Manager); !path.Empty() {
					c.get(path.String()).Attribute++
					return false
				}
			}
			walked := walkUnresolvedAttr(resolved, params.Manager, child)
			if !walked.Empty() {
				c.get(walked.String()).Attribute++
				return false
			}
			return true

		case *pythonast.NameExpr:
			if ref := resolved.References[child]; ref != nil && ref.Value != nil {
				if path := pythoncode.ValuePath(ref.Value, params.Manager); !path.Empty() {
					c.get(path.String()).Name++
				}
			}
			return false

		case pythonast.Expr:
			if ref := resolved.References[child]; ref != nil && ref.Value != nil {
				if path := pythoncode.ValuePath(ref.Value, params.Manager); !path.Empty() {
					c.get(path.String()).Expr++
				}
			}
			return true

		default:
			return true
		}
	})

	return c, nil
}

func walkUnresolvedAttr(resolved *pythonanalyzer.ResolvedAST, rm pythonresource.Manager, attr *pythonast.AttributeExpr) pythonimports.DottedPath {
	// check if base was resolved and the resolved value found in the symbol graph
	if ref := resolved.References[attr.Value]; ref != nil && ref.Value != nil {
		if path := pythoncode.ValuePath(ref.Value, rm); !path.Empty() {
			return path.WithTail(attr.Attribute.Literal)
		}
		return pythonimports.DottedPath{}
	}

	// base was not resolved
	switch base := attr.Value.(type) {
	case *pythonast.NameExpr:
		return pythonimports.NewPath(base.Ident.Literal, attr.Attribute.Literal)
	case *pythonast.AttributeExpr:
		bases := walkUnresolvedAttr(resolved, rm, base)
		return bases.WithTail(attr.Attribute.Literal)
	default:
		return pythonimports.DottedPath{}
	}
}
