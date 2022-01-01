package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/helpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/builder"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/distidx"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/kwargs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/spf13/cobra"

	"github.com/texttheater/golang-levenshtein/levenshtein"
)

// configurable constants
var (
	manifestPath, distidxPath, compatPath string
)

func init() {
	cmd.Flags().StringVar(&manifestPath, "graph", "", "symbol graph manifest path (defaults to compiled-in KiteManifest)")
	cmd.Flags().StringVar(&distidxPath, "distidx", "", "distribution index path (defaults to compiled-in KiteIndex)")
	cmd.Flags().StringVar(&compatPath, "compat", "", "compatibility index path")
}

// builder

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func build(cmd *cobra.Command, args []string) {
	bOpts := builder.DefaultOptions
	bOpts.ManifestPath = args[0]
	bOpts.ResourceRoot = args[1]
	b := builder.New(bOpts)

	kwOpts := pythoncode.DefaultKwargsOptions

	// Load compat
	if compatPath == "" {
		fail(fmt.Errorf("--compat argument required"))
	}
	index, err := helpers.LoadCompat(compatPath)
	fail(err)

	// Load all the old format data
	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	fail(err)

	data, err := pythoncode.LoadKwargsIndex(graph, kwOpts, pythoncode.DefaultKwargs)
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

	// Index returns kwargs response for every node, using the thresholding options passed in above via pythoncode.DefaultKwargsOptions
	kwargsIndex := data.Index()

	// Sharding
	idShards := make(map[keytypes.Distribution]map[pythonimports.Hash]int64) // for handling name collisions deterministically
	shards := make(map[keytypes.Distribution]kwargs.Entities)
	fail(index.Shard(rm, graph, func(sym pythonresource.Symbol, node *pythonimports.Node) {
		inKwargs := kwargsIndex[node.ID]
		if inKwargs == nil {
			return
		}

		dist := sym.Dist()
		pathHash := sym.PathHash()
		argSpec := rm.ArgSpec(sym)

		// dedupe
		idShard := idShards[dist]
		if idShard == nil {
			idShard = make(map[pythonimports.Hash]int64)
			idShards[sym.Dist()] = idShard
		}

		if prevNodeID, exists := idShard[pathHash]; exists {
			log.Printf("kwargs already exists for node %x with path %s (%d)\n", node.ID, sym.PathString(), pathHash)
			if node.ID <= prevNodeID {
				return
			}
		}
		idShard[pathHash] = node.ID

		// shard
		shard := shards[dist]
		if shard == nil {
			shard = make(kwargs.Entities)
			shards[dist] = shard
		}

		shard[pathHash] = transformKwargs(inKwargs, argSpec, kwOpts)
	}))

	for dist, rs := range shards {
		fail(b.PutResource(dist, rs))
	}
	fail(b.Commit())
}

func distanceFromArgs(arg string, spec *pythonimports.ArgSpec, addedArgs []kwargs.KeywordArg, opts pythoncode.KwargsOptions) bool {
	for _, a := range addedArgs {
		if levenshtein.DistanceForStrings([]rune(arg), []rune(a.Name), levenshtein.DefaultOptions) <= opts.MinDistance {
			return false
		}
	}

	if spec != nil {
		for _, a := range spec.Args {
			if a.Name != "" && levenshtein.DistanceForStrings([]rune(arg), []rune(a.Name), levenshtein.DefaultOptions) <= opts.MinDistance {
				return false
			}
		}
	}
	return true
}

func transformKwargs(in *response.PythonKwargs, argSpec *pythonimports.ArgSpec, opts pythoncode.KwargsOptions) kwargs.KeywordArgs {
	out := kwargs.KeywordArgs{
		Name: in.Name,
	}
	for _, kw := range in.Kwargs {
		if distanceFromArgs(kw.Name, argSpec, out.Kwargs, opts) {
			out.Kwargs = append(out.Kwargs, kwargs.KeywordArg{
				Name:  kw.Name,
				Types: kw.Types,
			})
		}
	}
	return out
}

// cli
var cmd = cobra.Command{
	Use:   "kwargs dst_manifest.json dst_data_dir/",
	Short: "generate kwargs resources from old kwargs data",
	Args:  cobra.ExactArgs(2),
	Run:   build,
}

func main() {
	cmd.Execute()
}
