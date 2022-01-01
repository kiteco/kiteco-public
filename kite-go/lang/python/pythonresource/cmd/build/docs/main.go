package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/helpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/rawgraph/types"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/builder"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/distidx"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/docs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/spf13/cobra"
)

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

var (
	manifestPath, distidxPath, compatPath string
)

func init() {
	cmd.Flags().StringVar(&manifestPath, "graph", "", "symbol graph manifest path (defaults to compiled-in KiteManifest)")
	cmd.Flags().StringVar(&distidxPath, "distidx", "", "distribution index path (defaults to compiled-in KiteIndex)")
	cmd.Flags().StringVar(&compatPath, "compat", "", "compatibility index path")
}

func buildGraph(files []string) map[keytypes.Distribution]docs.Entities {
	res := make(map[keytypes.Distribution]docs.Entities)
	for _, inp := range files {
		log.Printf("[INFO] processing input %s\n", inp)
		validated := func() types.ExplorationData {
			var dat types.ExplorationData
			r, err := os.Open(inp)
			fail(err)
			defer r.Close()
			fail(dat.Decode(r))
			return dat
		}()

		rs := make(docs.Entities)
		for _, g := range validated.TopLevels {
			for _, dat := range g {
				if dat.Docstring == "" {
					continue
				}

				h := pythonimports.PathHash([]byte(dat.CanonicalName))
				text := pythonlocal.DedentDocstring(dat.Docstring)
				rs[h] = docs.Entity{Text: text}
			}
		}

		if len(rs) > 0 {
			res[keytypes.Distribution{Name: validated.PipPackage, Version: validated.PipVersion}] = rs
		}
	}
	return res
}

func build(cmd *cobra.Command, args []string) {
	bOpts := builder.DefaultOptions
	bOpts.ManifestPath = args[0]
	bOpts.ResourceRoot = args[1]
	b := builder.New(bOpts)

	// Load compat
	if compatPath == "" {
		fail(fmt.Errorf("--compat argument required"))
	}
	index, err := helpers.LoadCompat(compatPath)
	fail(err)

	// Load all the old format data
	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	fail(err)

	corpus, err := pythondocs.LoadCorpus(graph, pythondocs.DefaultSearchOptions)
	fail(err)

	// Load the symbol graph
	opts := pythonresource.DefaultOptions
	if manifestPath != "" {
		mF, err := os.Open(manifestPath)
		fail(err)
		opts.Manifest, err = manifest.New(mF)
		fail(err)
		mF.Close()
	}
	opts.Manifest = opts.Manifest.SymbolOnly()
	if distidxPath != "" {
		dF, err := os.Open(distidxPath)
		fail(err)
		opts.DistIndex, err = distidx.New(dF)
		fail(err)
		dF.Close()
	}
	rm, errc := pythonresource.NewManager(opts)
	fail(<-errc)

	shards := buildGraph(args[2:])                                           // docstrings
	idShards := make(map[keytypes.Distribution]map[pythonimports.Hash]int64) // for handling name collisions deterministically

	// reshard old rich docs, overriding rich docs generated from docstrings
	fail(index.Shard(rm, graph, func(sym pythonresource.Symbol, node *pythonimports.Node) {
		entity, _ := corpus.Entity(node)
		if entity == nil || entity.StructuredDoc == nil || entity.StructuredDoc.DescriptionHTML == "" {
			return
		}

		dist := sym.Dist()
		pathHash := sym.PathHash()

		// check for duplicates
		idShard := idShards[dist]
		if idShard == nil {
			idShard = make(map[pythonimports.Hash]int64)
			idShards[sym.Dist()] = idShard
		}

		if prevID, exists := idShard[pathHash]; exists {
			log.Printf("documentation already found for path %d %s (ID %x)\n", pathHash, sym.PathString(), node.ID)
			if node.ID <= prevID {
				return
			}
		}
		idShard[pathHash] = node.ID

		// shard
		shard := shards[dist]
		if shard == nil {
			shard = make(docs.Entities)
			shards[dist] = shard
		}

		// maintain any existing Text docstring
		ent := shard[pathHash]
		ent.HTML = entity.StructuredDoc.DescriptionHTML
		shard[pathHash] = ent
	}))

	var totalCount int
	for dist, rs := range shards {
		for _, ent := range rs {
			if ent.Text != "" || ent.HTML != "" {
				totalCount++
			}
		}
		fail(b.PutResource(dist, rs))
	}
	fail(b.Commit())
	log.Printf("generated docs for %d total symbols", totalCount)
}

var cmd = cobra.Command{
	Use:   "documentation DST_MANIFEST.json DST_DATA_DIR/ INPUT_GRAPH ...",
	Short: "generate documentation resources from old documentation data",
	Args:  cobra.MinimumNArgs(2),
	Run:   build,
}

func main() {
	cmd.Execute()
}
