package callprobcallmodel

import (
	"encoding/json"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/threshold"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

const (
	predOp = "pred/pred"
)

// Model for predicting the probability of a call
type Model struct {
	model  *tensorflow.Model
	params threshold.Thresholds
}

// NewBaseModel from the specified dir, this loads the tf model but NOT the
// thresholds
func NewBaseModel(dir string) (*Model, error) {
	modelName := "full_call_prob_model.frozen.pb"
	model, err := tensorflow.NewModel(fileutil.Join(dir, modelName))
	if err != nil {
		return nil, errors.Errorf("error building model: %v", err)
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
		return nil, errors.Errorf("error opening parameter file %s: %v", pi, err)
	}
	defer r.Close()

	var thresholdSet threshold.Set
	if err := json.NewDecoder(r).Decode(&thresholdSet); err != nil {
		return nil, errors.Errorf("error decoding params %s: %v", pi, err)
	}
	thresholds := thresholdSet.FullCall

	return &Model{
		model:  model.model,
		params: thresholds,
	}, nil
}

// Inputs used to generate model features and to run inference
type Inputs struct {
	RM          pythonresource.Manager
	Cursor      int64
	RAST        *pythonanalyzer.ResolvedAST
	CallComps   []pythongraph.PredictedCall
	Sym         pythonresource.Symbol
	NumOrigArgs int // number of the argument that user already typed if any
	ScopeSize   int
}

// Reset unloads data
func (m *Model) Reset() {
	m.model.Unload()
}

// Infer returns a list of floats representing the inferred confidences of each call completion.
func (m *Model) Infer(ctx kitectx.Context, in Inputs) ([]float32, error) {
	if !m.IsLoaded() {
		return nil, errors.Errorf("model not loaded")
	}
	defer modelInferDuration.DeferRecord(time.Now())

	start := time.Now()
	feat, err := NewFeatures(in)
	if err != nil {
		return nil, err
	}
	modelWeight, err := m.Weights()
	if err != nil {
		return nil, err
	}
	for i, f := range feat.Comp {
		in.CallComps[i].MetaData.FilteringFeatures = append(f.Weights(), feat.Contextual.Weights()...)
		in.CallComps[i].MetaData.ModelWeight = modelWeight
	}

	newFeaturesDuration.RecordDuration(time.Since(start))

	res, err := m.model.Run(feat.feeds(), []string{predOp})
	if err != nil {
		return nil, err
	}

	pred := res[predOp].([][]float32)
	confidences := make([]float32, 0, len(pred))
	for i, row := range pred {
		conf := row[0]
		if feat.Comp[i].Skip {
			conf = -1
		}
		confidences = append(confidences, conf)
	}

	return confidences, nil
}

// Params returns the necessary thresholds for the callProb model.
func (m *Model) Params() threshold.Thresholds {
	return m.params
}

// IsLoaded returns true if the underlying model was successfully loaded.
func (m *Model) IsLoaded() bool {
	return m.model != nil
}

// Weights returns the raw weights from the model
func (m *Model) Weights() ([]pythongraph.NameAndWeight, error) {
	if !m.IsLoaded() {
		return nil, errors.Errorf("model not loaded")
	}

	fd := Features{}.feeds()

	weightsOp := "weights"

	res, err := m.model.Run(fd, []string{weightsOp})
	if err != nil {
		return nil, err
	}

	var weights []float32
	for _, f := range res[weightsOp].([][]float32) {
		weights = append(weights, f[0])
	}
	context := ContextualFeatures{}.Weights()
	context[0].Weight = weights[0]
	context[1].Weight = weights[1]

	comps := CompFeatures{}.Weights()
	for i, comp := range comps {
		comp.Weight = weights[2+i]
		comps[i] = comp
	}

	return append(context, comps...), nil
}
