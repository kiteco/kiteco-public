package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
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

	for r.Next() {
		var snippet pythoncode.Snippet
		err = json.Unmarshal(r.Value(), &snippet)
		if err != nil {
			log.Fatal(err)
		}

		// Emit each incantation, keyed by the name of each incantation.
		for _, inc := range snippet.Incantations {
			buf, err := json.Marshal(inc)
			if err != nil {
				log.Fatal(err)
			}

			// We want to group incantations by their node id's so that we
			// collect all ways that this method is called.
			node, err := graph.Find(inc.ExampleOf)
			if err != nil {
				continue
			}

			if validIncantation(inc, node) {
				err = w.Emit(fmt.Sprintf("%d", node.ID), buf)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	if err := r.Err(); err != nil {
		log.Fatalln("error reading stdin:", err)
	}
}

// validIncantation determines whether the arguments parsed in the provided incantation
// conform to the argspec in `node`. # of positional arguments are enforced if there is no
// vararg, and keyword names are enforced if there is no **kwarg argument.
func validIncantation(inc *pythoncode.Incantation, node *pythonimports.Node) bool {
	argSpec := node.ArgSpec
	if argSpec == nil {
		return true
	}

	var args int
	kwargs := make(map[string]bool)
	for _, arg := range argSpec.Args {
		if arg.DefaultType == "" {
			args++
		} else {
			kwargs[arg.Name] = true
		}
	}

	// Only enforce positional argument check if Vararg is empty
	if argSpec.Vararg == "" {
		if args != len(inc.Args) {
			return false
		}
	}

	// Only enforce keyword argument check if Kwarg is empty
	if argSpec.Kwarg == "" {
		for _, arg := range inc.Kwargs {
			if _, exists := kwargs[arg.Key]; !exists {
				return false
			}
		}
	}

	return true
}
