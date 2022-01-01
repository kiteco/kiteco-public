package inspect

import (
	"fmt"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

// Attention distribution (num tokens, num tokens)
type Attention [][]float32

// Attentions ...
type Attentions struct {
	// (layers, heads, num before, num total tokens)
	Befores [][]Attention
	// (layers, heads, num after, num total tokens)
	Afters [][]Attention
	// (layers, heads, num predict, num total tokens)
	Predicts [][]Attention

	// aggregated
	// (num before, num total)
	Before Attention

	// (num after, num total)
	After Attention

	// (num predict, num total)
	Predict Attention
}

// GetAttention gives the attention values, by head and aggregated.
func GetAttention(sample Sample) (Attentions, error) {
	layers, err := getLayers(sample)
	if err != nil {
		return Attentions{}, errors.Wrapf(err, "error getting layerwise attention")
	}

	before, err := aggregate(layers.Befores)
	if err != nil {
		return Attentions{}, errors.Wrapf(err, "error aggregating before attention")
	}
	layers.Before = before

	after, err := aggregate(layers.Afters)
	if err != nil {
		return Attentions{}, errors.Wrapf(err, "error aggregating after attention")
	}
	layers.After = after

	predict, err := aggregate(layers.Predicts)
	if err != nil {
		return Attentions{}, errors.Wrapf(err, "error aggregating predict attention")
	}
	layers.Predict = predict

	return layers, nil
}

func getLayers(sample Sample) (Attentions, error) {
	params, err := GetParams(sample)
	if err != nil {
		return Attentions{}, errors.Wrapf(err, "unable to get params")
	}

	var attns Attentions
	// TODO: make this more efficient by getting all of the layers at once
	for i := 0; i < params.NumLayers; i++ {
		layer, err := getLayer(sample, i)
		if err != nil {
			return Attentions{}, err
		}

		attns.Befores = append(attns.Befores, layer.Befores[0])
		if len(layer.Afters) > 0 {
			attns.Afters = append(attns.Afters, layer.Afters[0])
		}
		if len(layer.Predicts) > 0 {
			attns.Predicts = append(attns.Predicts, layer.Predicts[0])
		}
	}
	return attns, nil
}

func getLayer(sample Sample, layer int) (Attentions, error) {
	predictor, err := getPredictor(sample.Query)
	if err != nil {
		return Attentions{}, errors.Wrapf(err, "error getting predictor")
	}

	if predictor.GetHParams().ModelType != predict.ModelTypePrefixSuffix {

		feed := map[string]interface{}{
			"placeholders/context": [][]int64{toInt64(sample.Prediction.Meta.ContextBefore)},
		}
		if lt := predictor.GetEncoder().LangTagForPath(sample.Query.Path); lt > -1 {
			feed["placeholders/langs"] = []int64{int64(lt)}
		}

		model := predictor.GetModel()
		op := fmt.Sprintf("transformer/h%d/attn/truediv", layer)
		if ok, _ := model.OpExists(op); !ok {
			op = fmt.Sprintf("transformer/h%d/attn/attn_weights", layer)
		}

		res, err := model.Run(feed, []string{op})
		if err != nil {
			return Attentions{}, errors.Wrapf(err, "error getting attentions from model")
		}
		// (batch, heads, num tokens, num tokens)
		values := res[op].([][][][]float32)

		// (heads, num toks, num toks)
		attns := make([]Attention, 0, len(values[0]))
		for _, head := range values[0] {
			attns = append(attns, head)
		}

		return Attentions{
			Befores: [][]Attention{attns},
		}, nil
	}

	feed := predict.PrefixSuffixInputs{
		Before:  toInt64(sample.Prediction.Meta.ContextBefore),
		After:   toInt64(sample.Prediction.Meta.ContextAfter),
		Predict: [][]int64{toInt64(sample.Prediction.Meta.ContextPredict)},
	}.Feed(-1)

	op := fmt.Sprintf("test/transform/transform/h%d/attn/attn_weights", layer)

	// (1, heads, num tokens all, num tokens all)
	res, err := predictor.GetModel().Run(feed, []string{op})
	if err != nil {
		return Attentions{}, errors.Wrapf(err, "error getting attentions from model")
	}

	nBefore := len(sample.Prediction.Meta.ContextBefore)
	nAfter := len(sample.Prediction.Meta.ContextAfter)

	var befores, afters, predicts []Attention
	for _, head := range res[op].([][][][]float32)[0] {
		var before, after, predict Attention
		for dest, row := range head {
			// re arrange the source tokens so they match the original
			// token ordering
			// in tensorflow the source tokens are [before, after, predict]
			// and we want [before, predict, after]
			newRow := append([]float32{}, row[:nBefore]...)
			newRow = append(newRow, row[nBefore+nAfter:]...)
			newRow = append(newRow, row[nBefore:nBefore+nAfter]...)
			switch {
			case dest < nBefore:
				before = append(before, newRow)
			case dest < nBefore+nAfter:
				after = append(after, newRow)
			default:
				predict = append(predict, newRow)
			}
		}
		befores = append(befores, before)
		afters = append(afters, after)
		predicts = append(predicts, predict)
	}

	return Attentions{
		Befores:  [][]Attention{befores},
		Afters:   [][]Attention{afters},
		Predicts: [][]Attention{predicts},
	}, nil
}

// aggregate across layers and heads
func aggregate(layers [][]Attention) (Attention, error) {
	var aggregated Attention
	for _, layer := range layers {
		if len(layers[0]) != len(layer) {
			return nil, errors.New("invalid shape, non-uniform layer dimensions")
		}
		for _, head := range layer {
			if len(layer[0]) != len(head) {
				return nil, errors.New("invalid shape, non-uniform head dimensions")
			}

			for dest, row := range head {
				if len(row) != len(head[0]) {
					return nil, errors.New("invalid shape, non-uniform number of sources")
				}
				if dest >= len(aggregated) {
					aggregated = append(aggregated, []float32{})
				}
				for src, val := range row {
					if src >= len(aggregated[dest]) {
						aggregated[dest] = append(aggregated[dest], 0)
					}
					aggregated[dest][src] += val
				}
			}
		}
	}

	for i, row := range aggregated {
		normRow, err := normalize(row)
		if err != nil {
			return nil, err
		}
		aggregated[i] = normRow
	}

	return aggregated, nil
}

func normalize(weights []float32) ([]float32, error) {
	var norm float32
	for _, weight := range weights {
		if weight < 0 {
			return nil, errors.New("negative weight")
		}
		norm += weight
	}
	var normalized []float32
	for _, weight := range weights {
		normalized = append(normalized, weight/norm)
	}
	return normalized, nil
}

func toInt64(nums []int) []int64 {
	var result []int64
	for _, num := range nums {
		result = append(result, int64(num))
	}
	return result
}
