package main

import (
	"compress/gzip"
	"encoding/json"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/helpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/distidx"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/spf13/cobra"
)

func fail(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

var (
	manifestPath string
	distidxPath  string
)

func init() {
	cmd.Flags().StringVarP(&manifestPath, "graph", "g", "", "symbol graph manifest path (defaults to compiled-in KiteManifest)")
	cmd.Flags().StringVarP(&distidxPath, "distidx", "d", "", "distribution index path (defaults to compiled-in KiteIndex)")
}

func build(cmd *cobra.Command, args []string) {
	// Load the old import graph
	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
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

	// try using anypaths & canonical names to lookup symbols
	anypaths := pythonimports.ComputeAnyPaths(graph)

	fixed := make(map[int64]struct{}) // those nodes which were found using canonical path navigation should not be overwritten
	index := make(helpers.Compat)
	for i := range graph.Nodes {
		// don't iterate over _, n, because that copies the node, and we need the pointer to lookup anypaths
		n := &graph.Nodes[i]
		if _, ok := index[n.ID]; ok {
			panic("multiple nodes with same ID")
		}

		path := anypaths[n]
		if path.Empty() {
			path = n.CanonicalName
		}
		if path.Empty() {
			log.Printf("[ERROR] no (any)path found for %s\n", n)
			continue
		}

		sym, err := rm.PathSymbol(path)
		if err != nil {
			log.Printf("[WARN] %s\n", err)
			continue
		}

		if path.Hash == n.CanonicalName.Hash {
			fixed[n.ID] = struct{}{}
		}

		sym = sym.Canonical()
		index[n.ID] = keytypes.Symbol{
			Dist: sym.Dist(),
			Path: sym.Path(),
		}
	}

	// reverse index by looking up all symbols in the import graph
	for _, dist := range opts.Manifest.Distributions() {
		syms, err := rm.CanonicalSymbols(dist)
		fail(err)
		for _, sym := range syms {
			tentative := keytypes.Symbol{
				Dist: sym.Dist(),
				Path: sym.Path(),
			}

			n, err := graph.Navigate(tentative.Path)
			if err != nil {
				log.Printf("[WARN] could not navigate symbol %s in import graph\n", sym)
				continue
			}

			if current, ok := index[n.ID]; ok {
				if _, ok := fixed[n.ID]; ok || current.Less(tentative) { // keep only the smallest or fixed nodes
					log.Printf("[WARN] dropping symbol %s for %s in favor of %s\n", tentative, n, current)
					continue
				}
				log.Printf("[WARN] dropping symbol %s for %s in favor of %s\n", current, n, tentative)
			}

			index[n.ID] = tentative
		}
	}

	// compute statistics for remaining/missing nodes
	pkgCounts := make(map[string]uint64)
	for i := range graph.Nodes {
		n := &graph.Nodes[i]
		if _, ok := index[n.ID]; ok {
			continue
		}

		path := n.CanonicalName
		if p, ok := anypaths[n]; ok {
			path = p
		}
		pkgCounts[path.Head()]++
	}

	// print stats
	for pkg, cnt := range pkgCounts {
		log.Printf("[STATS] top-level %s: failed to lookup %d nodes\n", pkg, cnt)
	}

	// write the compat mapping
	f, err := os.Create(args[0])
	fail(err)
	gz := gzip.NewWriter(f)
	defer gz.Close()
	fail(json.NewEncoder(gz).Encode(index))
}

var cmd = cobra.Command{
	Use:   "compat --graph manifest.json compat.json.gz",
	Short: "generate canonical name index for backwards compatibility",
	Args:  cobra.ExactArgs(1),
	Run:   build,
}

func main() {
	cmd.Execute()
}
