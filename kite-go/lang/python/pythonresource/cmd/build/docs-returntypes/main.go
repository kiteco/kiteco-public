package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext"
	epyast "github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/epytext/ast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/numpydoc"
	npast "github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/numpydoc/ast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/helpers/rettypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/cmd/build/rawgraph/types"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/returntypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/spf13/cobra"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

// - extractor state

type extractor struct {
	dat types.ExplorationData

	index map[string]string

	numFunctions int
	numExtracted int
	unresolvable map[string]int
}

func newExtractor(dat types.ExplorationData) *extractor {
	x := &extractor{
		dat:          dat,
		unresolvable: make(map[string]int),
		index:        make(map[string]string),
	}

	// build index of <last component> -> <full path>
	// if we insert "" as a full path, that indicates a collision
	for _, g := range x.dat.TopLevels {
		for _, node := range g {
			if node.Reference == "" && node.Classification != "type" {
				continue
			}
			// for reference nodes, we cannot check the kind here, so we just hope that it's a type for now
			// we validate in the returntypes builder
			path := node.CanonicalName
			if node.Reference != "" {
				path = node.Reference
			}
			ident := path
			if idx := strings.LastIndex(path, "."); idx > -1 {
				ident = ident[idx+1:]
			}

			if oldPath, ok := x.index[ident]; ok && path != oldPath {
				// multiple nodes with same ident
				x.index[ident] = ""
				continue
			}

			x.index[ident] = path
		}
	}

	return x
}

func (x *extractor) extractAll() returntypes.Entities {
	all := make(returntypes.Entities)

	for _, g := range x.dat.TopLevels {
		for _, node := range g {
			if node.Reference != "" || node.Classification != "function" {
				// skip non-functions and external references
				continue
			}
			x.numFunctions++

			if strings.TrimSpace(node.Docstring) == "" {
				continue
			}

			ent := make(returntypes.Entity)
			x.extractNumpydoc(node, ent)
			x.extractEpytext(node, ent)
			if len(ent) == 0 {
				continue
			}

			x.numExtracted++
			log.Printf("[INFO] found return types %v for %s", ent, node)

			all[uint64(pythonimports.PathHash([]byte(node.CanonicalName)))] = ent
		}
	}
	return all
}

// - format-specific extraction

// resolve finds the path of the node for which the last component matches the provided ident.
// if it finds multiple matches, it returns "".
func (x *extractor) resolve(t keytypes.Truthiness, ident string) string {
	// handle builtin types explicitly: https://docs.python.org/3/library/stdtypes.html
	switch ident {
	case "bool",
		"int", "float", "complex",
		"list", "tuple",
		"str",
		"bytes", "bytearray", "memoryview",
		"set", "frozenset",
		"dict":
		return fmt.Sprintf("builtins.%s", ident)
	case "None":
		return "builtins.None.__class__"
	// and aliases
	case "string":
		return "builtins.str"
	case "dictionary":
		return "builtins.dict"
	case "object", "type":
		// don't resolve these because they're too generic to be useful
		return ""
	}

	// TODO(naman) handle dotted paths in a better way; for now we just search for the last component
	var query string
	for _, part := range strings.Split(ident, ".") {
		// if any part of the dotted path isn't a valid ident, abort
		if !pythonscanner.IsValidIdent(part) {
			return ""
		}
		query = part
	}

	path, ok := x.index[query]
	if path == "" && ok {
		log.Printf("[WARN] %s: too many types matching type query %s (%s)", t, query, ident)
	}
	return path
}

var punctuationSpace = regexp.MustCompile(`([^\w.]| )+`)

func (x *extractor) guessTypesFromText(t keytypes.Truthiness, text string, out returntypes.Entity) {
	// TODO(naman) inflections (e.g. plural)?
	for _, candidate := range punctuationSpace.Split(text, -1) {
		if candidate == "or" {
			// "or" indicates a logical disjunction
			continue
		}

		path := x.resolve(t, candidate)
		if path == "" {
			x.unresolvable[candidate]++
			continue
		}
		out[path] |= t
	}
}

func (x *extractor) extractNumpydoc(node *types.NodeData, out returntypes.Entity) {
	doc, err := numpydoc.Parse([]byte(node.Docstring))
	if err != nil {
		return
	}

	var tuple bool
	var texts []string
	for _, n := range doc.Content {
		section, ok := n.(*npast.Section)
		if !ok {
			continue
		}
		// find the returns section
		if strings.ToLower(strings.TrimSpace(section.Header)) != "returns" {
			continue
		}

		for _, n := range section.Content {
			def := n.(*npast.Definition)
			if len(def.Type) == 0 { // len(def.Subject) == 1
				// this is a bug in our numpydoc parser, where we store the type as the "subject" if there's no "subject"
				// according to the spec, Returns sections must have types and subjects are optional,
				// while Parameters sections must have subjects and types are optional
				texts = append(texts, string(def.Subject[0].(npast.Text)))
			} else { // len(def.Type) == 1 && len(def.Subject) == 1
				if strings.Contains(string(def.Subject[0].(npast.Text)), ",") {
					// multiple return values
					tuple = true
				}
				texts = append(texts, string(def.Type[0].(npast.Text)))
			}
		}

		if len(texts) == 0 {
			log.Printf("[ERROR] numpydoc: found no types in returns section for %s", node)
		}
	}

	if len(texts) > 0 {
		log.Printf("[INFO] numpydoc: found candidate type information for %s", node)
	}

	if tuple || len(texts) > 1 {
		log.Printf("[DEBUG] numpydoc: adding tuple type due to multiple return values for %s", node)
		// add this explicitly because `builtins.tuple` may or may not resolve in the current graph
		out["builtins.tuple"] |= keytypes.NumpydocTruthiness
	}
	for _, text := range texts {
		x.guessTypesFromText(keytypes.NumpydocTruthiness, text, out)
	}
}

type epytextVisitor struct {
	texts []string
}

func (v *epytextVisitor) Visit(n epyast.Node) epyast.Visitor {
	switch n := n.(type) {
	case epyast.LeafNode:
		v.texts = append(v.texts, n.Text())
	case *epyast.CrossRefMarkup:
		v.texts = append(v.texts, n.Object)
		return nil // no need to walk inside CrossRefMarkup, since Object contains the relevant info
	}
	return v
}

func (x *extractor) extractEpytext(node *types.NodeData, out returntypes.Entity) {
	block, err := epytext.Parse([]byte(node.Docstring))
	if err != nil {
		return
	}
	if block == nil {
		log.Printf("[ERROR] epytext: parsed nil AST with no error for %s", node)
		return
	}

	v := &epytextVisitor{}
	for _, n := range block.Nodes {
		field, ok := n.(*epyast.FieldBlock)
		if !ok {
			continue
		}
		switch field.Name {
		case "return", "rtype":
		default:
			continue
		}

		epyast.Walk(v, field)

		if len(v.texts) == 0 {
			log.Printf("[ERROR] epytext: found no types in returns for %s", node)
		}
	}

	if len(v.texts) > 0 {
		log.Printf("[INFO] epytext: found candidate type information for %s", node)
	}
	for _, text := range v.texts {
		x.guessTypesFromText(keytypes.EpytextTruthiness, text, out)
	}
}

// -

func build(cmd *cobra.Command, args []string) {
	outfile := args[0]

	out, err := os.Create(outfile)
	fail(err)
	defer out.Close()

	var all []rettypes.DistReturns
	for _, fname := range args[1:] {
		dat := func() types.ExplorationData {
			r, err := os.Open(fname)
			fail(err)
			defer r.Close()

			var dat types.ExplorationData
			fail(dat.Decode(r))
			return dat
		}()

		x := newExtractor(dat)
		returns := x.extractAll()
		if len(returns) > 0 {
			dist := keytypes.Distribution{Name: dat.PipPackage, Version: dat.PipVersion}
			all = append(all, rettypes.DistReturns{Dist: dist, Returns: returns})
			log.Printf("[STATS] %s: extracted return types for %d of %d functions", dist, x.numExtracted, x.numFunctions)
		}
	}

	fail(rettypes.EncodeAll(out, all))
}

var cmd = cobra.Command{
	Use:   "docs-returntypes DST.JSON INPUT_GRAPH ...",
	Short: "generate argspec resource data from validated raw graphs",
	Args:  cobra.MinimumNArgs(2),
	Run:   build,
}

func main() {
	fail(cmd.Execute())
}
