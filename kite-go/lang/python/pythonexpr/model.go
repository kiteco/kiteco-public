package pythonexpr

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonattribute"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncall"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph/traindata"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

const maxHops = 3

var feedConfig = pythongraph.GraphFeedConfig{
	EdgeSet: []pythongraph.EdgeType{
		pythongraph.ASTChild,
		pythongraph.NextToken,
		pythongraph.DataFlow,
	},
}

// MetaInfo for the model
type MetaInfo struct {
	Attr              pythonattribute.SymbolInfo `json:"attr"`
	Call              pythoncall.FuncInfos       `json:"call"`
	AttrBase          AttrBaseInfo               `json:"attr_base"`
	NameSubtokenIndex traindata.SubtokenIndex    `json:"name_subtoken_index"`
	TypeSubtokenIndex traindata.SubtokenIndex    `json:"type_subtoken_index"`
	ProductionIndex   traindata.ProductionIndex  `json:"production_index"`
}

// ToModelMeta returns the components of the metainfo that are necessary for prediction
func (m MetaInfo) ToModelMeta() pythongraph.ModelMeta {
	return pythongraph.ModelMeta{
		NameSubtokenIndex: m.NameSubtokenIndex,
		TypeSubtokenIndex: m.TypeSubtokenIndex,
		ProductionIndex:   m.ProductionIndex,
	}
}

// Options for setting up the model
type Options struct {
	// TFCallback, if set, will be called every time the underlying Tensorflow model is run
	TFCallback tensorflow.RunCallback `json:"-"`

	// UseUncompressed model for inference
	UseUncompressed bool
}

// DefaultOptions ...
var DefaultOptions = Options{}

// ModelShard for predicting expressions for a subset of packages
type ModelShard struct {
	dir         string
	info        MetaInfo
	model       *tensorflow.Model
	canonToSyms map[pythonimports.Hash][]pythonimports.DottedPath
	opts        Options
}

// NewMetaInfo from the specified directory
func NewMetaInfo(dir string) (MetaInfo, error) {
	mif := fileutil.Join(dir, "metainfo-inference.json")
	r, err := fileutil.NewCachedReader(mif)
	if err != nil {
		return MetaInfo{}, fmt.Errorf("error opening metainfo %s: %v", mif, err)
	}
	defer r.Close()

	var mi MetaInfo
	if err := json.NewDecoder(r).Decode(&mi); err != nil {
		return MetaInfo{}, err
	}
	return mi, nil
}

// NewModel is a temporary/hack wrapper for NewModelShard so old users of the ExprModel don't break.
// TODO(tarak): This needs to be handled once we truely switch to shards. User will have to specify
// which shard they want before running the model. API for this TBD.
func NewModel(dir string, opts Options) (Model, error) {
	return newModelShard(Shard{ModelPath: dir}, opts)
}

// newModelShard from the specified Shard.
func newModelShard(shard Shard, opts Options) (Model, error) {
	mi, err := NewMetaInfo(shard.ModelPath)
	if err != nil {
		return nil, err
	}
	var path string
	if opts.UseUncompressed {
		path = fileutil.Join(fileutil.Dir(shard.ModelPath), "expr_model.uncompressed.frozen.pb")
	} else {
		path = fileutil.Join(shard.ModelPath, "expr_model.frozen.pb")
	}

	model, err := tensorflow.NewModel(path)
	if err != nil {
		return nil, fmt.Errorf("error building model: %v", err)
	}
	model.RunCallback = opts.TFCallback

	canonToSyms := make(map[pythonimports.Hash][]pythonimports.DottedPath)
	for c, syms := range mi.Attr.CanonToSyms {
		ch := pythonimports.NewDottedPath(c).Hash
		for _, sym := range syms {
			canonToSyms[ch] = append(canonToSyms[ch], pythonimports.NewDottedPath(sym))
		}
	}
	mi.Attr.CanonToSyms = nil

	return &ModelShard{
		dir:         shard.ModelPath,
		model:       model,
		info:        mi,
		canonToSyms: canonToSyms,
		opts:        opts,
	}, nil

}

// Load will load the model if unloaded
func (m *ModelShard) Load() error {
	return m.model.LoadAndLock()
}

// Reset unloads data
func (m *ModelShard) Reset() {
	m.model.Unload()
}

// IsLoaded returns true if the underlying model was successfully loaded.
func (m *ModelShard) IsLoaded() bool {
	return m.model != nil
}

// AttrSupported returns nil if the Model is able to provide completions for the
// specified parent.
func (m *ModelShard) AttrSupported(rm pythonresource.Manager, parent pythonresource.Symbol) error {
	if !m.IsLoaded() {
		return fmt.Errorf("model is not loaded")
	}

	_, _, err := m.AttrCandidates(rm, parent)
	if err != nil {
		return err
	}
	return nil
}

func (m *ModelShard) attrCandidates(rm pythonresource.Manager, parent pythonresource.Symbol) ([]int32, []pythonresource.Symbol, error) {
	children, err := rm.Children(parent)
	if err != nil {
		return nil, nil, err
	}

	syms := make([]pythonresource.Symbol, 0, len(children))
	idxs := make([]int32, 0, len(children))

	for _, child := range children {
		cs, err := rm.ChildSymbol(parent, child)
		if err != nil {
			continue
		}

		idx, ok := m.info.ProductionIndex.Index(cs.PathHash())
		if !ok {
			continue
		}
		syms = append(syms, cs)
		idxs = append(idxs, idx)
	}

	if len(syms) == 0 {
		return nil, nil, fmt.Errorf("no valid candidates for %v", parent)
	}

	return idxs, syms, nil
}

// AttrCandidates for the specified parent symbol
func (m *ModelShard) AttrCandidates(rm pythonresource.Manager, parent pythonresource.Symbol) ([]int32, []pythonresource.Symbol, error) {
	if !m.IsLoaded() {
		return nil, nil, fmt.Errorf("model is not loaded")
	}

	// try symbol as is
	idxs, syms, err := m.attrCandidates(rm, parent)
	if err == nil {
		return idxs, syms, nil
	}

	// try canonicalizing
	parent = parent.Canonical()
	for _, sym := range m.canonToSyms[parent.PathHash()] {
		ps, err := rm.PathSymbol(sym)
		if err != nil {
			continue
		}
		idxs, syms, err = m.attrCandidates(rm, ps)
		if err == nil {
			return idxs, syms, nil
		}
	}

	return nil, nil, fmt.Errorf("unsupported parent %v", parent)
}

// CallSupported returns nil if the model is able to provide call completions for the
// specified symbol.
func (m *ModelShard) CallSupported(rm pythonresource.Manager, sym pythonresource.Symbol) error {
	if !m.IsLoaded() {
		return fmt.Errorf("model is not loaded")
	}

	_, err := m.FuncInfo(rm, sym)
	if err != nil {
		return err
	}
	return nil
}

// Dir returns the directory from which the model was loaded.
func (m *ModelShard) Dir() string {
	return m.dir
}

// FuncInfo gets all the needed info for call completion
func (m *ModelShard) FuncInfo(rm pythonresource.Manager, sym pythonresource.Symbol) (*pythongraph.FuncInfo, error) {
	fSym := pythoncall.SymbolForFunc(rm, sym)
	if fSym.Nil() {
		return nil, fmt.Errorf("unsupported symbol %s", sym.PathString())
	}

	fs := fSym.PathString()

	fi := m.info.Call.Infos[fs]
	if fi == nil {
		return nil, fmt.Errorf("unsupported func %s", fs)
	}

	patterns := traindata.NewCallPatterns(rm, sym)
	if patterns == nil {
		return nil, fmt.Errorf("no patterns for %s", fs)
	}

	fig := &pythongraph.FuncInfo{
		Symbol:             fSym,
		Patterns:           patterns,
		ArgTypeIdxs:        make(map[traindata.ArgType]int32, len(traindata.ArgTypes)),
		KwargNameIdxs:      make([]pythongraph.NameAndIdx, 0, len(fi.KwargNames)),
		ArgPlaceholderIdxs: make(map[string]map[traindata.ArgPlaceholder]int32, len(patterns.ArgsByName)),
	}

	for _, at := range traindata.ArgTypes {
		key := traindata.IDForChooseArgType(fs, at)
		idx, ok := m.info.ProductionIndex.Index(key)
		if !ok {
			return nil, fmt.Errorf("no arg type index for %s", key)
		}
		fig.ArgTypeIdxs[at] = idx
	}

	for name := range patterns.ArgsByName {
		apToIdx := make(map[traindata.ArgPlaceholder]int32, 2)

		for _, ap := range traindata.ArgPlaceholders {
			key := traindata.IDForChooseArgPlaceholder(fs, name, ap)
			idx, ok := m.info.ProductionIndex.Index(key)
			if !ok {
				return nil, fmt.Errorf("no arg placeholder index for %s", key)
			}
			apToIdx[ap] = idx
		}
		fig.ArgPlaceholderIdxs[name] = apToIdx
	}

	for _, kwarg := range fi.KwargNames {
		// TODO: make this consistent
		if _, ok := patterns.ArgsByName[kwarg]; !ok {
			continue
		}

		key := traindata.IDForChooseKwarg(fs, kwarg)
		idx, ok := m.info.ProductionIndex.Index(key)
		if !ok {
			return nil, fmt.Errorf("no kwarg name index for %s", key)
		}
		fig.KwargNameIdxs = append(fig.KwargNameIdxs, pythongraph.NameAndIdx{
			Name: kwarg,
			Idx:  idx,
		})
	}
	return fig, nil
}

func (m *ModelShard) byPopularity(rm pythonresource.Manager, parent pythonresource.Symbol) ([]pythongraph.ScoredAttribute, error) {
	children, err := rm.Children(parent)
	if err != nil {
		return nil, fmt.Errorf("error getting children: %v", err)
	}

	scored := make([]pythongraph.ScoredAttribute, 0, len(children))
	for _, c := range children {
		cs, err := rm.ChildSymbol(parent, c)
		if err != nil {
			continue
		}

		score := rm.SymbolCounts(cs)
		if score == nil {
			continue
		}

		scored = append(scored, pythongraph.ScoredAttribute{
			Symbol: cs,
			Score:  float32(score.Attribute),
		})
	}

	if len(scored) == 0 {
		return nil, fmt.Errorf("unable to find any valid children for %v", parent)
	}

	return scored, nil
}

// Input bundles the data needed for expr prediction
type Input struct {
	RM                          pythonresource.Manager
	RAST                        *pythonanalyzer.ResolvedAST
	Words                       []pythonscanner.Word
	Src                         []byte
	Expr                        pythonast.Expr
	Arg                         *pythonast.Argument
	MaxPatterns                 int
	AlwaysUsePopularityForAttrs bool
	MungeBufferForAttrs         bool
	Tracer                      io.Writer
	Saver                       pythongraph.Saver
	Depth                       int
	UsePartialDecoder           bool
}

// GGNNResults is a combination of old and new results from the GGNN predictor
// TODO remove and replace by only the Prediction slice when removing UsePartialDecoder feature flag
type GGNNResults struct {
	OldPredictorResult *pythongraph.PredictionTreeNode
	NewPredictorResult []pythongraph.Prediction
}

// Predict an expression completion
func (m *ModelShard) Predict(ctx kitectx.Context, in Input) (*GGNNResults, error) {
	if !m.IsLoaded() {
		return nil, fmt.Errorf("model is not loaded")
	}

	In := pythongraph.Inputs{
		RM:     in.RM,
		RAST:   in.RAST,
		Words:  in.Words,
		Buffer: in.Src,
	}

	callbacks := pythongraph.ExprCallbacks{
		Attr: pythongraph.AttrCallbacks{
			Supported:    m.AttrSupported,
			Candidates:   m.AttrCandidates,
			ByPopularity: m.byPopularity,
		},
		Call: pythongraph.CallCallbacks{
			Supported: m.CallSupported,
			Info:      m.FuncInfo,
		},
	}

	// TODO remove the else part when removing UsePartialDecoder feature flag
	if in.UsePartialDecoder {
		predictor, err := pythongraph.NewNewPredictor(ctx, pythongraph.ContextGraphConfig{
			Graph:     feedConfig,
			MaxHops:   maxHops,
			Propagate: true,
		}, pythongraph.PredictorInputs{
			ModelMeta:            m.info.ToModelMeta(),
			Model:                m.model,
			In:                   In,
			Site:                 in.Expr,
			Tracer:               in.Tracer,
			Callbacks:            callbacks,
			Saver:                in.Saver,
			UseUncompressedModel: m.opts.UseUncompressed,
		})

		if err != nil {
			return nil, err
		}

		predictions, err := predictor.Expand(ctx)
		if err != nil {
			return nil, err
		}
		return &GGNNResults{nil, predictions}, nil

	}

	predictor, err := pythongraph.NewPredictor(ctx, pythongraph.ContextGraphConfig{
		Graph:     feedConfig,
		MaxHops:   maxHops,
		Propagate: true,
	}, pythongraph.PredictorInputs{
		ModelMeta:            m.info.ToModelMeta(),
		Model:                m.model,
		In:                   In,
		Site:                 in.Expr,
		Tracer:               in.Tracer,
		Callbacks:            callbacks,
		Saver:                in.Saver,
		UseUncompressedModel: m.opts.UseUncompressed,
	})
	if err != nil {
		return nil, err
	}

	predictionTreeNode, err := predictor.PredictExpr(ctx)
	if err != nil {
		return nil, err
	}
	return &GGNNResults{predictionTreeNode, nil}, nil
}

// MetaInfo for the model
func (m *ModelShard) MetaInfo() MetaInfo {
	return m.info
}
