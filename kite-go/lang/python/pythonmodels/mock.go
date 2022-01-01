package pythonmodels

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonkeyword"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/callprob"
)

// Mock returns empty Models.
func Mock() *Models {
	return &Models{
		Keyword:      &pythonkeyword.Model{},
		Expr:         &pythonexpr.ModelShard{},
		FullCallProb: &callprob.Model{},
	}
}
