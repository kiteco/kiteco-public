package main

import (
	"compress/gzip"
	"encoding/json"
	"log"
	"os"
	"sort"

	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/helpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/helpers/rettypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/builder"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/distidx"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/returntypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/serialization"
	"github.com/spf13/cobra"
)

var (
	manifestPath   string
	distidxPath    string
	analyzedPath   string
	docsPath       string
	includeEMModel bool
)

var (
	dynamicUsagesPath      = "s3://kite-data/dynamic-usages/20190320.json.gz"
	dynamicReturnTypesPath = pythoncuration.DefaultSearchOptions.CurationRoot + "/return-types.json.gz"
	emModelDir             = pythoncode.EMModelRoot
	emBetterThanUniform    = 0.2
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func computeReturnTypes(key string, rm pythonresource.Manager, analyzed *pythonenv.SourceTree, out map[keytypes.Distribution]returntypes.Entities) {
	seenMap := make(map[pythonimports.Hash]struct{})

	for _, tlSym := range analyzed.ListAbs("/") {
		q := []pythontype.SourceValue{tlSym.Value.(pythontype.SourceValue)}
		for len(q) > 0 {
			val := q[0]
			q = q[1:]

			addr := val.Address()
			if addr.Nil() {
				log.Printf("[DEBUG] nil address for explored value %v", map[string]interface{}{
					"key": key,
					"val": val,
				})
				continue
			}

			path, err := helpers.QualifiedPathAnalysis(addr)
			fail(err)
			if _, seen := seenMap[path.Hash]; seen {
				continue
			}
			seenMap[path.Hash] = struct{}{}

			var table map[string]*pythontype.Symbol
			switch val := val.(type) {
			case *pythontype.SourceModule:
				table = val.Members.Table
			case *pythontype.SourcePackage:
				if val.Init != nil {
					q = append(q, val.Init) // walk the init module
				}
				table = val.DirEntries.Table
			case *pythontype.SourceClass:
				table = val.Members.Table
			case *pythontype.SourceFunction:
				ret := val.Call(pythontype.Args{})
				if ret == nil {
					log.Printf("[DEBUG] no return value %v", map[string]interface{}{
						"key":  key,
						"val":  val,
						"path": path,
					})
					continue
				}

				syms, err := rm.PathSymbols(kitectx.Background(), path)
				if err != nil {
					log.Printf("[DEBUG] no symbols found for explored value path %v", map[string]interface{}{
						"key":  key,
						"val":  val,
						"path": path,
						"err":  err,
					})
					continue
				}

				for _, retType := range pythontype.DisjunctsNoCtx(pythontype.TranslateNoCtx(ret.Type(), rm)) {
					addr := retType.Address()
					if addr.Nil() {
						log.Printf("[DEBUG] nil address for return type value %v", map[string]interface{}{
							"key":     key,
							"val":     val,
							"path":    path,
							"retType": retType,
						})
						continue
					}

					retPath, err := helpers.QualifiedPathAnalysis(addr)
					fail(err)

					retSyms, _ := rm.PathSymbols(kitectx.Background(), retPath)
					if err != nil {
						log.Printf("[DEBUG] no symbols found for return type path %v", map[string]interface{}{
							"key":     key,
							"val":     val,
							"path":    path,
							"retPath": retPath,
							"err":     err,
						})
						continue
					}

					for _, sym := range syms {
						sym = sym.Canonical()
						shard := out[sym.Distribution()]
						if shard == nil {
							shard = make(returntypes.Entities)
							out[sym.Distribution()] = shard
						}
						for _, retSym := range retSyms {
							retSym = retSym.Canonical()
							log.Printf("[INFO] adding return type %v", map[string]interface{}{
								"key":    key,
								"val":    val,
								"sym":    sym,
								"retSym": retSym,
							})
							symPathHash := uint64(sym.PathHash())
							if shard[symPathHash] == nil {
								shard[symPathHash] = make(returntypes.Entity)
							}
							shard[symPathHash][retSym.PathString()] |= keytypes.StubTruthiness
						}
					}
				}
			}

			if table != nil {
				for _, sym := range table {
					// exploring private symbols is fine here, since we're not modifying the graph
					switch v := sym.Value.(type) {
					case pythontype.SourceValue:
						q = append(q, v)
					case pythontype.Union:
						for _, v := range v.Constituents {
							if v, ok := v.(pythontype.SourceValue); ok {
								q = append(q, v)
							}
						}
					}
				}
			}
		}
	}
	return
}

func addFromDocs(rm pythonresource.Manager, fromDocs []rettypes.DistReturns, out map[keytypes.Distribution]returntypes.Entities) {
	for _, dr := range fromDocs {
		outShard := out[dr.Dist]
		if outShard == nil {
			outShard = make(returntypes.Entities)
			out[dr.Dist] = outShard
		}

		for pathHash, fromDocTypes := range dr.Returns {
			outTypes := outShard[pathHash]

			// validate & merge fromDocTypes into outTypes
			if len(outTypes) == 0 {
				outShard[pathHash] = fromDocTypes
				continue
			}

			for typePathStr, truthiness := range fromDocTypes {
				typeSym, err := rm.PathSymbol(pythonimports.NewDottedPath(typePathStr))
				if err != nil {
					log.Printf("[ERROR] fromDocs: symbol graph does not contain type path %s", typePathStr)
					continue
				}
				if rm.Kind(typeSym) != keytypes.TypeKind {
					log.Printf("[WARN] fromDocs: ignoring non-type symbol for path %s", typePathStr)
					continue
				}
				outTypes[typePathStr] |= truthiness
			}
		}
	}
}

func addFromEMModel(rm pythonresource.Manager, out map[keytypes.Distribution]returntypes.Entities) {
	files, err := fileutil.ListDir(emModelDir)
	fail(err)

	for _, file := range files {
		f, err := fileutil.NewCachedReader(file)
		fail(err)
		defer f.Close()

		var funcAndReturnTypes []pythoncode.EMReturnTypes
		fail(json.NewDecoder(f).Decode(&funcAndReturnTypes))

		for _, funcWithTypes := range funcAndReturnTypes {
			dist := funcWithTypes.Dist
			funcPath := funcWithTypes.Func
			candidates := funcWithTypes.ReturnTypes

			funcSym, err := rm.NewSymbol(dist, funcPath)
			if err != nil {
				log.Printf("[WARN] from-em-model: symbol graph does not contain function path %s", funcPath)
				continue
			}

			if rm.Kind(funcSym) != keytypes.FunctionKind {
				log.Printf("[ERROR] from-em-model: ignoring non-function symbol for path %s", funcPath)
				continue
			}
			funcSym = funcSym.Canonical()

			shard := out[dist]
			if shard == nil {
				shard = make(returntypes.Entities)
				out[dist] = shard
			}

			funcPathHash := uint64(funcSym.PathHash())
			ent := shard[funcPathHash]
			// For the functions that already have return types from other sources, don't add them
			if ent != nil {
				continue
			}

			var filtered []pythonresource.Symbol
			for _, cand := range candidates {
				typeDist := cand.Dist
				typePath := cand.Sym
				typeProb := cand.Prob
				typeSym, err := rm.NewSymbol(typeDist, typePath)
				if err != nil {
					log.Printf("[ERROR] from-em-model: symbol graph does not contain type path %s", typePath)
					continue
				}
				if rm.Kind(typeSym) != keytypes.TypeKind {
					log.Printf("[WARN] from-em-model: ignoring non-type symbol for path %s", typePath)
					continue
				}
				// If there are multiple candidates, only include the ones with probability 20% higher than uniform
				if len(candidates) > 1 && typeProb < (1+emBetterThanUniform)*(1/float64(len(candidates))) {
					log.Printf("[WARN] from-em-model: ignoring type symbol %s, prob %f too small", typePath, typeProb)
					continue
				}
				filtered = append(filtered, typeSym)
			}

			if len(filtered) == 0 {
				log.Printf("[WARN] from-em-model: no valid return type for function path %s", funcPath)
				continue
			}

			ent = make(returntypes.Entity)
			shard[funcPathHash] = ent

			for _, sym := range filtered {
				log.Printf("[DEBUG] from-em-model: added return type %s for %s", sym.Canonical().PathString(), funcSym)
				ent[sym.Canonical().PathString()] |= keytypes.EMModelTruthiness
			}
		}
	}
}

func addFromDynamicReturnTypes(rm pythonresource.Manager, out map[keytypes.Distribution]returntypes.Entities) {
	gzR, err := fileutil.NewCachedReader(dynamicReturnTypesPath)
	fail(err)
	jsonR, err := gzip.NewReader(gzR)
	fail(err)

	var types map[string][]string
	fail(json.NewDecoder(jsonR).Decode(&types))

	for funcPath, retPaths := range types {
		funcSyms, err := rm.PathSymbols(kitectx.Background(), pythonimports.NewDottedPath(funcPath))
		if err != nil {
			log.Printf("[WARN] dynamic-return: symbol graph does not contain function path %s", funcPath)
			continue
		}
		sort.Slice(funcSyms, func(i, j int) bool {
			iKind := rm.Kind(funcSyms[i])
			jKind := rm.Kind(funcSyms[j])
			return iKind == keytypes.FunctionKind && jKind != keytypes.FunctionKind
		})
		for i, sym := range funcSyms {
			if rm.Kind(sym) != keytypes.FunctionKind {
				funcSyms = funcSyms[:i]
				break
			}
			funcSyms[i] = sym.Canonical()
		}
		if len(funcSyms) == 0 {
			log.Printf("[WARN] dynamic-return: no symbols of function kind match path %s", funcPath)
			continue
		}

		for _, retPath := range retPaths {
			retSym, err := rm.PathSymbol(pythonimports.NewDottedPath(retPath))
			if err != nil {
				newPath := "builtins." + retPath
				retSym, err = rm.PathSymbol(pythonimports.NewDottedPath(newPath))
				if err == nil {
					retPath = newPath
				}
			}
			if err != nil {
				newPath := "builtins." + retPath
				retSym, err = rm.PathSymbol(pythonimports.NewDottedPath(newPath))
				if err == nil {
					retPath = newPath
				}
			}
			if err != nil {
				log.Printf("[WARN] dynamic-return: symbol graph does not contain type path %s", retPath)
				continue
			}
			if rm.Kind(retSym) != keytypes.TypeKind {
				log.Printf("[ERROR] dynamic-return: ignoring non-type symbol for path %s", retPath)
				continue
			}

			for _, fnSym := range funcSyms {
				dist := fnSym.Dist()
				shard := out[dist]
				if shard == nil {
					shard = make(returntypes.Entities)
					out[dist] = shard
				}

				fnPathHash := uint64(fnSym.PathHash())
				ent := shard[fnPathHash]
				if ent == nil {
					ent = make(returntypes.Entity)
					shard[fnPathHash] = ent
				}

				if ent[retPath] == 0 {
					log.Printf("[DEBUG] dynamic-return: added return type %s for %s", retPath, fnSym)
				}
				ent[retPath] |= keytypes.DynamicAnalysisTruthiness
			}
		}
	}
}

func addFromDynamicUsages(rm pythonresource.Manager, out map[keytypes.Distribution]returntypes.Entities) {
	fail(serialization.Decode(dynamicUsagesPath, func(u *dynamicanalysis.Usage) {
		if u.ReturnedFrom == "" || u.Type == "" {
			return
		}

		funcPath := pythonimports.NewDottedPath(u.ReturnedFrom)
		typePath := pythonimports.NewDottedPath(u.Type)

		typeSym, err := rm.PathSymbol(typePath)
		if err != nil {
			log.Printf("[WARN] dynamic: symbol graph does not contain type path %s", u.Type)
			return
		}
		if rm.Kind(typeSym) != keytypes.TypeKind {
			log.Printf("[ERROR] dynamic: ignoring non-type symbol for path %s", u.Type)
			return
		}

		funcSyms, err := rm.PathSymbols(kitectx.Background(), funcPath)
		if err != nil {
			log.Printf("[WARN] dynamic: symbol graph does not contain function path %s", u.ReturnedFrom)
			return
		}

		for _, fnSym := range funcSyms {
			if rm.Kind(fnSym) != keytypes.FunctionKind {
				log.Printf("[ERROR] dynamic: ignoring non-function symbol for path %s", u.ReturnedFrom)
				continue
			}
			fnSym = fnSym.Canonical()

			dist := fnSym.Dist()
			shard := out[dist]
			if shard == nil {
				shard = make(returntypes.Entities)
				out[dist] = shard
			}

			fnPathHash := uint64(fnSym.PathHash())
			ent := shard[fnPathHash]
			if ent == nil {
				ent = make(returntypes.Entity)
				shard[fnPathHash] = ent
			}

			log.Printf("[DEBUG] dynamic: added return type %s for %s", u.Type, fnSym)
			ent[u.Type] |= keytypes.DynamicAnalysisTruthiness
		}
	}))
}

func build(cmd *cobra.Command, args []string) {
	if analyzedPath == "" && docsPath == "" {
		log.Fatalln("at least one of --analyzed or --docs must be passed")
	}

	// resource manager
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

	out := make(map[keytypes.Distribution]returntypes.Entities)
	// rettypes from analyzed stubs & packages
	if analyzedPath != "" {
		analyzedMap, err := helpers.LoadAnalyzed(analyzedPath, rm)
		fail(err)
		for key, analyzed := range analyzedMap {
			computeReturnTypes(key, rm, analyzed, out)
		}
	}
	// rettypes from docstrings
	if docsPath != "" {
		fromDocs := func() []rettypes.DistReturns {
			docsF, err := os.Open(docsPath)
			fail(err)
			defer docsF.Close()
			all, err := rettypes.DecodeAll(docsF)
			fail(err)
			return all
		}()
		addFromDocs(rm, fromDocs, out)
	}

	addFromDynamicUsages(rm, out)
	addFromDynamicReturnTypes(rm, out)

	// rettypes from emmodel
	if includeEMModel {
		addFromEMModel(rm, out)
	}

	// builder
	bOpts := builder.DefaultOptions
	bOpts.ManifestPath = args[0]
	bOpts.ResourceRoot = args[1]
	b := builder.New(bOpts)
	for dist, rs := range out {
		fail(b.PutResource(dist, rs))
	}
	fail(b.Commit())
}

func init() {
	cmd.Flags().StringVarP(&manifestPath, "graph", "g", "", "symbol graph manifest path (defaults to compiled-in KiteManifest)")
	cmd.Flags().StringVarP(&distidxPath, "distidx", "d", "", "distribution index path (defaults to compiled-in KiteIndex)")
	cmd.Flags().StringVarP(&analyzedPath, "analyzed", "y", "", "result of offline analyzer (analyze)")
	cmd.Flags().StringVarP(&docsPath, "docs", "c", "", "result of docs_returntypes")
	cmd.Flags().BoolVarP(&includeEMModel, "emmodel", "e", true, "result of EM model based on attribute access")
}

var cmd = cobra.Command{
	Use:   "returntypes --graph in_symgraph.json --analyzed analyzed.json.gz --docs docs_rettypes.json dst_manifest.json dst_data/",
	Short: "generate returntypes resource data, mapping function symbols to their return types",
	Args:  cobra.ExactArgs(2),
	Run:   build,
}

func main() {
	cmd.Execute()
}
