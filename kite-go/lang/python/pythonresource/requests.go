package pythonresource

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

// NumArgsFrequencyRequest is a helper to implement the remote resource manager
type NumArgsFrequencyRequest struct {
	Symbol  Symbol
	NumArgs int
}

// KeywordArgFrequencyRequest is a helper to implement the remote resource manager
type KeywordArgFrequencyRequest struct {
	Symbol Symbol
	Arg    string
}

// FloatBoolResponse is a helper to implement the remote resource manager
type FloatBoolResponse struct {
	Float float64
	Bool  bool
}

// IntBoolResponse is a helper to implement the remote resource manager
type IntBoolResponse struct {
	Int  int
	Bool bool
}

// NewSymbolRequest is a helper to implement the remote resource manager
type NewSymbolRequest struct {
	Dist keytypes.Distribution
	Path pythonimports.DottedPath
}

// ChildSymbolRequest is a helper to implement the remote resource manager
type ChildSymbolRequest struct {
	Symbol Symbol
	String string
}
