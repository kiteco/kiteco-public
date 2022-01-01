package main

import (
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/helpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/rawgraph/types"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/builder"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/symgraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/stringutil"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/spf13/cobra"
)

func fail(err error) {
	if err != nil {
		log.Fatalln("[FATAL]", err)
	}
}

func load(inp string) types.ExplorationData {
	var dat types.ExplorationData
	r, err := os.Open(inp)
	fail(err)
	defer r.Close()
	fail(dat.Decode(r))
	return dat
}

func newNode(dat *types.NodeData) symgraph.Node {
	n := symgraph.Node{
		Canonical: symgraph.CastDottedPath(pythonimports.NewDottedPath(dat.CanonicalName)),
		Children:  make(symgraph.ChildMap),
	}

	switch dat.Classification {
	case "module":
		n.Kind = symgraph.Kind(keytypes.ModuleKind)
	case "type":
		n.Kind = symgraph.Kind(keytypes.TypeKind)
	case "descriptor":
		n.Kind = symgraph.Kind(keytypes.DescriptorKind)
	case "function":
		n.Kind = symgraph.Kind(keytypes.FunctionKind)
	case "object":
		n.Kind = symgraph.Kind(keytypes.ObjectKind)
	default:
		log.Printf("[SEVERE] unrecognized kind for %s\n", dat)
	}

	return n
}

func translateGraph(graph types.Graph, rootID types.NodeID) []symgraph.Node {
	root := graph[rootID]

	// add the root node first
	index := []symgraph.Node{newNode(root)}
	idxMap := make(map[types.NodeID]int)
	idxMap[root.ID] = 0

	for _, cur := range graph {
		if cur == root { // the root is processed separately above
			continue
		}
		if cur.Reference != "" { // skip external nodes
			continue
		}

		// construct the node and add it to the index
		idxMap[cur.ID] = len(index)
		index = append(index, newNode(cur))
	}

	// set the children, bases, type on our `Node`s
	for id, dat := range graph {
		ref, ok := idxMap[id]
		if !ok {
			continue // external node
		}
		node := &index[ref]

		for attr, childID := range dat.Children {
			if childIdx, ok := idxMap[childID]; ok {
				// Internal node, since it's indexed
				node.Children[stringutil.ToUint64(attr)] = symgraph.NodeRef{Internal: childIdx}
			} else {
				// External node, since it's not indexed
				childDat := graph[childID]
				path := pythonimports.NewDottedPath(childDat.Reference)
				node.Children[stringutil.ToUint64(attr)] = symgraph.NodeRef{External: symgraph.CastDottedPath(path)}
			}
		}

		for _, baseID := range dat.Bases {
			if baseIdx, ok := idxMap[baseID]; ok {
				node.Bases = append(node.Bases, symgraph.NodeRef{Internal: baseIdx})
			} else if baseDat := graph[baseID]; baseDat != nil {
				node.Bases = append(node.Bases, symgraph.NodeRef{External: symgraph.CastDottedPath(pythonimports.NewDottedPath(baseDat.Reference))})
			} else {
				log.Printf("[SEVERE] cannot locate base class node %d for %s", baseID, dat)
			}
		}

		if typeIdx, ok := idxMap[dat.TypeID]; ok {
			node.Type = &symgraph.NodeRef{Internal: typeIdx}
		} else if typeDat := graph[dat.TypeID]; typeDat != nil {
			node.Type = &symgraph.NodeRef{External: symgraph.CastDottedPath(pythonimports.NewDottedPath(typeDat.Reference))}
		} else {
			log.Printf("[SEVERE] cannot locate type node %d for %s", dat.TypeID, dat)
		}
	}

	return index
}

func translate(validated types.ExplorationData) symgraph.Graph {
	out := make(symgraph.Graph)
	for name, graph := range validated.TopLevels {
		out[string(name)] = translateGraph(graph, validated.RootIDs[name])
	}
	return out
}

func build(cmd *cobra.Command, args []string) {
	analyzedMap, err := loadAnalyzed()
	fail(err)

	bOpts := builder.DefaultOptions
	bOpts.ManifestPath = args[0]
	bOpts.ResourceRoot = args[1]
	b := builder.New(bOpts)

	for _, inp := range args[2:] {
		log.Printf("[INFO] processing input %s\n", inp)
		dat := load(inp)

		dist := keytypes.Distribution{Name: dat.PipPackage, Version: dat.PipVersion}
		graph := translate(dat)
		if len(graph) == 0 {
			log.Printf("[ERROR] empty translated symbol graph for distribution %s\n", dist)
			continue
		}

		if analyzedMap != nil {
			// TODO(naman) there are various bugs when trying to augment the graph with analysis-sourced symbols for non-stubs
			// so in the interest of expediency, we only do this for builtins & 3rd party stubs
			switch dist {
			case keytypes.BuiltinDistribution3:
				insertAnalyzed(dist, graph, analyzedMap[helpers.Builtin3StubKey])
			default:
				insertAnalyzed(dist, graph, analyzedMap[helpers.ThirdParty3StubKey])
				insertAnalyzed(dist, graph, analyzedMap[helpers.ThirdParty2StubKey])
			}
		}

		fail(b.PutResource(dist, &graph))
	}

	fail(b.Commit())
}

var cmd = cobra.Command{
	Use:   "symgraphs [--graph PREV_GRAPH.JSON --analyzed analyzed.json.gz] DST_MANIFEST DST_DATAPATH/ INPUT_GRAPH ...",
	Short: "generate symbol graph resource data from validated raw graphs",
	Args:  cobra.MinimumNArgs(3),
	Run:   build,
}

func main() {
	fail(cmd.Execute())
}
