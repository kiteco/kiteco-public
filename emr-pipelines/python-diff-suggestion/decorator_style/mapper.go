package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythondiffs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
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

			// For each decorator, we emit its canonical name as the key and a pythondiffs.DecoratorStyle
			// object as the value.
			for _, dec := range fun.Decorators {
				// Got bad decorator name. The decorator name should not be
				// empty if `cls` is the first argument of a function.
				literal := literalName(dec)
				if literal == "" {
					continue
				}
				canonical := literal
				if ref, found := resolved.References[dec]; found {
					if name := canonicalName(ref); name != "" {
						canonical = name
					}
				}

				// Find the root expression of the decorator
				rootExpr := decoratorRootExpr(dec)
				if rootExpr == nil {
					continue
				}

				var root string
				if ref, found := resolved.References[rootExpr]; found {
					root = canonicalName(ref)
				}

				// If we can't find the root canonical name, we can't use it, so give this up.
				if root == "" {
					continue
				}

				dec := pythondiffs.DecoratorStyle{
					Canonical:   canonical,
					Literal:     literal,
					LiteralRoot: root,
				}

				out, err := json.Marshal(dec)
				if err != nil {
					continue
				}
				w.Emit(canonical, out)
			}
		}
	}

	if err := r.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}
}

// decoratorRootExpr returns the
func decoratorRootExpr(expr pythonast.Expr) pythonast.Expr {
	switch expr := expr.(type) {
	case *pythonast.NameExpr:
		return expr
	case *pythonast.AttributeExpr:
		return decoratorRootExpr(expr.Value)
	case *pythonast.CallExpr:
		return decoratorRootExpr(expr.Func)
	}
	return nil
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
