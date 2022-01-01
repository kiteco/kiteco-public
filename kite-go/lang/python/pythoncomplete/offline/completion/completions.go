package completion

import (
	"context"
	"time"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/example"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/legacy"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Provider contains everything needed to get completions for an example
type Provider struct {
	api api.API
	rm  pythonresource.Manager
}

// NewProvider instanciate a completion provider from a resource manager
func NewProvider(rm pythonresource.Manager) *Provider {
	models, err := pythonmodels.New(pythonmodels.DefaultOptions)
	maybeQuit(err)
	return &Provider{
		api: api.New(context.Background(), api.Options{
			ResourceManager: rm,
			Models:          models,
		}, licensing.Pro),
	}
}

// GetNRCompletions generates all completion for the given example
// The buffer will be truncated at the cursor position
// addParenthesis and addPlaceholder modifies the buffer by resp. adding closing parenthesis and a placeholder just
// after the cursor.
func (cp *Provider) GetNRCompletions(ex example.Example, addParenthesis bool, addPlaceholder bool) []data.NRCompletion {
	src := ex.Buffer[:ex.Cursor]

	filename := "/fakefile.py"
	cursor := int(ex.Cursor)

	var suffix string
	var endSelection int
	if addPlaceholder {
		suffix += "[placeholder]"
		endSelection += len(suffix)
	}
	if addParenthesis {
		suffix += ")"
	}

	req := data.APIRequest{
		UMF: data.UMF{Filename: filename},
		SelectedBuffer: data.SelectedBuffer{
			Buffer: data.NewBuffer(src + suffix),
			Selection: data.Selection{
				Begin: cursor,
				End:   cursor + endSelection,
			},
		},
	}

	opts := api.IDCCCompleteOptions
	opts.BlockDebug = true

	var response data.APIResponse
	err := kitectx.Background().WithTimeout(3000*time.Second, func(ctx kitectx.Context) error {
		response = cp.api.Complete(ctx, opts, req, nil, nil)
		return response.ToError()
	})
	maybeQuit(err)
	return response.Completions
}

// GetCompletions return all the NRCompletion after having them converted to example.Completion
func (cp *Provider) GetCompletions(ex example.Example) []example.Completion {
	completions := cp.GetNRCompletions(ex, false, false)
	compGroups := make(legacy.SignatureMap)
	groupNRCompletions(completions, compGroups)
	result := make([]example.Completion, 0)
	completionMap(completions, func(completion *data.RCompletion) {
		result = append(result, getExampleCompletion(completion, len(result), compGroups))
	})
	return result
}

func getExampleCompletion(completion *data.RCompletion, rank int, compGroups legacy.SignatureMap) example.Completion {
	shouldKeep := true
	mc := completion.Debug.(legacy.MixCompletion)
	if compGroups != nil {
		shouldKeep = legacy.CompletionNotInSigs(mc, compGroups)
	}
	return example.Completion{
		Identifier:    completion.Snippet.Text,
		Score:         float64(-rank),
		MixCompletion: mc,
		Duplicate:     !shouldKeep,
	}
}
