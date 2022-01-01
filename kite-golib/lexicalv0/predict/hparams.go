package predict

import (
	"encoding/json"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// ModelType specifies the type of the underlying model
type ModelType string

const (
	// ModelTypeLexical ...
	ModelTypeLexical ModelType = "lexical"

	// ModelTypePrefixSuffix ...
	ModelTypePrefixSuffix ModelType = "prefix_suffix"
)

// HParams holds the hyperparameters used for a model
type HParams struct {
	VocabSize          int       `json:"n_vocab"`
	EmbeddingSize      int       `json:"n_embd"`
	ContextSize        int       `json:"n_ctx"`
	NumHeads           int       `json:"n_head"`
	NumLayers          int       `json:"n_layer"`
	NumPredictionSlots int       `json:"n_prediction_slots"`
	ModelType          ModelType `json:"model_type"`
	NLangs             int       `json:"n_langs"`
	UseBytes           bool      `json:"use_bytes"`
	FullEmbdSize       int       `json:"n_full_embd"`
}

// NewHParams loads HParams from the provided path
func NewHParams(path string) (HParams, error) {
	f, err := fileutil.NewCachedReader(path)
	if err != nil {
		return HParams{}, errors.Errorf("error reading params from '%s': %v", path, err)
	}
	defer f.Close()

	var params HParams
	if err := json.NewDecoder(f).Decode(&params); err != nil {
		return HParams{}, errors.Errorf("error decoding params from '%s': %v", path, err)
	}

	// backwards compatibility
	if params.ModelType == "" {
		params.ModelType = ModelTypeLexical
	}

	return params, nil
}
