package main

import (
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/distidx"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/argspec"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

var (
	typeshedPath              string
	manifestPath, distidxPath string
)

func init() {
	cmd.Flags().StringVar(&typeshedPath, "typeshed", "./typeshed", "typeshed repository path")
	cmd.Flags().StringVar(&manifestPath, "graph", "", "symbol graph manifest path (defaults to compiled-in KiteManifest)")
	cmd.Flags().StringVar(&distidxPath, "distidx", "", "distribution index path (defaults to compiled-in KiteIndex)")
}

// lookuper is a function that lookups a symbol by path string
type lookuper = func(pythonresource.Manager, string) (pythonresource.Symbol, error)

func builtinLookuper(version string) lookuper {
	return func(rm pythonresource.Manager, path string) (pythonresource.Symbol, error) {
		return rm.NewSymbol(keytypes.Distribution{Name: "builtin-stdlib", Version: version}, pythonimports.NewDottedPath(path))
	}
}

func thirdPartyLookuper(rm pythonresource.Manager, path string) (pythonresource.Symbol, error) {
	return rm.PathSymbol(pythonimports.NewDottedPath(path))
}

func buildTypeshed() map[keytypes.Distribution]argspec.Entities {
	// input encapsulates an item of work to process when sharding the typeshed argspecs
	type input struct {
		lookup lookuper
		specs  map[string]argspec.Entity
		src    string
	}

	// collect typeshed argspecs into a list of inputs
	process := func(s string) map[string]argspec.Entity { return collect(fmt.Sprintf("%s/%s/", typeshedPath, s)) }
	builtin6 := process("stdlib/2and3")
	inps := []input{
		input{builtinLookuper(keytypes.BuiltinDistribution3.Version), process("stdlib/3"), "stdlib/3"},
		input{builtinLookuper(keytypes.BuiltinDistribution3.Version), builtin6, "stdlib/2and3"},
		input{thirdPartyLookuper, process("third_party/3"), "third_party/3"},
		input{thirdPartyLookuper, process("third_party/2and3"), "third_party/2and3"},
		input{thirdPartyLookuper, process("third_party/2"), "third_party/2"},
	}

	// load the symbol graph
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

	rm, errc := pythonresource.NewManager(opts)
	if err := <-errc; err != nil {
		log.Fatalln("[FATAL]", err)
	}

	// process inputs in order, sharding by distribution/symbol
	sources := make(map[keytypes.Distribution]map[pythonimports.Hash]string) // for nice errors
	shards := make(map[keytypes.Distribution]argspec.Entities)
	for _, inp := range inps {
		for path, spec := range inp.specs {
			sym, err := inp.lookup(rm, path)
			if err != nil {
				log.Printf("[WARN] skipping argspec for unresolvable symbol path %s: %s\n", path, err)
				continue
			}

			processer := func(sym pythonresource.Symbol) {
				pathHash := sym.PathHash()

				shard := shards[sym.Dist()]
				if shard == nil {
					shard = make(argspec.Entities)
					shards[sym.Dist()] = shard
				}

				srcShard := sources[sym.Dist()]
				if srcShard == nil {
					srcShard = make(map[pythonimports.Hash]string)
					sources[sym.Dist()] = srcShard
				}

				if prevSpec, ok := shard[pathHash]; ok {
					prevSrc := srcShard[pathHash]
					if !reflect.DeepEqual(prevSpec, spec) {
						log.Printf("[WARN] dropping argspec for %s from %s (differing specfound in %s)\n", sym, inp.src, prevSrc)
					}
					return
				}

				shard[pathHash] = spec
				srcShard[pathHash] = inp.src
			}
			processer(sym)
			processer(sym.Canonical())

		}
	}

	return shards
}
