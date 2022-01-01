package pythonresource

import "github.com/kiteco/kiteco/kite-golib/errors"

// RangeSemiCanonicalSymbols execute symbolProcessor on every canonical symbols and all their children
// Returning false after processing a symbol stop the iteration
// The error list returned contains all the errors received while listing the distribution, the canonical symbol
// and transforming the child symbol strings into real symbol
func (rm *manager) RangeSemiCanonicalSymbols(symbolProcessor func(symbol Symbol) bool) error {
	var errorList errors.Errors

mainloop:
	for _, dist := range rm.Distributions() {
		symbols, err := rm.CanonicalSymbols(dist)
		errorList = errors.Append(errorList, err)
		for _, cSym := range symbols {

			children, err := rm.Children(cSym)
			errorList = errors.Append(errorList, err)
			for _, ssym := range children {
				sym, err := rm.ChildSymbol(cSym, ssym)
				errorList = errors.Append(errorList, err)
				cont := symbolProcessor(sym)
				if !cont {
					break mainloop
				}
			}
		}
	}

	return errorList
}
