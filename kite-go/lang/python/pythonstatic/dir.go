package pythonstatic

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Design note: added this here because we really only care about
// this for the `from foo import *` case. The other option is to
// define a `pythontype.Dirable` interface and let each pythontype.Value
// implement their own version, the problem with this approach is then
// we have to define semantics for what does it mean to call `Dir` on
// an arbitrary value.
func dir(ctx kitectx.Context, graph pythonresource.Manager, v pythontype.Value) []string {
	ctx.CheckAbort()

	switch v := v.(type) {
	case pythontype.Union:
		seen := make(map[string]struct{})
		for _, d := range pythontype.Disjuncts(ctx, v) {
			for _, attr := range dir(ctx, graph, d) {
				seen[attr] = struct{}{}
			}
		}

		var attrs []string
		for attr := range seen {
			attrs = append(attrs, attr)
		}

		return attrs
	case pythontype.ExplicitModule:
		var attrs []string
		for attr := range v.Members {
			attrs = append(attrs, attr)
		}
		return attrs
	case *pythontype.SourcePackage:
		if v.Init != nil {
			return dir(ctx, graph, v.Init)
		}
		return nil
	case *pythontype.SourceModule:
		var dict []string
		for name, sym := range v.Members.Table {
			if sym != nil && !sym.Private {
				dict = append(dict, name)
			}
		}
		return dict
	case pythontype.External:
		if v.Kind() != pythontype.ModuleKind {
			return nil
		}

		children, err := graph.Children(v.Symbol())
		if err != nil {
			return nil
		}

		return children
	default:
		return nil
	}
}
