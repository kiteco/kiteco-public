package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
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
	typeInducer, err := typeinduction.NewClientFromPaths(typeinduction.DefaultClientPaths)
	if err != nil {
		log.Fatalln(err)
	}

	builtin := graph.PkgToNode["__builtin__"]
	if builtin == nil {
		log.Fatal("cannot find builtin in graph")
	}

	for r.Next() {
		var snippet pythoncode.Snippet
		err := json.Unmarshal(r.Value(), &snippet)
		if err != nil {
			log.Fatal(err)
		}

		var parseopts pythonparser.Options
		parseopts.ErrorMode = pythonparser.FailFast
		mod, err := pythonparser.Parse(kitectx.Background(), []byte(snippet.Code), parseopts)
		if err != nil {
			continue
		}

		r := pythonanalyzer.NewResolver(graph, typeInducer, pythonanalyzer.Options{})
		resolved, err := r.Resolve(mod)
		if err != nil {
			continue
		}

		parentTable := constructParentTable(mod)
		for node, context := range parentTable {
			expr, isExpr := node.(pythonast.Expr)
			if !isExpr {
				continue
			}
			ref, found := resolved.References[expr]
			if !found {
				continue
			}

			if builtin.HasMember(ref.Rvalue.Node) {
				key := ref.Rvalue.Node.CanonicalName
				value := pythonanalyzer.ContextName(context, node)
				if value == "pythonast.CallExpr.Func" {
					key += "-" + value
					value = pythonanalyzer.ContextName(parentTable[context], context)
				}
				w.Emit(key, []byte(value))
			}
		}
	}

	if err := r.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}
}

func constructParentTable(node pythonast.Node) map[pythonast.Node]pythonast.Node {
	parents := make(map[pythonast.Node]pythonast.Node)
	var curParent pythonast.Node
	pythonast.Inspect(node, func(n pythonast.Node) bool {
		if n == nil {
			curParent = parents[curParent]
			return false
		}
		parents[n] = curParent
		curParent = n
		return true
	})
	return parents
}
