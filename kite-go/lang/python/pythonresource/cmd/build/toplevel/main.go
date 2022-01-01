package main

import (
	"compress/gzip"
	"encoding/gob"
	"log"
	"os"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/distidx"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/toplevel"
	"github.com/spf13/cobra"
)

var (
	manifestPath string
	distidxPath  string
)

func init() {
	cmd.Flags().StringVarP(&manifestPath, "manifest", "g", "", "manifest path for symbol graph, docs, and symbol counts (defaults to compiled-in KiteManifest)")
	cmd.Flags().StringVarP(&distidxPath, "distidx", "d", "", "distribution index path (defaults to compiled-in KiteIndex)")
}

func build(cmd *cobra.Command, args []string) {
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

	if distidxPath != "" {
		dF, err := os.Open(distidxPath)
		if err != nil {
			log.Fatalln(err)
		}
		opts.DistIndex, err = distidx.New(dF)
		if err != nil {
			log.Fatalln(err)
		}
		dF.Close()
	}

	opts.ToplevelPath = ""
	resourceManager, errc := pythonresource.NewManager(opts)
	if err := <-errc; err != nil {
		log.Fatalln(err)
	}

	entries := make(toplevel.Entities)
	for _, dist := range opts.Manifest.Distributions() {
		pkgs, err := resourceManager.TopLevels(dist)
		if err != nil {
			panic(err)
		}
		for _, pkg := range pkgs {
			sym, err := resourceManager.NewSymbol(dist, pythonimports.NewPath(pkg))
			if err != nil {
				panic(err)
			}

			docs := resourceManager.Documentation(sym)
			if docs == nil {
				log.Printf("no docs for symbol %s", sym)
			}
			counts := resourceManager.SymbolCounts(sym)
			if counts == nil {
				log.Printf("no counts for symbol %s", sym)
			}

			key := toplevel.DistributionTopLevel{
				Distribution: dist,
				TopLevel:     pkg,
			}
			if _, ok := entries[key]; !ok {
				entries[key] = toplevel.Entity{
					Docs:   docs,
					Counts: counts,
				}
			}
		}
	}

	f, err := os.Create(args[0])
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	err = gob.NewEncoder(gz).Encode(entries)
	if err != nil {
		log.Fatalln(err)
	}
}

var cmd = cobra.Command{
	Use:   "toplevel --graph manifest.json toplevel.blob",
	Short: "generate docs and symbol dataset for top-level packages",
	Args:  cobra.ExactArgs(1),
	Run:   build,
}

func main() {
	cmd.Execute()
}
