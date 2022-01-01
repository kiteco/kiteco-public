package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythondiffs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/text"
)

func main() {
	r := awsutil.NewEMRIterator(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	// Load import graph
	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		log.Fatalln(err)
	}

	// Load type induction client
	typeInducer, err := typeinduction.NewClientFromOptions(typeinduction.DefaultClientOptions)
	if err != nil {
		log.Fatalln(err)
	}

	for r.Next() {
		var code string
		err := json.Unmarshal(r.Value(), &code)
		if err != nil {
			log.Fatal("error unmarshaling json:", err)
		}

		var parseopts pythonparser.Options
		parseopts.ErrorMode = pythonparser.FailFast
		mod, err := pythonparser.Parse(kitectx.Background(), []byte(code), parseopts)
		if err != nil {
			continue
		}

		r := pythonanalyzer.NewResolver(graph, typeInducer, pythonanalyzer.Options{})
		resolved, err := r.Resolve(mod)
		if err != nil {
			continue
		}

		// Get the packages that are imported
		imported := getImportedPackages(resolved)

		// Go through function def stmts and see if there are any decorators.
		for fun := range resolved.Functions {
			if len(fun.Parameters) == 0 {
				continue
			}

			var firstParam string
			if name, ok := fun.Parameters[0].Name.(*pythonast.NameExpr); ok {
				firstParam = name.Ident.Literal
			}

			// We only look at functions whose first parameter is "cls"
			if firstParam != "cls" {
				continue
			}

			fd := pythondiffs.FuncDecorator{
				Func: fun.Name.Ident.Literal,
			}

			if len(fun.Decorators) == 0 {
				// If there are no decorators, we also want to count it, so
				// emit a FuncDecorator with Decorator being empty.
				out, err := json.Marshal(fd)
				if err != nil {
					continue
				}
				for _, im := range imported {
					w.Emit(im, out)
				}
			}

			for _, dec := range fun.Decorators {
				var canonical string
				if ref, found := resolved.References[dec]; found {
					canonical = canonicalName(ref)
				}
				// Got bad decorator name. The decorator name should not be
				// empty if `cls` is the first argument of a function.
				if canonical == "" {
					continue
				}
				fd.Decorator = canonical
				out, err := json.Marshal(fd)
				if err != nil {
					continue
				}
				for _, im := range imported {
					w.Emit(im, out)
				}
			}
		}
	}

	if err := r.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}
}

// temporary backwards-compatibility hack to get a canonical name from a reference until
// we switch everything over to use *pythonimports.Node
func canonicalName(ref *pythonanalyzer.Reference) string {
	if ref.Path != "" {
		return ref.Path
	}
	if ref.Node != nil {
		if ref.Node.CanonicalName != "" {
			return ref.Node.CanonicalName
		}
		if ref.Node.Type != nil {
			return ref.Node.Type.CanonicalName
		}
	}
	return ""
}

// literalName returns the literal string of the expression.
func literalName(expr pythonast.Expr) string {
	switch expr := expr.(type) {
	case *pythonast.NameExpr:
		return expr.Ident.Literal
	case *pythonast.AttributeExpr:
		return literalName(expr.Value) + "." + expr.Attribute.Literal
	case *pythonast.CallExpr:
		return literalName(expr.Func)
	}
	return ""
}

func getImportedPackages(ast *pythonanalyzer.ResolvedAST) []string {
	var importedPackages []string
	for _, imports := range ast.ImportStmts {
		for _, im := range imports {
			if im.External == nil {
				continue
			}
			if tokens := strings.Split(im.External.Path, "."); len(tokens) > 0 && tokens[0] != "" {
				importedPackages = append(importedPackages, tokens[0])
			}
		}
	}
	return text.Uniquify(importedPackages)
}
