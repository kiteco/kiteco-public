package traindata

import (
	"fmt"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// WalkFn is the common callback for the Walker.
// Param:
// - the first argument is the name of the toplevel being explored
// - the second argument is the parent of the symbol currently being explored (Nil for top level symbols)
// - the third argument is the (child) symbol currently being explored
// - the fourth argument is true if the specified symbol is a top level symbol
// Return
// - an error if anything went wrong, at the first error the walker
//   stops and returns the error.
type WalkFn func(string, pythonresource.Symbol, pythonresource.Symbol, bool) error

// SkipFn returns true if a given symbol should be skipped.
type SkipFn func(pythonresource.Symbol) bool

// Walker walks symbols
type Walker struct {
	canonicalize bool
	seen         map[pythonimports.Hash]bool
	fn           WalkFn
	skip         SkipFn
	rm           pythonresource.Manager
}

// NewWalker using the specified walk fn, skip may be nil.
func NewWalker(rm pythonresource.Manager, canonicalize bool, walk WalkFn, skip SkipFn) *Walker {
	return &Walker{
		canonicalize: canonicalize,
		seen:         make(map[pythonimports.Hash]bool),
		fn:           walk,
		skip:         skip,
		rm:           rm,
	}
}

func (w *Walker) walk(tl string, sym pythonresource.Symbol) error {
	children, err := w.rm.Children(sym)
	if err != nil {
		return fmt.Errorf("error getting children of %s %v: %v", tl, sym, err)
	}
	sort.Strings(children)

	for _, c := range children {
		cs, err := w.rm.ChildSymbol(sym, c)
		if err != nil {
			// happens for symbols that are not walkable
			// like __bases__[%d] or non walkable class members
			continue
		}

		if w.canonicalize {
			cs = cs.Canonical()
		}

		if w.seen[cs.PathHash()] {
			// check for loops
			continue
		}

		if !cs.Path().HasPrefix(tl) {
			// skip external symbols
			continue
		}

		if w.skip != nil && w.skip(cs) {
			continue
		}

		// need to mark the path as visited once we have acutally
		// visited it, otherwise we may mark it as visited when it is actually
		// skipped because one of the above conditions was hit (e.g it was an external symbol)
		// and then later passes with a different top level will never visit it.
		w.seen[cs.PathHash()] = true

		if err := w.fn(tl, sym, cs, false); err != nil {
			return err
		}

		switch w.rm.Kind(cs) {
		case keytypes.ModuleKind, keytypes.TypeKind:
			if err := w.walk(tl, cs); err != nil {
				return err
			}
		}
	}

	return nil
}

// Walk the specified top level.
// TODO: might be easier to reason about if we just maintain a one to one mapping
// between top level and walker, so each top level requires a new walker, and then the caller is in charge of keeping things unique if
// needed.
func (w *Walker) Walk(tl string) error {
	syms, err := w.rm.PathSymbols(kitectx.Background(), pythonimports.NewDottedPath(tl))
	if err != nil {
		return fmt.Errorf("error translating top level %s to symbols: %v", tl, err)
	}

	// for now just choose the first symbol
	// TODO: support multiple symbols for the same top level
	sym := syms[0]
	if w.canonicalize {
		sym = sym.Canonical()
	}

	w.seen[sym.PathHash()] = true

	w.fn(tl, pythonresource.Symbol{}, sym, true)

	if err := w.walk(tl, sym); err != nil {
		return fmt.Errorf("error walking %s %v: %v", tl, sym, err)
	}

	return nil
}
