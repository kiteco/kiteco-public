package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/jaytaylor/html2text"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-go/lang/python/seo"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

func main() {
	datadeps.Enable()
	log.SetFlags(0)
	log.SetPrefix("")

	flag.Parse()
	filename := flag.Arg(0)
	if filename == "" {
		log.Fatalln("usage: ./build output.gob.gz")
	}
	run(filename)
}

func run(filename string) {
	importGraph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	fail(err)
	curated, err := pythoncuration.NewSearcher(importGraph, &pythoncuration.DefaultSearchOptions)
	fail(err)
	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)

	data := build(rm, curated)

	f, err := os.Create(filename)
	fail(err)
	defer f.Close()

	fail(data.Encode(f))
}

func build(rm pythonresource.Manager, curated *pythoncuration.Searcher) seo.Data {
	p := newPartition(rm)
	p.checkInheritedDocs(rm)

	var total int
	data := make(seo.Data)
	for _, dist := range rm.Distributions() {
		tls, err := rm.TopLevels(dist)
		fail(err)

		// compute links
		out := make(map[pythonimports.Hash]pythonimports.DottedPath)
		for _, tl := range tls {
			fail(computeDisplay(rm, curated, p, dist, tl, out))
		}

		// only add it to the global map if it has anything left
		for k, path := range out {
			if path.Empty() {
				delete(out, k)
			}
		}
		log.Printf("[info] included %d symbols for %s", len(out), dist)
		total += len(out)
		if len(out) > 0 {
			data[dist] = out
		}
	}

	log.Printf("[info] included %d total symbols", total)

	return data
}

// computeDisplay computes the display paths for each canonical symbol rooted at the given toplevel.
//
// Display paths have the following properties:
//   1. The path canonicalizes to the corresponding canonical symbol.
//   2. The path is also rooted at the given top-level.
//   3. The path contains no private components (i.e. starting with "_")
//   4. There is no shorter path satisfying properties 1, 2, 3.
//   5. There is no more frequent path (according to Expr symbol counts) satisfying properties 1, 2, 3, 4.
// Any other ties are broken deterministically.
//
// If no path satisfies the above properties (e.g. if all paths to it have a private component), none is added to the out map.
func computeDisplay(rm pythonresource.Manager, curated *pythoncuration.Searcher, p partition,
	dist keytypes.Distribution, toplevel string, out map[pythonimports.Hash]pythonimports.DottedPath) error {

	if strings.HasPrefix(toplevel, "_") {
		return nil // private
	}

	rootSym, err := rm.NewSymbol(dist, pythonimports.NewPath(toplevel))
	if err != nil {
		return errors.WithStack(err)
	}

	// breadth-first search
	// each iteration computes a new frontier, with path length one greater than the previous
	frontier := []pythonresource.Symbol{rootSym}
	for {
		var next []pythonresource.Symbol

		// candidate display paths for newly seen canonical symbols
		newCandidates := make(map[pythonimports.Hash][]pythonresource.Symbol)
		for _, sym := range frontier {
			canon := p.canonicalize(sym)

			if canon.Dist() != dist || canon.PathHead() != toplevel {
				continue // not internal to the current toplevel
			}

			canonHash := canon.PathHash()
			if _, exists := out[canonHash]; exists {
				continue // we already have a path for this, and it must be shorter by construction
			}

			newCandidates[canonHash] = append(newCandidates[canonHash], sym)

			children, err := rm.Children(sym)
			fail(err)
			for _, child := range children {
				if strings.HasPrefix(child, "_") {
					continue // private
				}
				csym, err := rm.ChildSymbol(sym, child)
				if err != nil {
					continue // this is normal
				}
				next = append(next, csym)
			}
		}

		for h, choices := range newCandidates {
			if len(choices) == 0 {
				continue
			}

			if noIndex(rm, curated, choices[0]) {
				out[h] = pythonimports.DottedPath{}
				continue
			}

			topCount := -1
			var topChoice pythonresource.Symbol
			for _, choice := range choices {
				var count int
				if counts := rm.SymbolCounts(choice); counts != nil {
					count = counts.Expr
				}

				// take the most common; break ties via arbitrary (lexicographic) ordering
				if topCount < 0 || count > topCount || (count == topCount && choice.Less(topChoice)) {
					topCount = count
					topChoice = choice
				}
			}

			out[h] = topChoice.Path()
		}

		if len(newCandidates) == 0 {
			// no newly seen canonical symbols: we're done
			break
		}
		frontier = next
	}

	return nil
}

// filterNoIndex filters out symbols that should not be indexed due to missing docs and examples.
// We try to account for incorrect docs and/or examples (see also realDocs).
func noIndex(rm pythonresource.Manager, curated *pythoncuration.Searcher, sym pythonresource.Symbol) bool {
	sym = sym.Canonical()

	if len(realDocs(rm, sym)) > 20 {
		return false
	}

	// e.g. examples for curated.FRIDAY include one for dateutil.rrule.HOURLY, since both are the integer 4.
	// so early return false before the curated examples check if we're an instance of int
	if tySym, err := rm.Type(sym); err == nil {
		switch tySym.Canonical().PathString() {
		case "builtins.int":
			return true
		}
	}

	examples, found := curated.Canonical(sym.PathString())
	found = found && len(examples) > 0
	if found {
		return false
	}

	return true
}

// -

// realDocs attempts to correct for docstrings being "inherited" from types or base classes (as per inspect.getdoc).
// It is not complete, due to issues such as #8701 "failure to track base class hierarchy due to unindexed distributions."
func realDocs(rm pythonresource.Manager, sym pythonresource.Symbol) string {
	docs := rm.Documentation(sym)
	if docs == nil {
		return ""
	}
	if docs.Text == "" {
		text, err := html2text.FromString(docs.HTML)
		if err != nil {
			return text
		}
		return docs.HTML
	}

	// check if inherited docs from type of self
	if tySym, err := rm.Type(sym); err == nil {
		if tyDocs := rm.Documentation(tySym); tyDocs != nil && tyDocs.Text == docs.Text {
			return ""
		}
	}

	// check if inherited doc from base class of self
	for _, bSym := range rm.Bases(sym) {
		if bDocs := rm.Documentation(bSym); bDocs != nil && bDocs.Text == docs.Text {
			return ""
		}
	}

	return docs.Text
}

func deferErr(err *error, f func() error) {
	if e := f(); *err == nil {
		*err = e
	}
}

func fail(err error) {
	if err != nil {
		panic(err)
	}
}
