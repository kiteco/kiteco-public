package pythonresource

import (
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode/symbolcounts"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/docs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/kwargs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/internal/resources/sigstats"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var badCanonicalisation = map[string]bool{
	"builtins.object.__init__":  true,
	"builtin__.object.__init__": true,
}

// ArgSpec returns the argspec for the symbol if available
func (rm *manager) ArgSpec(sym Symbol) *pythonimports.ArgSpec {
	if _, isBadCan := badCanonicalisation[sym.Canonical().PathString()]; isBadCan {
		return rm.argSpecNonCanonical(sym)
	}
	return rm.argSpecNonCanonical(sym.Canonical())
}

func (rm *manager) argSpecNonCanonical(sym Symbol) *pythonimports.ArgSpec {
	rg := rm.resourceGroup(sym.Dist())
	if rg == nil {
		return nil
	}

	if argspec, exists := rg.ArgSpec[sym.PathHash()]; exists {

		return &argspec
	}
	return nil
}

// PopularSignatures returns the filtered signature patterns for the symbol if available.
// All the low information/illegal signatures have been filtered out
func (rm *manager) PopularSignatures(sym Symbol) []*editorapi.Signature {
	sym = sym.Canonical()

	rg := rm.resourceGroup(sym.Dist())
	if rg == nil {
		return nil
	}

	if pats, exists := rg.PopularSignatures[sym.PathHash()]; exists {
		return pats.Cast()
	}
	return nil
}

// CumulativeNumArgsFrequency returns the frequency for the given symbol and at least argument number
// This is a cumulative frequency, so if numArgs == 1, it returns the number of calls with at least one argument
// 0 always return the total number of calls
// For a non cumulative frequency (ie number of  call with exactly k args see NumArgsFrequency)
func (rm *manager) CumulativeNumArgsFrequency(sym Symbol, numArgs int) (float64, bool) {
	sym = sym.Canonical()

	rg := rm.resourceGroup(sym.Dist())
	if rg == nil {
		return 0, false
	}

	stats, exists := rg.SignatureStats[sym.PathHash()]
	if !exists {
		return 0, false
	}

	if len(stats.Positional) < numArgs {
		return 0, false
	}

	if numArgs == 0 {
		return 1, true // stats.Count/stats.Count == 1, ie number of time the function has been called with at least 0 arg
	}
	return float64(stats.Positional[numArgs-1].Count) / float64(stats.Count), true // Positional is 0 indexed
	// So we need to check index 0 for the 1 arg.
}

// KeywordArgFrequency return the number of time a keyword arg has been used
// That's an absolute frequency, to have a percentage, divide the result by the total number of call to the function
func (rm *manager) KeywordArgFrequency(sym Symbol, arg string) (int, bool) {
	stats := rm.SigStats(sym)
	if stats == nil {
		return 0, false
	}

	argStat, ok := stats.ArgsByName[arg]
	if !ok {
		return 0, false
	}
	return argStat.Count, true
}

// NumArgsFrequency returns the non cumulative frequency for the given symbol and argument number
// ie it returns the number of calls of this symbol with this exact number of argument (and not at least numArgs arguments as S
func (rm *manager) NumArgsFrequency(sym Symbol, numArgs int) (float64, bool) {
	sym = sym.Canonical()

	rg := rm.resourceGroup(sym.Dist())
	if rg == nil {
		return 0, false
	}
	signatures, exists := rg.PopularSignatures[sym.PathHash()]
	if !exists || len(signatures) == 0 {

		return 0, false
	}

	var numArgsFreq float64
	for _, signature := range signatures {
		lenArgs := len(signature.Args)
		if signature.LanguageDetails.Python != nil {
			lenArgs += len(signature.LanguageDetails.Python.Kwargs)
		}
		if numArgs == lenArgs {
			numArgsFreq += signature.Frequency
		}

	}
	return numArgsFreq, true
}

// Documentation returns the documentation for the symbol if available
func (rm *manager) Documentation(sym Symbol) *docs.Entity {
	// TODO maybe we want to try the non-canonical symbol before falling back to canonical?
	sym = sym.Canonical()

	// check for toplevel
	tlDat := rm.topLevelData(sym)
	if tlDat != nil {
		return tlDat.Docs
	}

	rg := rm.resourceGroup(sym.Dist())
	if rg == nil {
		return nil
	}

	if doc, exists := rg.Documentation[sym.PathHash()]; exists {
		return &doc
	}
	return nil
}

// SymbolCounts returns the counts of a symbol obtained from a corpus
func (rm *manager) SymbolCounts(sym Symbol) *symbolcounts.Counts {
	// check for toplevel
	tlDat := rm.topLevelData(sym)
	if tlDat != nil {
		return tlDat.Counts
	}

	rg := rm.resourceGroup(sym.Dist())
	if rg == nil {
		return nil
	}
	if c, exists := rg.SymbolCounts[sym.PathString()]; exists {
		return &c
	}
	return nil
}

// KeywordArgs aliases kwargs.KeywordArgs
type KeywordArgs = kwargs.KeywordArgs

// Kwargs returns keyword arguments of a symbol
func (rm *manager) Kwargs(sym Symbol) *KeywordArgs {
	sym = sym.Canonical()

	rg := rm.resourceGroup(sym.Dist())
	if rg == nil {
		return nil
	}
	if kwargs, exists := rg.Kwargs[sym.PathHash()]; exists {
		return &kwargs
	}
	return nil
}

// TruthySymbol pairs a Symbol with a Truthiness
type TruthySymbol struct {
	Symbol     Symbol
	Truthiness keytypes.Truthiness
}

// TruthyReturnTypes returns a slice of TruthySymbols corresponding to the potential return types of sym
func (rm *manager) TruthyReturnTypes(sym Symbol) []TruthySymbol {
	sym = sym.Canonical()

	rg := rm.resourceGroup(sym.Dist())
	if rg == nil {
		return nil
	}

	var rets []TruthySymbol
	for pathStr, truth := range rg.ReturnTypes[uint64(sym.PathHash())] {
		syms, _ := rm.PathSymbols(kitectx.TODO(), pythonimports.NewDottedPath(pathStr))
		for _, sym := range syms {
			rets = append(rets, TruthySymbol{sym, truth})
		}
	}
	return rets
}

// ReturnTypes returns a slice of Symbols corresponding to the potential return types of sym
func (rm *manager) ReturnTypes(sym Symbol) []Symbol {
	sym = sym.Canonical()

	rg := rm.resourceGroup(sym.Dist())
	if rg == nil {
		return nil
	}

	var rets []Symbol
	for pathStr := range rg.ReturnTypes[uint64(sym.PathHash())] {
		syms, _ := rm.PathSymbols(kitectx.TODO(), pythonimports.NewDottedPath(pathStr))
		rets = append(rets, syms...)
	}
	return rets
}

// SigStats ...
type SigStats = sigstats.Entity

// SigStatArg ...
type SigStatArg = sigstats.ArgStat

// SigStatTypeInfo ...
type SigStatTypeInfo = sigstats.TypeInfo

// SigStats for the provided func
func (rm *manager) SigStats(sym Symbol) *SigStats {
	sym = sym.Canonical()

	rg := rm.resourceGroup(sym.Dist())
	if rg == nil {
		return nil
	}

	stats, exists := rg.SignatureStats[sym.PathHash()]
	if !exists {
		return nil
	}

	return &stats
}
