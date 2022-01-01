package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

func main() {
	r := awsutil.NewEMRIterator(os.Stdin)
	w := awsutil.NewEMRWriter(os.Stdout)
	defer w.Close()

	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		log.Fatalln("could not load import graph:", err)
	}

	inducer, err := typeinduction.NewClientFromPaths(typeinduction.DefaultClientPaths)
	if err != nil {
		log.Fatalln("could not load type inducer:", err)
	}

	var attributes []string
	var incantations []*pythoncode.Incantation

	for r.Next() {
		var snippet pythoncode.Snippet
		err := json.Unmarshal(r.Value(), &snippet)
		if err != nil {
			log.Fatal(err)
		}

		for _, inc := range snippet.Incantations {
			if resolved, err := resolveCallable(inc.ExampleOf, graph, inducer); err == nil {
				inc.ExampleOf = resolved
				for _, arg := range inc.Args {
					if arg.Type == "" {
						continue
					}
					arg.Type, _ = resolveArg(arg.Type, graph, inducer)
				}
				for _, kwarg := range inc.Kwargs {
					if kwarg.Type == "" {
						continue
					}
					kwarg.Type, _ = resolveArg(kwarg.Type, graph, inducer)
				}
				incantations = append(incantations, inc)
			}
		}
		snippet.Incantations = incantations
		incantations = incantations[:0]

		for _, dec := range snippet.Decorators {
			if resolved, err := resolveCallable(dec.ExampleOf, graph, inducer); err == nil {
				dec.ExampleOf = resolved
				incantations = append(incantations, dec)
			}
		}
		snippet.Decorators = incantations
		incantations = incantations[:0]

		for _, attr := range snippet.Attributes {
			if resolved, err := resolveAttr(attr, graph, inducer); err == nil {
				attributes = append(attributes, resolved)
			}
		}
		snippet.Attributes = attributes

		attributes = attributes[:0]
		incantations = incantations[:0]

		if len(snippet.Attributes) == 0 && len(snippet.Incantations) == 0 && len(snippet.Decorators) == 0 {
			continue
		}

		buf, err := json.Marshal(&snippet)
		if err != nil {
			log.Fatalln("could not marshal snippet:", err)
		}
		w.Emit(r.Key(), buf)
	}

	if err := r.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}
}

// resolveCallable is a wrapper to resolve, but only returns if the resulting node is a callable identifier (type or function).
func resolveCallable(input string, graph *pythonimports.Graph, inducer *typeinduction.Client) (string, error) {
	if n, err := graph.Find(input); err == nil {
		if n.Classification != pythonimports.Type && n.Classification != pythonimports.Function {
			return "", fmt.Errorf("node not a type or function")
		}
		return input, nil
	}

	node, err := resolve(input, graph, inducer)
	if err != nil {
		return "", err
	}
	if node.Classification != pythonimports.Type && node.Classification != pythonimports.Function {
		return "", fmt.Errorf("node not a type or function")
	}
	return node.CanonicalName, nil
}

// resolveArg is a wrapper to resolve, but only returns if the resulting node is a type
func resolveArg(input string, graph *pythonimports.Graph, inducer *typeinduction.Client) (string, error) {
	if n, err := graph.Find(input); err == nil {
		if n.Classification != pythonimports.Type {
			return "", fmt.Errorf("node not a type")
		}
		return input, nil
	}

	node, err := resolve(input, graph, inducer)
	if err != nil {
		return "", err
	}
	if node.Classification != pythonimports.Type {
		return "", fmt.Errorf("node not a type")
	}
	return node.CanonicalName, nil
}

// resolveAttr is a wrapper to resolve
func resolveAttr(input string, graph *pythonimports.Graph, inducer *typeinduction.Client) (string, error) {
	if _, err := graph.Find(input); err == nil {
		return input, nil
	}

	node, err := resolve(input, graph, inducer)
	if err != nil {
		return "", err
	}
	return node.CanonicalName, nil
}

// resolve takes "chained" identifier names and resolve them to their fully qualified name. We do this because
// the python parser cannot resolve all identifiers (e.g, functions and other member attributes). So, the parser will
// return identifiers such as `requests.get.json`, which don't actually map to a real identifer, but result from code like:
//
//	x = requests.get("<some url>")
//  print x.json()
//
// This code then maps this identifier to `requests.models.Response.json` via the import graph and type induction.
func resolve(input string, graph *pythonimports.Graph, inducer *typeinduction.Client) (*pythonimports.Node, error) {
	var node *pythonimports.Node
	parts := strings.Split(input, ".")
	for idx, part := range parts {
		var err error
		if idx == 0 {
			node, err = graph.Find(part)
			if err != nil {
				return nil, err
			}
		} else {
			child, exists := node.Members[part]
			if !exists {
				return nil, fmt.Errorf("failed to find %s component in %s", part, input)
			}
			if child == nil {
				return nil, fmt.Errorf("node member %s has nil child Node", part)
			}
			node, exists = graph.FindByID(child.ID)
			if !exists {
				return nil, fmt.Errorf("could not find id %d for %s in  %s", child.ID, part, input)
			}
		}

		switch node.Classification {
		case pythonimports.Function, pythonimports.Descriptor:
			observed := typeinduction.Observation{
				ReturnedFrom: node.CanonicalName,
			}
			estimate := inducer.EstimateType(&observed)
			if estimate != nil {
				resolved := estimate.MostProbableType
				node, err = graph.Find(resolved)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if len(node.CanonicalName) == 0 {
		return nil, fmt.Errorf("empty canonical name")
	}
	return node, nil
}
