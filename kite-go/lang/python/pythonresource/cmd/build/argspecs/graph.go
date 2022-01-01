package main

import (
	"log"
	"os"

	parser "github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/argspec"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/rawgraph/types"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/argspec"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

func load(inp string) types.ExplorationData {
	var dat types.ExplorationData
	r, err := os.Open(inp)
	fail(err)
	defer r.Close()
	fail(dat.Decode(r))
	return dat
}

func extract(validated types.ExplorationData) argspec.Entities {
	out := make(argspec.Entities)
	for _, g := range validated.TopLevels {
		for _, dat := range g {
			var e argspec.Entity

			if dat.ArgSpec != nil {
				e.Vararg = dat.ArgSpec.Vararg
				e.Kwarg = dat.ArgSpec.Kwarg

				// we discard annotation_type for now, until we add support for annotations
				for _, arg := range dat.ArgSpec.Args {
					e.Args = append(e.Args, pythonimports.Arg{
						Name:         arg.Name,
						DefaultType:  arg.DefaultType,
						DefaultValue: arg.DerivedDefaultValue(),
					})
				}
				for _, arg := range dat.ArgSpec.Kwonly {
					e.Args = append(e.Args, pythonimports.Arg{
						Name:         arg.Name,
						DefaultType:  arg.DefaultType,
						DefaultValue: arg.DerivedDefaultValue(),
						KeywordOnly:  true,
					})
				}
			} else {
				// only type (e.g. cProfile.Profile) and function nodes can have argspecs
				switch dat.Classification {
				case "function", "type":
				default:
					continue
				}

				// try to parse an argspec from the docstring (as per C functions from numpy, etc)
				if len(dat.Docstring) > 0 {
					spec, err := parser.Parse([]byte(dat.Docstring))

					if err != nil {
						log.Printf("[WARN] error while parsing argspec for %s : %s\n Corresponding docString : %s", dat, err, dat.Docstring)
					}
					if spec == nil {
						continue
					}
					log.Printf("[INFO] parsed (partial) argspec for %s", dat)

					e = *spec
				}
			}

			h := pythonimports.PathHash([]byte(dat.CanonicalName))
			out[h] = e
		}
	}

	return out
}

func buildGraph(files []string) map[keytypes.Distribution]argspec.Entities {
	res := make(map[keytypes.Distribution]argspec.Entities)
	for _, inp := range files {
		log.Printf("[INFO] processing input %s\n", inp)
		dat := load(inp)

		rs := extract(dat)
		if len(rs) > 0 {
			res[keytypes.Distribution{Name: dat.PipPackage, Version: dat.PipVersion}] = rs
		}
	}
	return res
}
