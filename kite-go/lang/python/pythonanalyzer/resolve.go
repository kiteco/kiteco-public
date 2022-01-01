package pythonanalyzer

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Models to resolve an ast against.
type Models struct {
	Importer pythonstatic.Importer
	// Shadow module from offline analysis, may be nil.
	Shadow *pythontype.SourceModule
}

// Resolve expressions in the specified module.
func Resolve(ctx kitectx.Context, m Models, ast *pythonast.Module, opts Options) (*ResolvedAST, error) {
	ctx.CheckAbort()

	res := NewResolverUsingImporter(m.Importer, opts)

	return res.ResolveContext(ctx, ast, false)
}
