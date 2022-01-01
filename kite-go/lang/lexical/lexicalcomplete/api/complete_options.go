package api

import (
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/driver"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

// CompleteOptions bundles options for API.Complete
type CompleteOptions = driver.Options

const maxReturnedCompletions = 2

// DefaultLexicalOptions provides lexical completions
var DefaultLexicalOptions = CompleteOptions{
	MixOptions: driver.MixOptions{
		MaxReturnedCompletions: maxReturnedCompletions,
		NestCompletions:        false,
		NoExactMatches:         true,
	},
	BlockTimeout: 200 * time.Millisecond,
}

// NewCompleteOptions returns CompleteOptions based on the APIOptions
func NewCompleteOptions(o data.APIOptions, l lang.Language) CompleteOptions {
	var opts CompleteOptions
	opts.APIOptions = o

	if opts.MixOptions.MaxReturnedCompletions == 0 {
		opts.MixOptions.MaxReturnedCompletions = maxReturnedCompletions
	}

	// exact matches are always disabled for all editors,
	// per: https://kite.quip.com/1ovKAL1hi2JZ/Spec-Lexical-Completions-Private-Beta
	opts.MixOptions.NoExactMatches = true

	// nesting is always disabled for all editors,
	// per: https://kite.quip.com/1ovKAL1hi2JZ/Spec-Lexical-Completions-Private-Beta#eLcACAqzrwL
	opts.MixOptions.NestCompletions = false

	// multiline completions are enabled for languages by default
	opts.MixOptions.AllowCompletionsWithNewlines = true

	switch o.Editor {
	case data.VSCodeEditor:
		if l != lang.CSS && l != lang.HTML {
			opts.MixOptions.PrependCompletionContext = true
		}
		opts.MixOptions.NoDollarSignDotCompletions = true
	case data.SublimeEditor:
		opts.MixOptions.NoDollarSignCompletions = true
		opts.MixOptions.AllowCompletionsWithNewlines = false
	case data.VimEditor:
		opts.MixOptions.AllowCompletionsWithNewlines = false
	case data.IntelliJEditor:
		opts.MixOptions.NoSmartStarInHint = true
	case data.SpyderEditor:
		opts.MixOptions.NoSmartStarInHint = true
		opts.MixOptions.SmartStarInDisplay = true
	}

	return opts
}
