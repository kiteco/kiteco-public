package pythonindex

import (
	"fmt"
	"log"
	"strings"

	"github.com/kennygrant/sanitize"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/text"
)

var (
	minCount = DefaultClientOptions.MinOccurrence
)

// resolve takes "chained" identifier names and resolve them to their fully qualified name. We do this because
// the python parser cannot resolve all identifiers (e.g, functions and other member attributes). So, the parser will
// return identifiers such as `requests.get.json`, which don't actually map to a real identifer, but result from code like:
//
//	x = requests.get("<some url>")
//  print x.json()
//
// This code then maps this identifier to `requests.models.Response.json` via the import graph and type induction.
func resolve(rm pythonresource.Manager, input string) (pythonresource.Symbol, error) {
	parts := strings.Split(input, ".")

	// computes the given child of the symbol and, if the child is a function/descriptor,
	// uses typeinduction to find the return type's symbol
	processPart := func(sym pythonresource.Symbol, part string) []pythonresource.Symbol {
		sym, err := rm.ChildSymbol(sym, part)
		if err != nil {
			return nil
		}
		switch rm.Kind(sym) {
		case keytypes.FunctionKind, keytypes.DescriptorKind:
			if rets := rm.ReturnTypes(sym); len(rets) > 0 {
				return rets
			}
		}
		return []pythonresource.Symbol{sym}
	}

	// there may be multiple matching symbols, so track them all
	var syms []pythonresource.Symbol

	pkg := parts[0]
	pkgPath := pythonimports.NewDottedPath(pkg)
	for _, dist := range rm.DistsForPkg(parts[0]) {
		if sym, err := rm.NewSymbol(dist, pkgPath); err == nil {
			syms = append(syms, sym)
		}
	}

	for _, part := range parts[1:] {
		if len(syms) == 0 {
			break
		}

		var childSyms []pythonresource.Symbol
		for _, sym := range syms {
			childSyms = append(childSyms, processPart(sym, part)...)
		}
		syms = childSyms
	}

	if len(syms) == 0 {
		return pythonresource.Symbol{}, fmt.Errorf("unable to find node for %s", input)
	}
	if len(syms) > 1 {
		// TODO(naman) this should probably happen rarely if at all, but do we want to return multiple symbols from this function?
		log.Printf("[pythonindex/helper.go:resolve] multiple resolutions found for input %s\n", input)
	}
	return syms[0], nil
}

func cleanHTML(htmlStr string) string {
	htmlStr = sanitize.HTML(htmlStr)
	ret := make([]rune, 0, len(htmlStr))
	for _, c := range htmlStr {
		switch c {
		case rune('\u00B6'): // filter our paragraph symbol
		default:
			ret = append(ret, c)
		}
	}
	return string(ret)
}

func indexNode(path string, identifier string, symbolToIdentCounts map[string][]*IdentCount,
	invertedIndex map[string][]*IdentCount) {

	if ncs, found := symbolToIdentCounts[path]; !found {
		nc := &IdentCount{
			Ident:       identifier,
			ForcedCount: minCount,
		}
		parts := text.Uniquify(strings.Split(identifier, "."))
		for _, part := range parts {
			invertedIndex[part] = append(invertedIndex[part], nc)
		}
		symbolToIdentCounts[path] = append(symbolToIdentCounts[path], nc)
	} else {
		for _, nc := range ncs {
			if nc.Ident == identifier && nc.Count < minCount {
				nc.ForcedCount = minCount
			}
		}
		symbolToIdentCounts[path] = ncs
	}
}

func appendNode(path string, ic *IdentCount, symbolToIdentCounts map[string][]*IdentCount) {
	if ncs, found := symbolToIdentCounts[path]; !found {
		symbolToIdentCounts[path] = append(symbolToIdentCounts[path], ic)
	} else {
		for _, nc := range ncs {
			if nc.Ident == ic.Ident && nc.Count < minCount {
				nc.ForcedCount = minCount
			}
		}
		symbolToIdentCounts[path] = ncs
	}
}
