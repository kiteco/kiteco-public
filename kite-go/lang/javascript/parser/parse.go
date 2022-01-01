package parser

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/javascript/ast"
	"github.com/kiteco/kiteco/kite-go/lang/javascript/parser/internal/pigeon"
)

// errMaxSteps is the error for when the parser takes too many steps.
var errMaxSteps = errors.New("parser took too many steps")

// DefaultOptions for a parser
var DefaultOptions = Options{
	MaxSteps: 10000000,
}

// Options for a parser
type Options struct {
	ModuleName string
	MaxSteps   int
}

// Parse the source file with the provided module name
func Parse(src []byte, opts Options) (*ast.Node, error) {
	defer parseDuration.DeferRecord(time.Now())

	// TODO(juan): max steps!
	rawAST, err := pigeon.Parse(opts.ModuleName, src)
	if err != nil {
		if strings.Contains(err.Error(), errMaxSteps.Error()) {
			tooManyStepsRatio.Hit()
		} else {
			tooManyStepsRatio.Miss()
		}
		return nil, err
	}
	tooManyStepsRatio.Miss()

	n, ok := rawAST.(*pigeon.Node)
	if !ok {
		return nil, fmt.Errorf("unable to cast %T to *node", rawAST)
	}
	return translate(n, src)
}
