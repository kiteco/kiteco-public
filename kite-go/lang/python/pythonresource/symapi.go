package pythonresource

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/symgraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/toplevel"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const maxRecursionDepth = 10 // maximum recursion depth for canonicalization

// recursionDepthError implements error
type recursionDepthError struct{}

// Error implements error
func (e recursionDepthError) Error() string {
	return "recursion depth limit (10) exceeded"
}

// DistLoadError is an error produced when the resource group for a distribution is not loaded
type DistLoadError keytypes.Distribution

// Error implements error
func (e DistLoadError) Error() string {
	return fmt.Sprintf("could not load resource group for distribution %s", keytypes.Distribution(e))
}

// Pkgs returns a list of all indexed top-level importable packages
func (rm *manager) Pkgs() []string {
	var pkgs []string
	for pkg := range rm.index {
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}

// DistsForPkg returns all distributions that expose pkg as a top-level importable name
func (rm *manager) DistsForPkg(pkg string) []keytypes.Distribution {
	return rm.index[pkg]
}

// Symbol wraps a keytypes.Symbol to track the canonical symbol and validation status
type Symbol struct {
	Symbol    keytypes.Symbol
	canonical keytypes.Symbol
	ref       symgraph.Ref
}

// Symbol:
// path.Hash (uint64) | number of path parts | []parts | dist-string
func writeSymbol(symbol keytypes.Symbol, buf *bytes.Buffer) error {
	// write hash
	err := binary.Write(buf, binary.BigEndian, symbol.Path.Hash)
	if err != nil {
		return err
	}

	// write number of parts
	var partCount = int64(len(symbol.Path.Parts))
	err = binary.Write(buf, binary.BigEndian, partCount)
	if err != nil {
		return err
	}

	// write parts
	for _, p := range symbol.Path.Parts {
		buf.WriteString(p)
		buf.WriteByte(byte('\n'))
	}

	// write dist
	buf.WriteString(symbol.Dist.String())
	buf.WriteByte(byte('\n'))
	return nil
}

func readSymbol(buf *bytes.Buffer) (keytypes.Symbol, error) {
	symbol := keytypes.Symbol{}

	// hash
	err := binary.Read(buf, binary.BigEndian, &symbol.Path.Hash)
	if err != nil {
		return symbol, err
	}

	// number of parts
	var partCount int64
	err = binary.Read(buf, binary.BigEndian, &partCount)
	if err != nil {
		return symbol, err
	}

	// parts
	if partCount == 0 {
		symbol.Path.Parts = nil
	} else {
		parts := make([]string, partCount)
		for i := 0; i < int(partCount); i++ {
			v, err := buf.ReadString('\n')
			if err != nil {
				return symbol, err
			}
			parts[i] = v[0 : len(v)-1]
		}
		symbol.Path.Parts = parts
	}

	// dist-string
	dist, err := buf.ReadString(byte('\n'))
	if err != nil {
		return symbol, err
	}
	symbol.Dist, err = keytypes.ParseDistribution(dist[0 : len(dist)-1])
	return symbol, err
}

func writeRef(ref symgraph.Ref, buf *bytes.Buffer) error {
	err := binary.Write(buf, binary.BigEndian, int64(ref.Internal))
	if err != nil {
		return err
	}

	buf.WriteString(ref.TopLevel + "\n")
	return nil
}

func readRef(buf *bytes.Buffer) (symgraph.Ref, error) {
	ref := symgraph.Ref{}
	var internalValue int64
	err := binary.Read(buf, binary.BigEndian, &internalValue)
	if err != nil {
		return ref, err
	}
	ref.Internal = int(internalValue)

	topLevel, err := buf.ReadString('\n')
	if err != nil {
		return ref, err
	}
	ref.TopLevel = topLevel[:len(topLevel)-1]
	return ref, err
}

// MarshalBinary implements gob encoding to be able to use this in the remote Python resourcemanager
func (s Symbol) MarshalBinary() ([]byte, error) {
	shortFormat := s.Symbol.Dist == s.canonical.Dist && s.Symbol.Path.Hash == s.canonical.Path.Hash

	var b bytes.Buffer
	if shortFormat {
		b.WriteByte(byte('a'))
		err := writeSymbol(s.Symbol, &b)
		if err != nil {
			return nil, err
		}
	} else {
		b.WriteByte(byte('b'))
		err := writeSymbol(s.Symbol, &b)
		if err != nil {
			return nil, err
		}

		err = writeSymbol(s.canonical, &b)
		if err != nil {
			return nil, err
		}
	}
	err := writeRef(s.ref, &b)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// UnmarshalBinary implements gob decoding for Symbol. Note: s has to be pointer-type to make this work
func (s *Symbol) UnmarshalBinary(data []byte) error {
	b := bytes.NewBuffer(data)
	markerByte, err := b.ReadByte()
	if err != nil {
		return err
	}

	typeMarker := int32(markerByte)
	if typeMarker == 'a' {
		sym, err := readSymbol(b)
		if err != nil {
			return err
		}
		s.Symbol = sym
		s.canonical = sym
	} else if typeMarker == 'b' {
		s.Symbol, err = readSymbol(b)
		if err != nil {
			return err
		}

		s.canonical, err = readSymbol(b)
		if err != nil {
			return err
		}
	} else {
		return errors.New("invalid symbol marker type")
	}

	ref, err := readRef(b)
	if err != nil {
		return err
	}
	s.ref = ref
	return nil
}

// String implements fmt.Stringer
func (s Symbol) String() string {
	if s.Symbol.Dist == s.canonical.Dist && s.Symbol.Path.Hash == s.canonical.Path.Hash {
		return s.canonical.String()
	}
	return fmt.Sprintf("Symbol(%s -> %s)", s.Symbol, s.canonical)
}

// Hash for the distribution and path
func (s Symbol) Hash() pythonimports.Hash {
	parts := strings.Join([]string{
		s.Symbol.Dist.Name,
		s.Symbol.Dist.Version,
		s.Symbol.Path.String(),
	}, ":")

	return pythonimports.Hash(spooky.Hash64([]byte(parts)))
}

// Less re-exposes the arbitrary but fixed ordering on the underlying keytypes.Symbol
func (s Symbol) Less(other Symbol) bool {
	return s.Symbol.Less(other.Symbol)
}

// Dist returns the symbol's distribution
func (s Symbol) Dist() keytypes.Distribution {
	return s.Symbol.Dist
}

// Nil tests whether the Symbol represents a "nil" (invalid) value
func (s Symbol) Nil() bool {
	return s.Symbol.Path.Empty() && s.Symbol.Dist == keytypes.Distribution{}
}

// Equals tests for equality with another symbol
func (s Symbol) Equals(other Symbol) bool {
	return s.Symbol.Dist == other.Symbol.Dist && s.Symbol.Path.Hash == other.Symbol.Path.Hash
}

// Path returns the symbol's path
func (s Symbol) Path() pythonimports.DottedPath {
	return s.Symbol.Path.Copy()
}

// PathHead returns the first component ("top level") of the symbol's path
func (s Symbol) PathHead() string {
	return s.Symbol.Path.Head()
}

// PathHash returns a hash of the symbol's path
func (s Symbol) PathHash() pythonimports.Hash {
	return s.Symbol.Path.Hash
}

// PathString returns the symbol's path as a string
func (s Symbol) PathString() string {
	return s.Symbol.Path.String()
}

// PathLast is a more efficient version of Path().Last()
func (s Symbol) PathLast() string {
	return s.Symbol.Path.Last()
}

// Canonical returns the canonical symbol
func (s Symbol) Canonical() Symbol {
	return Symbol{
		Symbol:    s.canonical,
		canonical: s.canonical,
		ref:       s.ref,
	}
}

// Distribution returns the distribution for the given symbol
func (s Symbol) Distribution() keytypes.Distribution {
	return s.Symbol.Dist
}

// PathSymbol returns the canonicalized Symbol from the "least" matching distribution for the given path
func (rm *manager) PathSymbol(path pythonimports.DottedPath) (Symbol, error) {
	syms, err := rm.PathSymbols(kitectx.TODO(), path)
	if err != nil {
		return Symbol{}, err
	}
	return syms[0], nil
}

// PathSymbols returns a slice of Symbols for each matching distributions (in distribution order) for the given path
// if the returned error is nil, the returned slice must be non-empty
func (rm *manager) PathSymbols(ctx kitectx.Context, path pythonimports.DottedPath) ([]Symbol, error) {
	var syms []Symbol
	err := ctx.WithCallLimit(maxRecursionDepth, func(ctx kitectx.CallContext) error {
		var err error
		syms, err = rm.pathSymbols(ctx, path)
		return err
	})
	return syms, err
}

// NewSymbol validates the given distribution and path, and returns a valid Symbol
func (rm *manager) NewSymbol(dist keytypes.Distribution, path pythonimports.DottedPath) (Symbol, error) {
	var sym Symbol
	err := kitectx.TODO().WithCallLimit(maxRecursionDepth, func(ctx kitectx.CallContext) error {
		var err error
		sym, err = rm.canonicalize(ctx, keytypes.Symbol{Dist: dist, Path: path})
		return err
	})
	return sym, err
}

// pathSymbols and canonicalize are mutually recursive functions that validate a symbol by following external
// references in the Symbol graph. The returned Symbol tracks both the queried symbol path as well as its
// canonicalization.

type noDistributionsMatchPathError pythonimports.DottedPath

func (e noDistributionsMatchPathError) Error() string {
	return fmt.Sprintf("no distributions match path %s", pythonimports.DottedPath(e))
}

type pathNotFoundError struct {
	cause error
	path  pythonimports.DottedPath
}

func (e pathNotFoundError) Error() string {
	return fmt.Sprintf("no distributions match path %s: %s", e.path, e.cause)
}

// pathSymbols validates a Symbol from a DottedPath, where the distribution is unknown;
// it returns a slice of matching symbols, ordered by the argument path's matching distributions
func (rm *manager) pathSymbols(ctx kitectx.CallContext, path pythonimports.DottedPath) ([]Symbol, error) {
	if ctx.AtCallLimit() {
		return nil, recursionDepthError{}
	}

	// get the matching distributions (in sorted order)
	dists := rm.DistsForPkg(path.Head())
	if len(dists) == 0 {
		return nil, noDistributionsMatchPathError(path)
	}

	// find all possible symbols that match
	var err error
	var syms []Symbol
	for _, dist := range dists {
		var sym Symbol
		if sym, err = rm.canonicalize(ctx, keytypes.Symbol{Dist: dist, Path: path}); err == nil {
			syms = append(syms, sym)
		}
	}

	// if there are none, return immediately
	if len(syms) == 0 {
		return nil, pathNotFoundError{cause: err, path: path}
	}

	return syms, nil
}

// canonicalize validates a Symbol from a Distribution & DottedPath
func (rm *manager) canonicalize(ctx kitectx.CallContext, sym keytypes.Symbol) (Symbol, error) {
	if ctx.AtCallLimit() {
		return Symbol{}, recursionDepthError{}
	}

	// if the resource group is inaccessible, we're screwed
	if !rm.resourceGroupLoadable(sym.Dist) {
		return Symbol{}, DistLoadError(sym.Dist)
	}

	// check if the toplevel actually exists in the provided package
	toplevel := sym.Path.Head()
	var found bool
	for _, dist := range rm.DistsForPkg(toplevel) {
		if dist == sym.Dist {
			found = true
			break
		}
	}
	if !found {
		return Symbol{}, symgraph.TopLevelNotFound(toplevel)
	}

	// if we're just looking for the toplevel, don't bother with the resource group since we know it exists
	if len(sym.Path.Parts) == 1 {
		return Symbol{
			Symbol:    sym,
			canonical: sym,
			ref:       symgraph.Ref{TopLevel: toplevel, Internal: 0},
		}, nil
	}

	// otherwise, we need the graph
	rg := rm.loadResourceGroup(sym.Dist, "canonicalize")
	if rg == nil {
		// this should theoretically never happen, since we check resourceGroupLoadable above
		return Symbol{}, DistLoadError(sym.Dist)
	}

	ref, err := rg.SymbolGraph.Lookup(sym.Path)
	if extErr, ok := err.(symgraph.ExternalEncountered); ok {
		// TODO(naman) should canonicalize return a slice of symbols? there are performance tradeoffs
		resSyms, err := rm.pathSymbols(ctx.Call(), extErr.WithRest())
		if err != nil {
			return Symbol{}, err
		}
		resSym := resSyms[0]
		// reset the `symbol` to what the input actually was
		resSym.Symbol = sym
		return resSym, nil
	} else if err != nil {
		return Symbol{}, err
	}

	// otherwise, we found a canonical path in the same distribution
	return Symbol{
		Symbol: sym,
		canonical: keytypes.Symbol{
			Dist: sym.Dist,
			Path: rg.SymbolGraph.Canonical(ref),
		},
		ref: ref,
	}, nil
}

// Kind returns the keytypes.Kind of a Symbol
func (rm *manager) Kind(s Symbol) keytypes.Kind {
	if rm.topLevelData(s) != nil {
		return keytypes.ModuleKind
	}

	rg := rm.resourceGroup(s.canonical.Dist)
	if rg == nil {
		return keytypes.NoneKind // TODO(naman)
	}

	return rg.SymbolGraph.Kind(s.ref)
}

// Type resolves the type Symbol for s
func (rm *manager) Type(s Symbol) (Symbol, error) {
	if rm.topLevelData(s) != nil {
		// top-levels don't currently have types in the graph
		return Symbol{}, errors.New("no types for top-levels")
	}

	rg := rm.resourceGroup(s.canonical.Dist)
	if rg == nil {
		return Symbol{}, DistLoadError(s.canonical.Dist)
	}

	ref, err := rg.SymbolGraph.Type(s.ref)
	if extErr, ok := err.(symgraph.ExternalEncountered); ok {
		// TODO(naman) we may want the non-canonical path here to be s.sym.Path.WithTail("__class__") if a __class__ attribute exists
		// in that case, we should actually update pkgexploration to not skip __class__ attributes
		return rm.PathSymbol(extErr.WithRest())
	} else if err != nil {
		return Symbol{}, err
	}

	// again, the non-canonical path may be better set to WithTail("__class__")
	sym := keytypes.Symbol{
		Dist: s.canonical.Dist,
		Path: rg.SymbolGraph.Canonical(ref),
	}
	return Symbol{
		Symbol:    sym,
		canonical: sym,
		ref:       ref,
	}, nil
}

// Bases resolves the base class Symbols for s, skipping any unresolvable base classes
func (rm *manager) Bases(s Symbol) []Symbol {
	if rm.topLevelData(s) != nil {
		// modules don't have base classes
		return nil
	}

	rg := rm.resourceGroup(s.canonical.Dist)
	if rg == nil {
		return nil // TODO(naman)
	}

	var bases []Symbol
	numBases := rg.SymbolGraph.NumBases(s.ref)
	for i := 0; i < numBases; i++ {
		ref, err := rg.SymbolGraph.GetBase(s.ref, i)
		if extErr, ok := err.(symgraph.ExternalEncountered); ok {
			base, err := rm.PathSymbol(extErr.WithRest())
			if err == nil {
				bases = append(bases, base)
			} else {
				// TODO(naman)
			}
			continue
		} else if err != nil {
			// TODO(naman)
			continue
		}

		sym := keytypes.Symbol{
			Dist: s.canonical.Dist,
			Path: rg.SymbolGraph.Canonical(ref),
		}
		bases = append(bases, Symbol{
			Symbol:    sym,
			canonical: sym,
			ref:       ref,
		})
	}

	return bases
}

// Children returns a list of strings identifying children of the given Symbol
func (rm *manager) Children(s Symbol) ([]string, error) {
	rg := rm.loadResourceGroup(s.canonical.Dist, "Children")
	if rg == nil {
		return nil, DistLoadError(s.canonical.Dist)
	}

	return rg.SymbolGraph.Children(s.ref), nil
}

// ChildSymbol computes a validated child Symbol specified by the string argument
func (rm *manager) ChildSymbol(s Symbol, c string) (Symbol, error) {
	rg := rm.loadResourceGroup(s.canonical.Dist, "ChildSymbol")
	if rg == nil {
		return Symbol{}, DistLoadError(s.canonical.Dist)
	}

	childSymbol := keytypes.Symbol{
		Dist: s.Symbol.Dist,
		Path: s.Symbol.Path.WithTail(c),
	}

	ref, err := rg.SymbolGraph.Child(s.ref, c)
	if extErr, ok := err.(symgraph.ExternalEncountered); ok {
		sym, err := rm.PathSymbol(extErr.WithRest())
		if err != nil {
			return Symbol{}, err
		}
		sym.Symbol = childSymbol
		return sym, nil
	} else if err != nil {
		return Symbol{}, err
	}

	return Symbol{
		Symbol: childSymbol,
		canonical: keytypes.Symbol{
			Dist: s.canonical.Dist,
			Path: rg.SymbolGraph.Canonical(ref),
		},
		ref: ref,
	}, nil
}

// -

// CanonicalSymbols returns a slice of canonical symbols for the given distribution
func (rm *manager) CanonicalSymbols(dist keytypes.Distribution) ([]Symbol, error) {
	rg := rm.loadResourceGroup(dist, "CanonicalSymbols")
	if rg == nil {
		return nil, DistLoadError(dist)
	}

	var out []Symbol
	for toplevel, nodes := range *rg.SymbolGraph {
		for i, n := range nodes {
			sym := keytypes.Symbol{
				Dist: dist,
				Path: n.Canonical.Cast(),
			}
			out = append(out, Symbol{
				Symbol:    sym,
				canonical: sym,
				ref: symgraph.Ref{
					TopLevel: toplevel,
					Internal: i,
				},
			})
		}
	}
	return out, nil
}

// TopLevels returns a slice of toplevel packages for the given distribution
func (rm *manager) TopLevels(dist keytypes.Distribution) ([]string, error) {
	rg := rm.resourceGroup(dist)
	if rg == nil {
		return nil, DistLoadError(dist)
	}

	var out []string
	for tl := range *rg.SymbolGraph {
		out = append(out, tl)
	}

	return out, nil
}

func (rm *manager) topLevelData(sym Symbol) *toplevel.Entity {
	if rm.toplevel == nil || len(sym.Symbol.Path.Parts) != 1 {
		return nil
	}
	res, ok := rm.toplevel[toplevel.DistributionTopLevel{
		Distribution: sym.Symbol.Dist,
		TopLevel:     sym.Symbol.Path.Head(),
	}]
	if !ok {
		return nil
	}
	return &res
}

// -

// MustInternalGraph is for internal (builder) use only; it may panic
func (rm *manager) MustInternalGraph(dist keytypes.Distribution) symgraph.Graph {
	rg := rm.resourceGroup(dist)
	if rg == nil {
		panic(DistLoadError(dist).Error())
	}
	return *rg.SymbolGraph
}
