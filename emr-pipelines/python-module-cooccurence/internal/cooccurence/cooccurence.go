package cooccurence

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var parseOpts = pythonparser.Options{
	ErrorMode: pythonparser.FailFast,
}

// ExtractModules extracts the (top level) packages/modules imported in  a file.
func ExtractModules(src []byte) ([]string, error) {
	mod, err := pythonparser.Parse(kitectx.Background(), src, parseOpts)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	pythonast.Inspect(mod, func(node pythonast.Node) bool {
		if pythonast.IsNil(node) {
			return false
		}

		switch node := node.(type) {
		case *pythonast.ImportNameStmt:
			for _, imp := range node.Names {
				seen[imp.External.Names[0].Ident.Literal] = struct{}{}
			}
			return false
		case *pythonast.ImportFromStmt:
			if node.Package != nil {
				seen[node.Package.Names[0].Ident.Literal] = struct{}{}
			}
			return false
		default:
			return true
		}
	})

	var modules []string
	for module := range seen {
		modules = append(modules, module)
	}
	return modules, nil
}

// Cooccurence records co occurence information for a module.
type Cooccurence struct {
	Module     string
	Cooccuring []string
}

// Cooccurences converts a slice of (top level) pacakges/modules that appeared in the same file into a
// slice of `Cooccurence`s.
func Cooccurences(modules []string) []Cooccurence {
	var cooccurs []Cooccurence
	for i, module := range modules {
		cooccur := Cooccurence{
			Module: module,
		}
		for j, module1 := range modules {
			if i == j {
				continue
			}
			cooccur.Cooccuring = append(cooccur.Cooccuring, module1)
		}
		cooccurs = append(cooccurs, cooccur)
	}
	return cooccurs
}
