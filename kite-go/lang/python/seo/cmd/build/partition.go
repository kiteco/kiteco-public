package main

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

// disjoint-set data structure to allow for "fixing" issues with the symbol graph not unifying enough symbols as canonical

type partition map[keytypes.Distribution]map[pythonimports.Hash]pythonresource.Symbol

func newPartition(rm pythonresource.Manager) partition {
	p := make(partition)
	for _, dist := range rm.Distributions() {
		syms, err := rm.CanonicalSymbols(dist)
		fail(err)

		p[dist] = make(map[pythonimports.Hash]pythonresource.Symbol)
		for _, sym := range syms {
			p[dist][sym.PathHash()] = sym
		}
	}
	return p
}

func (p partition) canonicalize(sym pythonresource.Symbol) pythonresource.Symbol {
	sym = sym.Canonical()
	for {
		parent := p[sym.Dist()][sym.PathHash()]
		if parent.Equals(sym) {
			break
		}
		// path splitting
		p[sym.Dist()][sym.PathHash()] = p[parent.Dist()][parent.PathHash()]
		sym = parent
	}
	return sym
}

func (p partition) setCanonical(canon, sym pythonresource.Symbol) {
	canon = p.canonicalize(canon)
	sym = p.canonicalize(sym)
	p[sym.Dist()][sym.PathHash()] = canon
}

// checkInheritedDocs heuristically (using docs) checks for failure to identify that a method is inherited from a base class
// TODO(naman) this is not quite sound, since inspect.getdoc returns a docstring from the base class method,
// even if that method is overridden, as long as the overridde does not itself specify a docstring.
func (p partition) checkInheritedDocs(rm pythonresource.Manager) {
	// note that the same symbol may be handled multiple times, so we're not being very efficient
	for _, dist := range rm.Distributions() {
		syms, err := rm.CanonicalSymbols(dist)
		fail(err)
		for _, sym := range syms {
			symPath := sym.Path()
			classPath := symPath.Predecessor()
			if classPath.Empty() {
				continue
			}

			// the containing class
			pSym, err := rm.NewSymbol(sym.Dist(), classPath)
			if err != nil {
				continue
			}

			// search for an inherited (same name) symbol with the same docs in the base classes
			symDocs := realDocs(rm, sym)
			for _, baseSym := range rm.Bases(pSym) {
				inheritedSym, err := rm.ChildSymbol(baseSym, symPath.Last())
				if err != nil {
					continue
				}
				if symDocs != realDocs(rm, inheritedSym) {
					continue
				}

				// found match; update canonical
				p.setCanonical(inheritedSym, sym)
				break
			}
		}
	}
}

// TODO(naman) we may also want to do more deduping based on docstring collision,
// but this is a bit complicated

// The raw number of candidates for removal based on a naive deduper is ~50k.

// This technique seems to potentially yields many false positives: there is a risk of accidentally merging too much.
// One approach might be to only merge in cases where all the matching symbols are in the same distribution:
//
// With this filter there are ~24k remaining candidates for removal.

// However, even then, there is almost never a clear candidate for the "canonical" symbol.
// Symbol counts tend to all be approximately even, and the shortest path tends not to be (close to) unique:
// most of the colliding symbols are usually of approximately equal length.
//
// If we only look at cases where there's a clear shortest path, only ~2.5k candidates remain for removal.

// This needs more thought overall, but it's unclear whether it's worth the time investment.
