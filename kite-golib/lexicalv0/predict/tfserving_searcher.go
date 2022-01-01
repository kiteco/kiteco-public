package predict

import (
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/status"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/tfserving"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

// ErrRemoteNotReady is returned when the client cannot retrieve model assets from the server,
// e.g the server is unreachable
var ErrRemoteNotReady = errors.New("remote not ready")

// TFServingOptions ...
type TFServingOptions struct {
	Addr      string
	ModelName string
	LangGroup lexicalv0.LangGroup
	ModelPath string
}

// Empty returns whether options are empty
func (t TFServingOptions) Empty() bool {
	return (len(t.Addr)+len(t.ModelPath)+len(t.ModelPath) == 0) && t.LangGroup.Empty()
}

// NewTFServingOptions creates TFServing options with a backup model path.
func NewTFServingOptions(addr, modelName string, remoteGroup lexicalv0.LangGroup, modelPath string) TFServingOptions {
	return TFServingOptions{
		Addr:      addr,
		ModelName: modelName,
		LangGroup: remoteGroup,
		ModelPath: modelPath,
	}
}

// TFServingSearcher is a predictor that queries a tfserving server
type TFServingSearcher struct {
	opts   TFServingOptions
	client *tfserving.Client

	m              sync.Mutex
	encoder        *lexicalv0.FileEncoder
	hparams        HParams
	config         SearchConfig
	prefixSuffixLM bool

	ready   int32
	loading int32
}

// NewTFServingSearcher connects to a tfserving server at the provided address
func NewTFServingSearcher(opts TFServingOptions) (*TFServingSearcher, error) {
	if opts.Empty() {
		return nil, errors.Errorf("TFServingSearcher given empty TFServingOptions")
	}

	// NOTE: KEEP THIS LAST -- if we need to fetch anything else here, make sure to do it before
	// attempting to connect to the server because we want them in datadeps, but server connection
	// could fail during datadeps generation
	client, err := tfserving.NewClient(opts.Addr, opts.ModelName)
	if err != nil {
		return nil, err
	}

	predictor := &TFServingSearcher{
		opts:   opts,
		client: client,
	}

	// asyncronously load remote model assets, as this can cause the client to hang sometimes
	kitectx.Go(func() error {
		predictor.loadModelAssets()
		return nil
	})

	return predictor, nil
}

func (t *TFServingSearcher) loadModelAssets() {
	// Only run if we aren't already loading
	if !atomic.CompareAndSwapInt32(&t.loading, 0, 1) {
		return
	}

	// Turn off loading flag once this method returns
	defer atomic.StoreInt32(&t.loading, 0)

	log.Println("loading remote model assets")

	encoder, params, config, err := LoadModelAssets(t.opts.ModelPath, t.opts.LangGroup)
	if err != nil {
		log.Println("failed to load remote model assets")
		return
	}

	log.Println("succesfully loaded remote model assets")

	t.encoder = encoder
	t.hparams = params
	t.config = config
	t.prefixSuffixLM = params.ModelType == ModelTypePrefixSuffix

	atomic.StoreInt32(&t.ready, 1)
}

// PredictChan ...
func (t *TFServingSearcher) PredictChan(kctx kitectx.Context, in Inputs) (chan Predicted, chan error) {
	if atomic.LoadInt32(&t.ready) == int32(0) {
		return handlePredictChanInitErr(ErrRemoteNotReady)
	}

	searchIn, err := newSearcherInputs(in, t.encoder, t.config, t.prefixSuffixLM)
	if err != nil {
		return handlePredictChanInitErr(err)
	}

	searchIn.Incremental = make(chan Predicted, searchIn.Config.Depth*searchIn.Config.BeamWidth)
	errChan := kitectx.Go(func() error {
		_, err := t.search(kctx, searchIn)

		// Check to see if we get an invalid argument error. This will happen when the model being
		// used in the backend changed to/from a tuned model because the hparams for vocab size change,
		// resulting in an error with the request (which depends on vocab size). We can safely reload the
		// params now because the server is now serving the updated model and vocab/hparams.
		if grpcstatus.Code(err) == codes.InvalidArgument {
			log.Println("detected invalid argument error, reloading vocab/hparams/searchconfig")
			kitectx.Go(func() error {
				t.loadModelAssets()
				return nil
			})
		}

		return err
	})
	return searchIn.Incremental, errChan
}

// GetHParams implements Predictor
func (t *TFServingSearcher) GetHParams() HParams {
	t.m.Lock()
	defer t.m.Unlock()
	return t.hparams
}

// GetEncoder implements Predictor
func (t *TFServingSearcher) GetEncoder() *lexicalv0.FileEncoder {
	t.m.Lock()
	defer t.m.Unlock()
	return t.encoder
}

// Unload implements Predictor
func (t *TFServingSearcher) Unload() {
	// no-op
}

// GetLexer implements Predictor
func (t *TFServingSearcher) GetLexer() lexer.Lexer {
	return t.encoder.Lexer
}

// SetStrictChecking enables stricter checks about the input, search params, etc
// implements Predictor
func (t *TFServingSearcher) SetStrictChecking(val bool) {
	// no-op
}

// GetModel implements Predictor
func (t *TFServingSearcher) GetModel() *tensorflow.Model {
	return nil
}

// Logits implements Predictor
func (t *TFServingSearcher) Logits(Inputs) ([]float32, error) {
	return nil, nil
}

// Predict implements Predictor
func (t *TFServingSearcher) Predict(kctx kitectx.Context, in Inputs) (Predictions, error) {
	if atomic.LoadInt32(&t.ready) == int32(0) {
		return Predictions{}, ErrRemoteNotReady
	}

	searchIn, err := newSearcherInputs(in, t.encoder, t.config, t.prefixSuffixLM)
	if err != nil {
		return Predictions{}, err
	}

	preds, err := t.search(kctx, searchIn)
	if err != nil {
		return Predictions{}, err
	}

	var filtered []Predicted
	for _, p := range preds {
		if len(p.TokenIDs) == searchIn.Config.Depth {
			filtered = append(filtered, p)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Prob > filtered[j].Prob
	})

	return Predictions{
		Preds: filtered,
		// TODO: we can add other meta info if we decide to use the
		// inspector with this model...
	}, nil
}

// --

func (t *TFServingSearcher) search(kctx kitectx.Context, in searcherInputs) ([]Predicted, error) {
	defer status.SearchDuration.DeferRecord(time.Now())

	if in.Incremental != nil {
		defer close(in.Incremental)
	}
	// check abort after we defer/close the channel in case we are canceled
	kctx.CheckAbort()

	var validPrefixIDs []int64
	if in.Prefix != "" {
		validPrefixIDs = idsMatchingPrefixSlice(t.encoder, in.Prefix)
	}

	var results [][]int64
	var probs [][]float32
	var err error

	prefixMask := PrefixIDMask(validPrefixIDs, t.hparams.VocabSize)
	if t.prefixSuffixLM {
		before, _ := PadContext(in.ContextBefore, in.Config.Window, -1)
		after, _ := PadContext(in.ContextAfter, in.Config.Window, -1)

		results, probs, err = t.client.SearchPrefixSuffix(
			kctx,
			before,
			after,
			prefixMask,
		)
	} else {
		context, contextMask := PadContext(in.ContextBefore, in.Config.Window, 0)
		results, probs, err = t.client.Search(
			kctx,
			context,
			contextMask,
			prefixMask,
		)
	}

	if err != nil {
		return nil, err
	}

	predicted, err := buildPredicted(results, probs, in.Config.MinP, in.Prefix, t.encoder)
	if err != nil {
		return nil, err
	}

	if in.Incremental != nil {
		for _, pred := range predicted {
			in.Incremental <- pred
		}
	}

	return predicted, err
}
