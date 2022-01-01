package main

import (
	"log"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/builder"
	rmcounts "github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/spf13/cobra"
)

// TODO(naman) next time we run this, we may want to do more offline filtering on aliases, since we only serve sufficiently popular aliases anyways

var manifestPath, distidxPath, cacheDir string

func init() {
	cmd.Flags().StringVar(&manifestPath, "graph", "", "symbol graph manifest path (defaults to compiled-in KiteManifest)")
	cmd.Flags().StringVar(&distidxPath, "distidx", "", "distribution index path (defaults to compiled-in KiteIndex)")
	cmd.Flags().StringVar(&cacheDir, "cachedir", "/tmp/kite-local-pipelines", "cache directory")
}

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func addCounts(counts symbolcounts.Counts, hcs []pythoncode.HashCounts) symbolcounts.Counts {
	for _, hc := range hcs {
		counts.Import += int(hc.Counts.Import)
		counts.ImportThis += int(hc.Counts.Import)
		counts.Name += int(hc.Counts.Name)
		counts.Attribute += int(hc.Counts.Attribute)
		counts.Expr += int(hc.Counts.Expr)
		if counts.ImportAliases == nil && len(hc.Counts.ImportAliases) > 0 {
			counts.ImportAliases = make(map[string]int)
		}
		for alias, count := range hc.Counts.ImportAliases {
			counts.ImportAliases[alias] += int(count)
		}
	}
	return counts
}

func addChildCounts(counts symbolcounts.Counts, hcs []pythoncode.HashCounts) symbolcounts.Counts {
	for _, hc := range hcs {
		counts.Import += int(hc.Counts.Import)
		counts.Name += int(hc.Counts.Name)
		counts.Attribute += int(hc.Counts.Attribute)
		// TODO(naman) does this make sense? if e.g. we counted the return value of a function,
		// should that symbol's ancestors counts be incremented?
		counts.Expr += int(hc.Counts.Expr)
		// explicitly don't add ImportThis or ImportAliases
	}
	return counts
}

func build(cmd *cobra.Command, args []string) {
	index, err := pythoncode.NewSymbolToHashesIndex(pythoncode.SymbolToHashesIndexPath, cacheDir)
	if err != nil {
		log.Fatalln("cannot open symbol counts dataset: ", err)
	}

	bOpts := builder.DefaultOptions
	bOpts.ManifestPath = args[0]
	bOpts.ResourceRoot = args[1]
	b := builder.New(bOpts)

	opts, err := pythonresource.DefaultOptions.WithCustomPaths(manifestPath, distidxPath)
	fail(err)
	symGraph, errc := pythonresource.NewManager(opts)
	fail(<-errc)

	shards := make(map[keytypes.Distribution]rmcounts.Entities)
	fail(index.IterateSlowly(func(pathStr string, hcs []pythoncode.HashCounts) error {
		path := pythonimports.NewDottedPath(pathStr)
		sym, err := symGraph.PathSymbol(path)
		if err != nil {
			log.Printf("ERROR no symbol for path string %s (%s)", pathStr, err)
			return nil
		}

		shard := shards[sym.Dist()]
		if shard == nil {
			shard = make(rmcounts.Entities)
			shards[sym.Dist()] = shard
		}

		// add counts to symbol, and (partially) to all predecessors
		shard[pathStr] = addCounts(shard[pathStr], hcs)
		for path = path.Predecessor(); !path.Empty(); path = path.Predecessor() {
			prefix := path.String()
			shard[prefix] = addChildCounts(shard[prefix], hcs)
		}
		return nil
	}))

	for dist, rs := range shards {
		if err := b.PutResource(dist, rs); err != nil {
			log.Fatalln(err)
		}
	}

	if err = b.Commit(); err != nil {
		log.Fatalln(err)
	}
}

var cmd = cobra.Command{
	Use:   "symbolcounts dst_manifest.json dst_data_dir/",
	Short: "generate symbol count resources from python-module-stats pipeline output",
	Args:  cobra.ExactArgs(2),
	Run:   build,
}

func main() {
	cmd.Execute()
}
