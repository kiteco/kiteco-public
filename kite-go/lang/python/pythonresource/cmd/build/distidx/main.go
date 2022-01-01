package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/distidx"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/spf13/cobra"
)

func fail(err error) {
	if err != nil {
		log.Fatalln("[FATAL]", err)
	}
}

// TODO support incremental building of the index
// the best thing to do may be to just build a forward index of dist -> toplevels, and reverse it on load,
// since we can incrementally update the forward index very easily as we do with manifests

var manifestPath string

func init() {
	cmd.Flags().StringVar(&manifestPath, "graph", "", "symbol graph manifest path (defaults to compiled-in KiteManifest)")
}

func index(cmd *cobra.Command, args []string) {
	opts := pythonresource.DefaultOptions
	if manifestPath != "" {
		mF, err := os.Open(manifestPath)
		if err != nil {
			log.Fatalln(err)
		}
		opts.Manifest, err = manifest.New(mF)
		if err != nil {
			log.Fatalln(err)
		}
		mF.Close()
	}
	opts.Manifest = opts.Manifest.SymbolOnly()
	opts.DistIndex = nil
	rm, errc := pythonresource.NewManager(opts)
	if <-errc != nil {
		log.Fatalln("[FATAL] failed to init resource manager for service")
	}

	index := make(distidx.Index)
	for _, dist := range opts.Manifest.Distributions() {
		names, err := rm.TopLevels(dist)
		fail(err)
		for _, name := range names {
			index[name] = append(index[name], dist)
		}
	}

	w, err := os.Create(args[0])
	fail(err)
	defer w.Close()
	fail(json.NewEncoder(w).Encode(index))
}

var cmd = cobra.Command{
	Use:   "index DST.json",
	Short: "generate the symbol index from Python pkgexploration output",
	Args:  cobra.ExactArgs(1),
	Run:   index,
}

func main() {
	fail(cmd.Execute())
}
