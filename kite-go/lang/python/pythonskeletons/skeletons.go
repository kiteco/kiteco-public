package pythonskeletons

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonskeletons/internal/skeleton"
	"github.com/kiteco/kiteco/kite-go/typeinduction"
)

var (
	indexSkel skeleton.Builder
	indexOnce sync.Once
)

func index() skeleton.Builder {
	indexOnce.Do(func() {
		err := loadIndex()
		if err != nil {
			panic("error loading skeletons: " + err.Error())
		}
	})
	return indexSkel
}

func loadIndex() error {
	gz, err := gzip.NewReader(bytes.NewBuffer(MustAsset("skeletons.gob.gz")))
	if err != nil {
		return err
	}
	if err := gob.NewDecoder(gz).Decode(&indexSkel); err != nil {
		return err
	}
	return nil
}

// UpdateGraph with python skeletons
func UpdateGraph(graph *pythonimports.Graph) error {
	log.Printf("Updating pythonimports.Graph with skeletons")

	builtins, err := pythonimports.NewBuiltinCache(graph)
	if err != nil {
		return fmt.Errorf("error constructing builtin cache: %v", err)
	}

	linker := newLinker(graph, builtins)

	// modules
	for _, mod := range index().Modules {
		node := linker.Link(mod.Path, pythonimports.Module)
		if node == nil {
			log.Printf("%30s not found in graph or could not be built, skipping\n", mod.Path.String())
			continue
		}

		// submodules
		for name, mod := range mod.SubModules {
			node.Members[name] = linker.Link(mod.Path, pythonimports.Module)
		}

		for name, path := range mod.Attrs {
			// TODO(juan): for now all attrs are assumed to be instances
			node.Members[name] = linker.LinkInstance(pythonimports.NewDottedPath(path))
		}

		for name, path := range mod.Functions {
			node.Members[name] = linker.Link(pythonimports.NewDottedPath(path), pythonimports.Function)
		}

		for name, path := range mod.Types {
			node.Members[name] = linker.Link(pythonimports.NewDottedPath(path), pythonimports.Type)
		}
	}

	// types
	for _, ty := range index().Types {
		node := linker.Link(ty.Path, pythonimports.Type)
		if node == nil {
			log.Printf("%30s not found in graph or could not be built, skipping\n", ty.Path.String())
			continue
		}

		// methods
		for name, path := range ty.Methods {
			node.Members[name] = linker.Link(pythonimports.NewDottedPath(path), pythonimports.Function)
		}

		// attrs
		for name, path := range ty.Attrs {
			// TODO(juan): for now all attrs are assumed to be instances
			node.Members[name] = linker.LinkInstance(pythonimports.NewDottedPath(path))
		}

		// bases
		for _, basename := range ty.Bases {
			base := linker.Link(pythonimports.NewDottedPath(basename), pythonimports.Type)
			if base == nil {
				log.Printf("%30s base class (of class %s) not found in graph or could not be built, skipping\n", basename, ty.Path.String())
				continue
			}
			node.Bases = append(node.Bases, base)
		}
	}

	// functions/methods
	for _, fn := range index().Functions {
		if node := linker.Link(fn.Path, pythonimports.Function); node == nil {
			log.Printf("%30s not found in graph or could not be built, skipping \n", fn.Path.String())
		}
	}

	// attrs
	for _, attr := range index().Attrs {
		ppath := pythonimports.NewPath(attr.Path.Parts[:len(attr.Path.Parts)-1]...)
		parent := linker.Link(ppath, pythonimports.None)
		if parent == nil {
			log.Printf("%30s (for attr %s) not found in graph or could not be built, skipping\n", ppath.String(), attr.Path.Last())
			continue
		}

		// TODO(juan): for now all attrs are assumed to be instances
		parent.Members[attr.Path.Last()] = linker.LinkInstance(pythonimports.NewDottedPath(attr.Type))
	}

	// need to recompute anypaths
	graph.AnyPaths = pythonimports.ComputeAnyPaths(graph)

	return nil
}

// UpdateReturnTypes with python skeletons.
// NOTE: this does NOT update the graph with new functions or types
func UpdateReturnTypes(graph *pythonimports.Graph, client *typeinduction.Client) {
	log.Printf("Updating typeinduction.Client with skeleton function return types\n")
	var returns, skipped int
	for _, fn := range index().Functions {
		if len(fn.Return) == 0 {
			continue
		}
		returns++

		fsym, err := client.SymbolGraph.PathSymbol(fn.Path)
		if err != nil {
			skipped++
			continue
		}

		var rts []*pythonimports.Node
		for _, rt := range fn.Return {
			tnode := resolveType(rt, graph)
			if tnode == nil || tnode.CanonicalName.Empty() {
				log.Printf("unable to find return type `%s` for function `%s`, skipping type\n", rt, fn.Path.String())
				continue
			}
			rts = append(rts, tnode)
		}

		if len(rts) == 0 {
			log.Printf("no return types found in graph for skeleton function `%s`, skipping", fn.Path)
			skipped++
			continue
		}

		ftable := client.Functions[fsym.PathHash()]
		if ftable == nil {
			ftable = &typeinduction.ResolvedFunction{
				Symbol: fsym,
			}
			client.Functions[fsym.PathHash()] = ftable
		}

		// clear existing return types and overwrite
		// with uniform distribution over provided return types
		ftable.ReturnType = ftable.ReturnType[:0]
		lp := -math.Log(float64(len(rts)))
		assertFinite(lp, "nonfinite log probablilty for function %s, has %d skeleton return types\n", fn.Path, len(rts))
		for _, rt := range rts {
			sym, err := client.SymbolGraph.PathSymbol(rt.CanonicalName)
			if err != nil {
				log.Printf("failed to find symbol with name %s: %s", rt.CanonicalName, err)
				continue
			}
			ftable.ReturnType = append(ftable.ReturnType, typeinduction.ResolvedCandidate{
				Symbol:         sym,
				LogProbability: lp,
			})
		}
	}
	log.Printf("Done updating typeinduction.Client with skeleton function return types, skipped %d (of %d) skeleton functions\n", skipped, returns)
}

// UpdateArgSpecs with python skeletons.
// NOTE: this does NOT update the graph with new functions or types
func UpdateArgSpecs(graph *pythonimports.Graph, specs pythonimports.ArgSpecs) {
	log.Printf("Updating arg specs with skeleton function parameter types\n")
	var total, skipped int
	for _, fn := range index().Functions {
		if len(fn.Params) == 0 {
			continue
		}
		total++

		fnode := resolveFunction(fn.Path, graph)
		if fnode == nil {
			skipped++
			continue
		}

		as := specs.Find(fnode)
		if as == nil {
			as = &pythonimports.ArgSpec{
				NodeID: fnode.ID,
			}
			specs.ImportGraphArgSpecs[fnode.ID] = as
		}

		// overwrite old arguments
		// TODO(juan): combine?
		var args []pythonimports.Arg
		for _, param := range fn.Params {
			args = append(args, pythonimports.Arg{
				Name:        param.Name,
				DefaultType: param.Default,
				Types:       param.Types,
			})
		}

		as.Args = args

		// update init and type arg specs if neccesary
		if fnode.Classification == pythonimports.Type {
			if init, ok := fnode.Members["__init__"]; ok && init != nil {
				specs.ImportGraphArgSpecs[init.ID] = as
			}
		} else if fnode.Classification == pythonimports.Function {
			if fn.Path.Last() == "__init__" {
				typ := strings.Join(fn.Path.Parts[:len(fn.Path.Parts)-1], ".")
				if tnode := resolveType(typ, graph); tnode != nil {
					specs.ImportGraphArgSpecs[tnode.ID] = as
				}
			}
		}
	}
	fmt.Printf("Done updating arg specs with skeleton function return types, skipped %d (of %d) skeleton functions\n", skipped, total)
}

func isFinite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

func assertFinite(x float64, fmt string, args ...interface{}) {
	if !isFinite(x) {
		log.Fatalf(fmt, args...)
	}
}

func resolveFunction(f pythonimports.DottedPath, graph *pythonimports.Graph) *pythonimports.Node {
	fnode, err := graph.Navigate(f)
	if fnode == nil || err != nil {
		log.Printf("no node found for skeleton function `%s`, skipping\n", f)
	} else if fnode.CanonicalName.Empty() {
		log.Printf("no CanonicalName for skeleton function `%s`, skipping\n", f)
		fnode = nil
	}
	return fnode
}

func resolveType(t string, graph *pythonimports.Graph) *pythonimports.Node {
	// remove .type suffix if present
	if pos := strings.Index(t, ".type"); pos > -1 {
		t = t[:pos]
	}

	tn, err := graph.Find(t)
	if tn != nil && err == nil {
		return tn
	}

	// builtins not included for builtin types
	tn, err = graph.Find("builtins." + t)
	if tn != nil && err == nil {
		return tn
	}

	// types not included for members of types package
	tn, err = graph.Find("types." + t)
	if tn != nil && err == nil {
		return tn
	}

	if t == "generator" {
		tn, err = graph.Find("types.GeneratorType")
		if err != nil || tn == nil {
			panic("unable to find types.GeneratorType in graph!")
		}
		return tn
	}

	if t == "function" {
		tn, err = graph.Find("types.FunctionType")
		if err != nil || tn == nil {
			panic("unable to find types.FunctionType in graph!")
		}
		return tn
	}

	return nil
}
