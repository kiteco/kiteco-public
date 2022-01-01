package pythonpipeline

import (
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
)

// Resolved wraps a pythonanalyzer.ResolvedAST
type Resolved struct {
	RAST  *pythonanalyzer.ResolvedAST
	Words []pythonscanner.Word
}

// SampleTag implements pipeline.Sample
func (Resolved) SampleTag() {}

// Parsed wraps a *pythonast.Module
type Parsed struct {
	Mod   *pythonast.Module
	Words []pythonscanner.Word
}

// SampleTag implements pipeline.Sample
func (Parsed) SampleTag() {}

// ParsedNonNil parses input sample.ByteSlice and returns nil if
// the parsed module is nil
func ParsedNonNil(opts pythonparser.Options, timeout time.Duration) transform.OneInOneOutFn {
	return func(s pipeline.Sample) pipeline.Sample {
		ast, words, err := Parse(opts, timeout, s.(sample.ByteSlice))
		// Parse, unlike pythonparser.ParseWords, returns an error iff the ast is nil
		if err != nil {
			return pipeline.CoerceError(err)
		}

		return Parsed{
			Mod:   ast,
			Words: words,
		}
	}
}

// Parse the provied sample.ByteSlice
func Parse(opts pythonparser.Options, timeout time.Duration, bs sample.ByteSlice) (*pythonast.Module, []pythonscanner.Word, error) {
	words, err := pythonscanner.Lex([]byte(bs), opts.ScanOptions)
	if err != nil {
		return nil, nil, err
	}

	var ast *pythonast.Module
	err = kitectx.Background().WithTimeout(timeout, func(ctx kitectx.Context) error {
		var err error
		ast, err = pythonparser.ParseWords(ctx, []byte(bs), words, opts)
		return err
	})
	if _, ok := err.(kitectx.ContextExpiredError); ok {
		return nil, nil, pipeline.WrapErrorAsError("context expired", err)
	}

	if ast == nil {
		return nil, nil, pipeline.WrapErrorAsError("parse failure", err)
	}

	return ast, words, nil
}

// ResolvedNonNil resolves an input sample.Parsed and returns nil
// if there was an error
func ResolvedNonNil(rm pythonresource.Manager, timeout time.Duration) transform.OneInOneOutFn {
	return func(s pipeline.Sample) pipeline.Sample {
		rast, err := Resolve(rm, timeout, s.(Parsed))
		if err != nil {
			return pipeline.CoerceError(err)
		}

		return Resolved{
			RAST:  rast,
			Words: s.(Parsed).Words,
		}
	}
}

// Resolve an input sample.Parsed
func Resolve(rm pythonresource.Manager, timeout time.Duration, p Parsed) (*pythonanalyzer.ResolvedAST, error) {
	var rast *pythonanalyzer.ResolvedAST
	err := kitectx.Background().WithTimeout(timeout, func(ctx kitectx.Context) error {
		var err error
		rast, err = pythonanalyzer.NewResolver(rm, pythonanalyzer.Options{
			Path: "/src.py",
		}).ResolveContext(ctx, p.Mod, false)
		return err
	})
	if _, ok := err.(kitectx.ContextExpiredError); ok {
		return nil, pipeline.WrapErrorAsError("context expired", err)
	}
	if err != nil {
		return nil, pipeline.WrapErrorAsError("resolve failure", err)
	}

	return rast, nil
}
