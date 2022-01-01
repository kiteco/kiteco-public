package pythonmixing

import (
	"fmt"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncompletions"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

const (
	predOp = "pred/pred"
)

// Model to mix completions
type Model struct {
	model *tensorflow.Model
}

// NewModel from path
func NewModel(path string) (*Model, error) {
	model, err := tensorflow.NewModel(path)
	if err != nil {
		return nil, err
	}

	return &Model{
		model: model,
	}, nil
}

// Inputs used to generate model features and to run inference
type Inputs struct {
	RM            pythonresource.Manager
	Src           []byte
	Words         []pythonscanner.Word
	RAST          *pythonanalyzer.ResolvedAST
	AttributeExpr *pythonast.AttributeExpr
	MixInputs     []pythoncompletions.MixInput
	CallProb      func(kitectx.Context, int64, pythoncompletions.Completion) (float32, error)
}

// Reset unloads data
func (m *Model) Reset() {
	m.model.Unload()
}

// Infer returns a list of floats representing the inferred confidences of each completion.
func (m *Model) Infer(ctx kitectx.Context, in Inputs) ([]float32, error) {
	if !m.IsLoaded() {
		return nil, fmt.Errorf("model not loaded")
	}
	defer modelInferDuration.DeferRecord(time.Now())

	start := time.Now()

	feat, err := NewFeatures(ctx, in)
	if err != nil {
		return nil, err
	}
	newFeaturesDuration.RecordDuration(time.Since(start))

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

// IsLoaded returns true if the underlying model was successfully loaded.
func (m *Model) IsLoaded() bool {
	return m.model != nil
}
