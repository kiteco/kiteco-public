package pythonmodels

import (
	"context"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonkeyword"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/callprob"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/callprobcallmodel"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/mtacconf"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

// Options defines the options used for loading models.
// this struct is serialized to JSON as part of kite_status metrics
type Options struct {
	Local                     bool `json:"-"`
	KeywordModelPath          string
	CallCompletionModelPath   string
	AttributeModelPath        string
	ExprModelShards           []pythonexpr.Shard
	ExprModelOpts             pythonexpr.Options
	CallProbSubtokenModelPath string
	CallProbCallModelPath     string
	MTACConfModelPath         string
}

// DefaultOptions provides defaults for the model options.
var DefaultOptions = Options{
	Local:                     true,
	ExprModelOpts:             pythonexpr.DefaultOptions,
	ExprModelShards:           pythonexpr.ExprModelShards(),
	KeywordModelPath:          "s3://kite-data/keyword-model/2019-06-19/keyword_model.frozen.pb",
	MTACConfModelPath:         "s3://kite-data/python-mtac-confidence/2019-05-03_08-02-59-PM/serve",
	CallProbSubtokenModelPath: "s3://kite-data/python-call-prob/2020-02-13_06-26-58-PM/serve",
	CallProbCallModelPath:     "s3://kite-data/python-call-prob/2019-11-12_06-22-20-PM/serve",

	// not currently used, but preserving this path for the near future (as of 2019-06-03)
	// MixModelPath: "s3://kite-data/python-completion-mixing/2019-01-25_09-06-42-PM/serve/mix_model.frozen.pb",
}

// Models is a wrapper around models used for completions.
type Models struct {
	Keyword           *pythonkeyword.Model
	Expr              pythonexpr.Model
	PartialCallProb   *callprob.Model
	FullCallProb      *callprob.Model
	MTACConf          *mtacconf.Model
	CallModelCallProb *callprobcallmodel.Model
}

// New loads the necessary models with a context.Background.
func New(opts Options) (*Models, error) {
	return NewWithCtx(context.Background(), opts)
}

// NewWithCtx loads the necessary models with a parent context.
func NewWithCtx(ctx context.Context, opts Options) (*Models, error) {
	kwModel, err := pythonkeyword.NewModel(opts.KeywordModelPath)
	if err != nil {
		return nil, errors.Errorf("could not load keyword model from %s: %v", opts.KeywordModelPath, err)
	}

	expr, err := pythonexpr.NewShardedModel(ctx, opts.ExprModelShards, opts.ExprModelOpts)
	if err != nil {
		return nil, errors.Errorf("could not load expr model from %s: %v", opts.ExprModelShards, err)
	}

	partialCallProb, err := callprob.NewModel(opts.CallProbSubtokenModelPath, true)
	if err != nil {
		return nil, errors.Errorf("could not load partialcall-prob model from %s: %v", opts.CallProbSubtokenModelPath, err)
	}

	fullCallProb, err := callprob.NewModel(opts.CallProbSubtokenModelPath, false)
	if err != nil {
		return nil, errors.Errorf("could not load fullcall-prob model from %s: %v", opts.CallProbSubtokenModelPath, err)
	}

	callModelCallProb, err := callprobcallmodel.NewModel(opts.CallProbCallModelPath)
	if err != nil {
		return nil, errors.Errorf("could not load callmodel-prob model from %s: %v", opts.CallProbCallModelPath, err)
	}

	mtacConf, err := mtacconf.NewModel(opts.MTACConfModelPath)
	if err != nil {
		return nil, errors.Errorf("could not load MTAC-confidence model from %s: %v", opts.MTACConfModelPath, err)
	}
	return &Models{
		Keyword:           kwModel,
		Expr:              expr,
		PartialCallProb:   partialCallProb,
		FullCallProb:      fullCallProb,
		MTACConf:          mtacConf,
		CallModelCallProb: callModelCallProb,
	}, nil
}

// Reset unloads data
func (m *Models) Reset() {
	m.Keyword.Reset()
	m.Expr.Reset()
	m.PartialCallProb.Reset()
	m.FullCallProb.Reset()
	m.MTACConf.Reset()
	m.CallModelCallProb.Reset()
}
