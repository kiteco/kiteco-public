package main

import (
	"log"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/builder"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/argspec"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/spf13/cobra"
)

func fail(err error) {
	if err != nil {
		log.Fatalln("[FATAL]", err)
	}
}

// merge merges typeshed and pkgexploration-sourced entities; it is destructive w.r.t. its arguments
func merge(typeshed, explored map[keytypes.Distribution]argspec.Entities) map[keytypes.Distribution]argspec.Entities {
	for dist, typeshedEnts := range typeshed {
		exploredEnts := explored[dist]

		// add typeshed entities to explored, if we didn't explore this distribution
		if exploredEnts == nil {
			explored[dist] = typeshedEnts
			continue
		}

		// otherwise, merge typeshed entities into explored entities
		for k, typeshedEnt := range typeshedEnts {
			if _, ok := exploredEnts[k]; ok {
				// both exist; do nothing
				// Once we'll have type info in argspec, copy typeinfo from typeshed to explored
			} else {
				// only typeshed exists; just copy it in
				exploredEnts[k] = typeshedEnt
			}
		}
	}
	return explored
}

func build(cmd *cobra.Command, args []string) {
	bOpts := builder.DefaultOptions
	bOpts.ManifestPath = args[0]
	bOpts.ResourceRoot = args[1]
	b := builder.New(bOpts)

	explored := buildGraph(args[2:])
	typeshed := buildTypeshed()
	shards := merge(typeshed, explored)

	for dist, ents := range shards {
		fail(b.PutResource(dist, ents))
	}
	fail(b.Commit())
}

var cmd = cobra.Command{
	Use:   "argspecs DST_MANIFEST DST_DATAPATH/ INPUT_GRAPH ...",
	Short: "generate argspec resource data from validated raw graphs",
	Args:  cobra.MinimumNArgs(3),
	Run:   build,
}

func main() {
	fail(cmd.Execute())
}
