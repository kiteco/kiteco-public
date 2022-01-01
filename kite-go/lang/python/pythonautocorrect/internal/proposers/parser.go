package proposers

import (
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonautocorrect/internal/api"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

// ParserBased makes proposals based on parser errors
type ParserBased struct {
	parseErrs errors.Errors
}

// NewParserBased proposer using the specified parser error
func NewParserBased(parseErrs errors.Errors) ParserBased {
	return ParserBased{
		parseErrs: parseErrs,
	}
}

// Propose satisfies the api.Proposer interface
func (p ParserBased) Propose(words []pythonscanner.Word) []api.Proposal {
	if p.parseErrs == nil {
		return nil
	}

	for _, err := range p.parseErrs.Slice() {
		posErr := err.(pythonscanner.PosError)
		if strings.Contains(posErr.Msg, "expected :") {
			return []api.Proposal{
				api.Proposal{
					Type:  api.Insertion,
					Pos:   int(posErr.Pos),
					Token: pythonscanner.Colon,
				},
			}
		}
	}
	return nil
}
