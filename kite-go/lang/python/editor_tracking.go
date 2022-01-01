package python

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
)

//
// ID tracking
//

func trackSymbolID(origID, finalID string, sb symbolBundle) {
	// Track the ratio of valid to invalid
	if finalID != "" {
		validSymbolRatio.Hit()
		return
	}
	validSymbolRatio.Miss()

	// Track the source
	if origID == "" {
		invalidSymbolSource.HitAndAdd("undefined")
		if sb.val == nil {
			invalidUndefinedSymbolValueKind.HitAndAdd("null value")
		} else {
			invalidUndefinedSymbolValueKind.HitAndAdd(sb.val.Kind().String())
		}
	} else if !pythonenv.IsLocator(origID) {
		invalidSymbolSource.HitAndAdd("import graph")
	} else {
		invalidSymbolSource.HitAndAdd("local code")
		if sb.val == nil {
			invalidLocalSymbolValueKind.HitAndAdd("null value")
		} else {
			invalidLocalSymbolValueKind.HitAndAdd(sb.val.Kind().String())
		}
	}
}
