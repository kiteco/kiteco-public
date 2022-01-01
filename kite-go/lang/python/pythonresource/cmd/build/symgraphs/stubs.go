package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/helpers"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/distidx"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/manifest"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/symgraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/stringutil"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
)

var (
	manifestPath string
	distidxPath  string
	analyzedPath string
)

func translateKind(k pythontype.Kind) keytypes.Kind {
	switch k {
	case pythontype.ModuleKind:
		return keytypes.ModuleKind
	case pythontype.TypeKind:
		return keytypes.TypeKind
	case pythontype.FunctionKind:
		return keytypes.FunctionKind
	case pythontype.InstanceKind:
		return keytypes.ObjectKind
	case pythontype.DescriptorKind:
		return keytypes.DescriptorKind
	}
	return keytypes.NoneKind
}

func computeKind(val pythontype.Value) keytypes.Kind {
	// TODO(naman) handle properties specially?
	kinds := make(map[keytypes.Kind]struct{})
	for _, val := range pythontype.DisjunctsNoCtx(val) {
		// TODO(naman) this currently chooses an arbitrary disjunct; make this deterministic
		k := translateKind(val.Kind())
		if k != keytypes.NoneKind {
			kinds[k] = struct{}{}
		}
	}

	if len(kinds) == 1 {
		for k := range kinds {
			return k
		}
	} else if len(kinds) > 1 {
		log.Printf("[ERROR] too many kinds for value %v", map[string]interface{}{
			"val":   val,
			"kinds": kinds,
		})
	}

	return keytypes.NoneKind
}

func computePaths(dist keytypes.Distribution, val pythontype.Value) map[pythonimports.Hash]pythonimports.DottedPath {
	paths := make(map[pythonimports.Hash]pythonimports.DottedPath)
	disjuncts := pythontype.DisjunctsNoCtx(val)

	// if there are externals in the disjuncts, point the attribute at the canonical symbol for the external
	for _, val := range disjuncts {
		switch val := val.(type) {
		case pythontype.External:
			// explicitly check the external case so we can canonicalize
			sym := val.Symbol()
			if sym.Dist() == dist { // this may not be true; particularly for 2.7 vs 3.5
				path := sym.Canonical().Path()
				paths[path.Hash] = path
			}
		case pythontype.SourceValue:
			// skip SourceValues, since we deal with them in a separate pass
		default:
			if addr := val.Address(); !addr.Path.Empty() {
				// we're certainly not looking at a stub / source value, so just use the path
				log.Printf("[DEBUG] using address %s of val %v for path", addr, val)
				paths[addr.Path.Hash] = addr.Path
			}
		}
	}

	if len(paths) > 0 {
		return paths
	}

	for _, val := range disjuncts {
		switch val := val.(type) {
		case pythontype.SourceValue:
			if addr := val.Address(); !addr.Nil() {
				path, err := helpers.QualifiedPathAnalysis(addr)
				fail(err)
				paths[path.Hash] = path
			}
		case pythontype.PropertyInstance:
			for hash, path := range computePaths(dist, val.FGet) {
				paths[hash] = path
			}
		}
	}

	return paths
}

// isNavigable verifies precondition 1 of addFauxCanonical
// when doing analysis of arbitrary code, the computed path may include e.g. [lambda] as a part, and not be navigable
func isNavigable(analyzed *pythonenv.SourceTree, path pythonimports.DottedPath) bool {
	tl := path.Head()
	curVal := analyzed.ImportAbs("/", tl).Value
	if curVal == nil {
		return false
	}
	for _, part := range path.Parts[1:] {
		// update curVal
		attrRes, err := pythontype.AttrNoCtx(curVal, part)
		if err != nil || !attrRes.Found() {
			return false
		}
		curVal = attrRes.Value()
	}
	return true
}

// precondition 0: every prefix of path is either internally navigable (i.e. not an external reference) or not navigable
// precondition 1: path is navigable via Attr lookups in analyzed data
// postcondition: the canonical paths invariants are maintained (prefixes of canonical paths are canonical)
func addFauxCanonical(graph symgraph.Graph, analyzed *pythonenv.SourceTree, path pythonimports.DottedPath) int {
	tl := path.Head()
	nodes := graph[tl]

	curIdx := 0
	curVal := analyzed.ImportAbs("/", tl).Value
	curPath := pythonimports.NewPath(tl)

	for i, part := range path.Parts[1:] {
		// update curVal
		attrRes, err := pythontype.AttrNoCtx(curVal, part)
		if err != nil || !attrRes.Found() {
			panic(fmt.Sprintf("addFauxCanonical: precondition 1 violated %v", map[string]interface{}{
				"path":    path,
				"curPath": strings.Join(path.Parts[:i+1], "."),
				"part":    part,
				"curVal":  curVal,
			}))
		}
		curVal = attrRes.Value()

		partKey := stringutil.ToUint64(part)

		// update curIdx
		node := nodes[curIdx]
		if childRef, ok := node.Children[partKey]; ok {
			if childRef.External.Cast().Empty() {
				curIdx = int(childRef.Internal)
			} else {
				panic(fmt.Sprintf("addFauxCanonical: precondition 0 violated %v", map[string]interface{}{
					"path":           path,
					"curPath":        strings.Join(path.Parts[:i+1], "."),
					"part":           part,
					"curVal":         curVal,
					"type(childRef)": fmt.Sprintf("%T", childRef),
				}))
			}
		} else {
			curIdx = len(nodes)
			node.Children[partKey] = symgraph.NodeRef{Internal: curIdx}

			newNode := symgraph.Node{
				// curPath is currently the previous path
				Canonical: symgraph.CastDottedPath(curPath.WithTail(part)), // up to and including i+1, the current part
				Children:  make(symgraph.ChildMap),
				Kind:      symgraph.Kind(computeKind(curVal)),
				// TODO(naman) add types here; it is probably prudent to do this in a separate pass to avoid excessive recursion
			}
			nodes = append(nodes, newNode)
			graph[tl] = nodes
			log.Printf("[INFO] successfully added node %s", newNode.Canonical.Cast())
		}

		// update curPath
		curPath = nodes[curIdx].Canonical.Cast()
	}

	return curIdx
}

// returns nil for SourceInstance, SourceFunction
func getSourceTable(val pythontype.SourceValue) map[string]*pythontype.Symbol {
	switch val := val.(type) {
	case *pythontype.SourceModule:
		return val.Members.Table
	case *pythontype.SourcePackage:
		return val.DirEntries.Table
	case *pythontype.SourceClass:
		return val.Members.Table
	default:
		return nil
	}
}

func insertAnalyzed(dist keytypes.Distribution, graph symgraph.Graph, analyzed *pythonenv.SourceTree) bool {
	if analyzed == nil {
		return false
	}

	var inserted bool
	for tl := range graph {
		// a piece of work is an symgraph.Node along with a symbol table to merge into that node
		type work struct {
			name  pythonimports.DottedPath
			table map[string]*pythontype.Symbol
			node  symgraph.Node
		}

		tlSym := analyzed.ImportAbs("/", tl)
		if tlSym == nil {
			continue // toplevel not in analyzed
		}
		tlVal := tlSym.Value
		inserted = true

		tlPath, err := helpers.QualifiedPathAnalysis(tlVal.Address())
		fail(err)
		q := []work{work{
			name:  tlPath,
			table: getSourceTable(tlVal.(pythontype.SourceValue)),
			node:  graph[tl][0],
		}}
		// seenMap tracks all of the Members maps we've seen by just tracking the paths of the corresponding works
		seenMap := make(map[pythonimports.Hash]struct{})
		seenMap[tlPath.Hash] = struct{}{}

		for len(q) > 0 {
			node := q[0].node
			table := q[0].table
			name := q[0].name
			q = q[1:]

			for attr, sym := range table {
				if sym.Private {
					continue // ignore private symbols
				}
				val := sym.Value

				if strings.HasPrefix(attr, "__") {
					log.Printf("[INFO] skipping attribute %v", map[string]interface{}{
						"val":            val,
						"node.Canonical": node.Canonical,
						"attr":           attr,
					})
					continue
				}

				// compute the "desired" path for the value
				var desiredPath pythonimports.DottedPath
				if paths := computePaths(dist, val); len(paths) == 0 {
					// this could be several things:
					//  - if var == nil, it's probably `...` (Ellipsis), which propagation doesn't handle
					//    * or stuff like __reduce__ which is manually assigned to nil for classes
					//	- or it could be e.g. a tuple `int_types = (int, long)`
					desiredPath = name.WithTail(attr) // certainly navigable
					log.Printf("[INFO] could not look up path for value %v", map[string]interface{}{
						"val":            val,
						"node.Canonical": node.Canonical,
						"attr":           attr,
						"desiredPath":    desiredPath,
					})
				} else if len(paths) > 1 {
					// this typically happens if the same symbol is defined in multiple ways depending on a sys.version check:
					// https://github.com/python/typeshed/blob/4ca0a6/stdlib/2and3/zipfile.pyi#L17
					// https://github.com/python/typeshed/blob/4ca0a6/stdlib/2and3/sre_compile.pyi#L9
					log.Printf("[ERROR] too many paths for value %v", map[string]interface{}{
						"val":            val,
						"node.Canonical": node.Canonical,
						"attr":           attr,
						"paths":          paths,
					})
					continue
				} else {
					for _, p := range paths {
						desiredPath = p
						break
					}
				}

				attrKey := stringutil.ToUint64(attr)

				// is the path an external reference?
				if desiredPath.Head() != tl {
					prevChild, exists := node.Children[attrKey]
					switch {
					case !exists:
						node.Children[attrKey] = symgraph.NodeRef{External: symgraph.CastDottedPath(desiredPath)}
						log.Printf("[INFO]0 successfully added attribute %s to %s", attr, node.Canonical.Cast())
					case !prevChild.External.Cast().Empty():
						// TODO(naman) take the valid external; this is tricky, since the new external may be created later
						if prevChild.External.Hash != uint64(desiredPath.Hash) {
							log.Printf("[CRITICAL]0 stub symgraph.External collides with explored symgraph.External %v", map[string]interface{}{
								"val":         val,
								"desiredPath": desiredPath,
								"prevChild":   prevChild.External,
							})
						}
					default:
						log.Printf("[CRITICAL]1 stub symgraph.External collides with explored symgraph.Internal %v", map[string]interface{}{
							"val":                 val,
							"desiredPath":         desiredPath,
							"privChild Canonical": graph[tl][prevChild.Internal].Canonical,
						})
					}
					continue
				}

				if !isNavigable(analyzed, desiredPath) {
					log.Printf("[ERROR] desired canonical path for value is non-navigable %v", map[string]interface{}{
						"val":            val,
						"node.Canonical": node.Canonical,
						"attr":           attr,
						"desiredPath":    desiredPath,
					})
					continue
				}

				// compute new child NodeRef
				var newChild *symgraph.NodeRef
				var newChildPath pythonimports.DottedPath // for logging
				ref, err := graph.Lookup(desiredPath)
				switch err := err.(type) {
				case nil:
					if ref.TopLevel != desiredPath.Head() {
						panic("unexpected toplevel mismatch from graph Lookup")
					}
					newChild = &symgraph.NodeRef{Internal: ref.Internal}
					newChildPath = graph.Canonical(ref)
				case symgraph.AttributeNotFound:
				case symgraph.ExternalEncountered:
					extPath := err.WithRest()
					log.Printf("[WARN] got external %s from lookup of internal path %s", extPath, desiredPath)
					newChild = &symgraph.NodeRef{External: symgraph.CastDottedPath(extPath)}
					newChildPath = extPath
				default:
					log.Printf("[CRITICAL]4 got error from lookup of internal path %s: %s", desiredPath, err)
				}

				// internal reference
				prevChild, exists := node.Children[attrKey]
				switch {
				case !exists:
					if newChild == nil {
						newChild = &symgraph.NodeRef{Internal: addFauxCanonical(graph, analyzed, desiredPath)}
					}
					node.Children[attrKey] = *newChild
					log.Printf("[INFO]1 successfully added attribute %s to %s", attr, node.Canonical.Cast())
				case prevChild.External.Cast().Empty():
					if newChild == nil || !newChild.External.Cast().Empty() || newChild.Internal != prevChild.Internal {
						log.Printf("[CRITICAL]2 stub %T collides with explored symgraph.Internal %v", newChild, map[string]interface{}{
							"val":                 val,
							"desiredPath":         desiredPath,
							"newChild":            newChild,
							"newChildPath":        newChildPath,
							"prevChild Canonical": graph[tl][prevChild.Internal].Canonical,
						})
					}
				default:
					if uint64(newChildPath.Hash) != prevChild.External.Hash {
						// TODO(naman) in the collision case, take the valid external; this is tricky, since the new external may not exist yet
						log.Printf("[CRITICAL]3 stub %T collides with explored symgraph.External %v", newChild, map[string]interface{}{
							"val":          val,
							"desiredPath":  desiredPath,
							"newChild":     newChild,
							"newChildPath": newChildPath,
							"prevChild":    prevChild,
						})
					}
				}

				if newChild != nil && newChild.External.Cast().Empty() {
					val, ok := val.(pythontype.SourceValue)
					if !ok {
						continue
					}
					addr := val.Address()
					if addr.Nil() {
						continue
					}
					table := getSourceTable(val)
					if table == nil { // not a source module/package/class
						continue
					}

					path, err := helpers.QualifiedPathAnalysis(val.Address())
					fail(err)
					if _, seen := seenMap[path.Hash]; seen {
						continue
					}

					seenMap[path.Hash] = struct{}{}
					q = append(q, work{
						name:  path,
						table: table,
						node:  graph[tl][newChild.Internal],
					})
				}
			}
		}
	}
	return inserted
}

func loadAnalyzed() (map[string]*pythonenv.SourceTree, error) {
	if analyzedPath == "" {
		return nil, nil
	}
	if manifestPath == "" {
		return nil, errors.New("if --analyzed is provided, --graph is required")
	}
	if distidxPath == "" {
		return nil, errors.New("if --analyzed is provided, --distidx is required")
	}

	// Load the symbol graph
	// TODO(naman) optimize which distributions we load based on the distidx & analyzed data
	opts := pythonresource.DefaultOptions

	mF, err := os.Open(manifestPath)
	fail(err)
	opts.Manifest, err = manifest.New(mF)
	fail(err)
	mF.Close()

	opts.Manifest = opts.Manifest.SymbolOnly()

	dF, err := os.Open(distidxPath)
	fail(err)
	opts.DistIndex, err = distidx.New(dF)
	fail(err)
	dF.Close()

	rm, errc := pythonresource.NewManager(opts)
	fail(<-errc)

	analyzedMap, err := helpers.LoadAnalyzed(analyzedPath, rm)
	fail(err)

	return analyzedMap, nil
}

func init() {
	cmd.Flags().StringVarP(&manifestPath, "graph", "g", "", "manifest with previous (stubless) symgraph resources, for linking with analyzed data")
	cmd.Flags().StringVarP(&distidxPath, "distidx", "d", "", "distribution index path, for linking with analyzed data")
	cmd.Flags().StringVarP(&analyzedPath, "analyzed", "a", "", "analyzed data")
}
