package pythonstatic

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
)

func computeBoolUnary(x bool, op pythonscanner.Token) pythontype.Value {
	switch op {
	case pythonscanner.Not:
		return pythontype.BoolConstant(!x)
	case pythonscanner.Add:
		if x {
			return pythontype.IntConstant(1)
		}
		return pythontype.IntConstant(0)
	case pythonscanner.Sub:
		if x {
			return pythontype.IntConstant(-1)
		}
		return pythontype.IntConstant(0)
	case pythonscanner.BitNot:
		return pythontype.IntInstance{}
	default:
		return pythontype.BoolInstance{}
	}
}

func computeBoolBinary(x, y bool, op pythonscanner.Token) pythontype.Value {
	// TODO
	return pythontype.BoolInstance{}
}
