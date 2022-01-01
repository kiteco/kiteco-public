package traindata

// SymbolDistEntry is an element in a symbol distribution
type SymbolDistEntry struct {
	Symbol       string  `json:"symbol"`
	Canonicalize bool    `json:"canonicalize"`
	Weight       float64 `json:"weight"`
}

// SymbolDist maps a symbol to a non negative weight.
// Notes:
// - Weigts must be non negative
// - Weigts do not need to sum to one, the weigths describe the relative probability of drawing a symbol
//   from the distribution, if one symbol s has a weight that is twice another symbol s' weigth then s
//   is twice as likely to be drawn as s'.
type SymbolDist map[string]*SymbolDistEntry
