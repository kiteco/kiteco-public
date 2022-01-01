package mtacconf

import (
	"encoding/json"
	"fmt"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/threshold"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/response"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

const (
	predOp = "pred/pred"
)

// Model for predicting the probability of a call
type Model struct {
	model  *tensorflow.Model
	params threshold.MTACThreshold
}

// NewBaseModel from the specified dir, this loads the tf model but NOT the
// thresholds
func NewBaseModel(dir string) (*Model, error) {
	model, err := tensorflow.NewModel(fileutil.Join(dir, "mtac_conf_model.frozen.pb"))
	if err != nil {
		return nil, fmt.Errorf("error building model: %v", err)
	}

	return &Model{
		model: model,
	}, nil
}

// NewModel from the specified path
func NewModel(dir string) (*Model, error) {
	model, err := NewBaseModel(dir)
	if err != nil {
		return nil, err
	}

	pi := fileutil.Join(dir, "params.json")
	r, err := fileutil.NewCachedReader(pi)
	if err != nil {
		return nil, fmt.Errorf("error opening parameter file %s: %v", pi, err)
	}
	defer r.Close()

	var thresholds threshold.MTACThreshold
	if err := json.NewDecoder(r).Decode(&thresholds); err != nil {
		return nil, fmt.Errorf("error decoding params %s: %v", pi, err)
	}

	return &Model{
		model:  model.model,
		params: thresholds,
	}, nil
}

// CallSym contains symbol of the call and the position argument where MTAC completion is being suggested
type CallSym struct {
	Sym pythonresource.Symbol
	Pos int
}

// MixData represents extra meta information used for mixing
type MixData struct {
	Call     CallSym // adding this for MTAC under call
	Scenario threshold.MTACScenario
}

// Completion represents a completion input
type Completion struct {
	Score    float64
	MixData  MixData
	Source   response.EditorCompletionSource
	Referent pythontype.Value
}

// Inputs used to generate model features and to run inference
type Inputs struct {
	RM     pythonresource.Manager
	Cursor int64
	Words  []pythonscanner.Word
	RAST   *pythonanalyzer.ResolvedAST
	Comps  []Completion
}

// Reset unloads data
func (m *Model) Reset() {
	m.model.Unload()
}

// Infer returns a list of floats representing the inferred confidences of each call completion.
func (m *Model) Infer(ctx kitectx.Context, in Inputs) ([]float32, error) {
	if !m.IsLoaded() {
		return nil, fmt.Errorf("model not loaded")
	}

	feat, err := NewFeatures(in)
	if err != nil {
		return nil, err
	}

	res, err := m.model.Run(feat.feeds(), []string{predOp})
	if err != nil {
		return nil, err
	}

	pred := res[predOp].([][]float32)
	confidences := make([]float32, 0, len(pred))
	for _, row := range pred {
		confidences = append(confidences, row[0])
	}

	return confidences, nil
}

// Params returns the necessary thresholds for the callProb model.
func (m *Model) Params() threshold.MTACThreshold {
	return m.params
}

// IsLoaded returns true if the underlying model was successfully loaded.
func (m *Model) IsLoaded() bool {
	return m.model != nil
}
