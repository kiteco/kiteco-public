package pigeon

import "github.com/kiteco/kiteco/kite-go/lang/javascript/ast"

func nilOrEmpty(i interface{}) bool {
	if i == nil {
		return true
	}

	if iSlice, ok := i.([]interface{}); ok && len(iSlice) == 0 {
		return true
	}
	return false
}

func terminal(n interface{}, idx int) ast.Terminal {
	if nn, ok := n.([]interface{}); ok {
		if idx < len(nn) {
			if chars, ok := nn[idx].([]uint8); ok {
				switch string(chars) {
				case ast.Colon:
					return ast.Colon
				case ast.Equals:
					return ast.Equals
				}
			}
		}
	}
	return ast.Unknown
}
