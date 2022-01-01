package selectors

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonautocorrect/internal/api"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
)

// DefaultSelector for the package.
func DefaultSelector(proposals []api.Proposal) api.Proposal {
	for _, prop := range proposals {
		if prop.Token == pythonscanner.Colon && prop.Type == api.Insertion {
			return prop
		}
	}
	return api.Proposal{
		Type: api.None,
	}
}
