package pythontype

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type danglingAttrError struct{}

// Error implements error
func (d danglingAttrError) Error() string { return "dangling attribute reference" }

type symAttrResult struct {
	child  pythonresource.Symbol
	parent pythonresource.Symbol
}

// symAttr checks just the symbol's children
func symAttr(g pythonresource.Manager, sym pythonresource.Symbol, name string) (symAttrResult, error) {
	// check the direct children of the symbol
	child, err := g.ChildSymbol(sym, name)
	if err != nil {
		return symAttrResult{}, err
	}
	return symAttrResult{child: child, parent: sym}, nil
}

// symAttrBases checks the symbol along with its base classes
// it is recursive and internally increments the call count on ctx
func symAttrBases(ctx kitectx.CallContext, g pythonresource.Manager, sym pythonresource.Symbol, name string) []symAttrResult {
	if ctx.AtCallLimit() {
		return nil
	}

	if res, err := symAttr(g, sym, name); err == nil {
		return []symAttrResult{res}
	}

	var out []symAttrResult
	for _, baseSym := range g.Bases(sym) {
		// recursive call to symAttrBases increments ctx call count
		out = append(out, symAttrBases(ctx.Call(), g, baseSym, name)...)
	}
	return out
}

// resolveExternalSymAttrs tries to resolve the given attribute on the External's symbol, base classes, and type
// it internally increments the call count on ctx
func resolveExternalSymAttrs(ctx kitectx.CallContext, ext External, name string) []symAttrResult {
	if syms := symAttrBases(ctx, ext.graph, ext.symbol, name); len(syms) > 0 {
		return syms
	}

	if tySym, err := ext.graph.Type(ext.symbol); err == nil {
		return symAttrBases(ctx, ext.graph, tySym, name)
	}

	return nil
}
