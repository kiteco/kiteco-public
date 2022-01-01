package api

import (
	"runtime"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/driver"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

// CompleteOptions bundles options for API.Complete
type CompleteOptions = driver.Options

const (
	maxReturnedCompletions = 20
	maxLexicalCompletions  = 2
)

// IDCCCompleteOptions provides IDCC completions
var IDCCCompleteOptions = CompleteOptions{
	MixOptions: driver.MixOptions{
		MaxReturnedCompletions: maxReturnedCompletions,
		MaxLexicalCompletions:  maxLexicalCompletions,
	},
}

// LegacyCompleteOptions provides legacy-API-compatible completions
var LegacyCompleteOptions = CompleteOptions{
	MixOptions: driver.MixOptions{
		MaxReturnedCompletions: maxReturnedCompletions,
		NoEmptyCalls:           true,
		NoElision:              true,
		NoExactMatch:           true,
	},
}

// NewCompleteOptions returns CompleteOptions based on the APIOptions
func NewCompleteOptions(o data.APIOptions) CompleteOptions {
	var opts CompleteOptions
	opts.APIOptions = o

	opts.MixOptions.MaxReturnedCompletions = maxReturnedCompletions
	opts.MixOptions.MaxLexicalCompletions = maxLexicalCompletions
	opts.BlockTimeoutLexical = 200 * time.Millisecond

	opts.MixOptions.AllowCompletionsWithNewlines = true
	opts.MixOptions.NoExactMatch = true

	switch o.Editor {
	case data.SublimeEditor:
		// because it screws with ordering
		opts.NoElision = true
		opts.MixOptions.AllowCompletionsWithNewlines = false
	case data.VimEditor:
		if runtime.GOOS == "windows" {
			opts.NoUnicode = true
		}
		opts.MixOptions.AllowCompletionsWithNewlines = false
	case data.VSCodeEditor:
		opts.NoAttributeToSubscript = true
		opts.MixOptions.PrependCompletionContext = true
	case data.SpyderEditor:
		opts.MixOptions.NoSmartStarInHint = true
		opts.MixOptions.SmartStarInDisplay = true
	}

	return opts
}
