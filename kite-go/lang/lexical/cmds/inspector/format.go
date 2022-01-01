package main

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

const gaps = 2

type display struct {
	FileName      string
	ModelPaths    []modelDisplay
	Config        predict.SearchConfig
	Before        []attentionDisplay
	BeforeLabels  [][]string
	After         []attentionDisplay
	AfterLabels   [][]string
	Predict       []attentionDisplay
	PredictLabels [][]string

	Expansions []expansion

	TopPPct              int
	PrefixRegularization float32
	MinP                 float32
	Shard                bool
	ShardSize            int
	Code                 string
	Duration             string
	ShowDuration         bool
	Prefix               string
	Completions          string
	Beams                string
	Latest               int
	Auto                 bool
	Key                  string
}

type modelDisplay struct {
	Path     string
	Selected bool
}

type attentionDisplay struct {
	Token string
	Color string
	Break bool
	Heads []string
}

type beam struct {
	ShowLayers    bool
	Before        []attentionDisplay
	BeforeLabels  [][]string
	After         []attentionDisplay
	AfterLabels   [][]string
	Predict       []attentionDisplay
	PredictLabels [][]string

	RawExtensions           []string
	RawExtensionScores      []string
	SelectedExtensions      []string
	SelectedExtensionScores []string
}

type expansion struct {
	Depth     string
	BeamsText string
	Beams     []beam
}

func displaySample(sample inspect.Sample, showLayers, auto bool) display {
	enc, err := inspect.GetEncoder(sample)
	if err != nil {
		return displayError(sample.Query, err, auto)
	}

	attns, err := inspect.GetAttention(sample)
	if err != nil {
		return displayError(sample.Query, err, auto)
	}

	// TODO: kind of nasty, the prefix suffix model operates on the full context
	// and the masking handles chunking.
	fullContext := append([]int{}, sample.Prediction.Meta.ContextBefore...)
	fullContext = append(fullContext, sample.Prediction.Meta.ContextPredict...)
	fullContext = append(fullContext, sample.Prediction.Meta.ContextAfter...)

	before, beforeLabels := displayAttention(fullContext, enc, attns.Befores, attns.Before)
	after, afterLabels := displayAttention(fullContext, enc, attns.Afters, attns.After)
	predict, predictLabels := displayAttention(fullContext, enc, attns.Predicts, attns.Predict)

	exps, err := expansions(sample, showLayers)
	if err != nil {
		return displayError(sample.Query, err, auto)
	}

	key, err := inspect.Key(sample)
	if err != nil {
		return displayError(sample.Query, err, auto)
	}
	completions, err := displayCompletions(sample)
	if err != nil {
		return displayError(sample.Query, err, auto)
	}
	beams, err := displayBeams(sample)
	if err != nil {
		return displayError(sample.Query, err, auto)
	}
	duration, showDuration := displayDuration(sample)
	return display{
		FileName:             displayPath(sample.Query.Path),
		ModelPaths:           displayModelPaths(sample.Query),
		TopPPct:              displayTopPPct(sample.Query),
		PrefixRegularization: sample.Query.Config.PrefixRegularization,
		MinP:                 sample.Query.Config.MinP,
		Code:                 sample.Query.Code,
		Config:               sample.Query.Config,
		Duration:             duration,
		ShowDuration:         showDuration,
		Prefix:               sample.Prediction.Meta.Prefix,
		Completions:          completions,
		Beams:                beams,
		Before:               before,
		BeforeLabels:         beforeLabels,
		After:                after,
		AfterLabels:          afterLabels,
		Predict:              predict,
		PredictLabels:        predictLabels,
		Latest:               activeTime,
		Auto:                 auto,
		Key:                  key,
		Expansions:           exps,
	}
}

func displayError(query inspect.Query, err error, auto bool) display {
	return display{
		FileName:             displayPath(query.Path),
		ModelPaths:           displayModelPaths(query),
		TopPPct:              displayTopPPct(query),
		PrefixRegularization: query.Config.PrefixRegularization,
		MinP:                 query.Config.MinP,
		Code:                 query.Code,
		Config:               query.Config,
		Prefix:               "",
		Completions:          fmt.Sprintf("Error: %s", err.Error()),
		Beams:                "",
		Latest:               activeTime,
		Auto:                 auto,
		Key:                  "",
	}
}

func displayModelPaths(query inspect.Query) []modelDisplay {
	var models []modelDisplay
	for _, path := range modelPaths {
		model := modelDisplay{
			Path:     path,
			Selected: path == query.ModelPath,
		}
		models = append(models, model)
	}
	return models
}

func displayTopPPct(query inspect.Query) int {
	return int(query.Config.TopP * 100)
}

func displayCompletions(sample inspect.Sample) (string, error) {
	return format(sample.Prediction.FinalPredictions, sample)
}

func displayDuration(sample inspect.Sample) (string, bool) {
	seconds := sample.Prediction.Duration.Seconds()
	if seconds == 0 {
		return "", false
	}
	return fmt.Sprintf("%.3f seconds", seconds), true
}

func displayBeams(sample inspect.Sample) (string, error) {
	var rounds []string
	for _, expansion := range sample.Prediction.Meta.Expansions {
		beam, err := format(expansion.BeamPredictions, sample)
		if err != nil {
			return "", err
		}
		rounds = append(rounds, beam)
	}
	return strings.Join(rounds, "\n\n"), nil
}

func displayAttention(context []int, enc *lexicalv0.FileEncoder, heads [][]inspect.Attention, total inspect.Attention) ([]attentionDisplay, [][]string) {
	decoded := enc.DecodeToVocab(context)

	var attention []attentionDisplay
	if len(total) > 0 {
		colors := [][]int{
			[]int{0x00, 0x00, 0xff},
			[]int{0x80, 0x00, 0x80},
			[]int{0xff, 0x00, 0x00},
		}
		shades := makeShades(heads, len(decoded), colors[:2])
		for i := range decoded {
			var next string
			if i+1 < len(decoded) {
				next = decoded[i+1]
			}
			attention = append(attention,
				attentionDisplay{
					decoded[i],
					shade(total[len(total)-1][i], colors[2]),
					isBreak(decoded[i], next),
					shades[i],
				},
			)
		}
	}
	layerLabels, headLabels := makeLabels(heads)
	return attention, [][]string{layerLabels, headLabels}
}

func makeShades(heads [][]inspect.Attention, n int, colors [][]int) [][]string {
	var shades [][]string
	for i := 0; i < n; i++ {
		var row []string
		for j, layer := range heads {
			color := colors[j%2]
			for k := 0; k < gaps; k++ {
				row = append(row, shade(0, color))
			}
			for _, head := range layer {
				row = append(row, shade(head[len(head)-1][i], color))
			}
		}
		shades = append(shades, row)
	}
	return shades
}

func makeLabels(heads [][]inspect.Attention) ([]string, []string) {
	layerLabels := []string{""}
	headLabels := []string{""}
	for j, layer := range heads {
		for k := 0; k < gaps; k++ {
			layerLabels = append(layerLabels, "")
			headLabels = append(headLabels, "")
		}
		for k := range layer {
			headLabels = append(headLabels, fmt.Sprintf("H%d", k))
			if k == 0 {
				layerLabels = append(layerLabels, fmt.Sprintf("L%d", j))
				continue
			}
			layerLabels = append(layerLabels, "")
		}
	}
	return layerLabels, headLabels
}

func shade(attention float32, ones []int) string {
	weight := float32(math.Pow(float64(attention), 0.5))
	if weight > 1 {
		weight = 1
	}
	zeros := []int{0xff, 0xff, 0xff}
	mids := []int{}
	for i, zero := range zeros {
		one := ones[i]
		mid := float32(zero) + (float32(one-zero) * weight)
		mids = append(mids, int(mid))
	}
	return fmt.Sprintf("#%02x%02x%02x", mids[0], mids[1], mids[2])
}

func isBreak(token string, next string) bool {
	if token == ";" {
		return true
	}
	if token == "{" && next != "}" {
		return true
	}
	return false
}

func format(predictions []predict.Predicted, sample inspect.Sample) (string, error) {
	if len(predictions) == 0 {
		return "No completions", nil
	}
	encoder, err := inspect.GetEncoder(sample)
	if err != nil {
		return "", err
	}

	var lines []string
	for _, prediction := range predictions {
		lines = append(lines, formatPrediction(prediction, encoder))
	}
	return strings.Join(lines, "\n"), nil
}

func formatPrediction(prediction predict.Predicted, encoder *lexicalv0.FileEncoder) string {
	var completion string
	if language.Lexer == lang.Text {
		// text models do rendering as part of prediction so don't include spaces
		completion = strings.Join(encoder.DecodeToStrings(prediction.TokenIDs), "")
	} else {
		completion = strings.Join(encoder.DecodeToStrings(prediction.TokenIDs), " ")
	}
	return fmt.Sprintf("%.3f: %s", prediction.Prob, completion)
}

func truncate(code string, cursor string) string {
	pieces := strings.Split(code, cursor)
	numLines := 2
	lines := strings.Split(pieces[1], "\n")
	if len(lines) < numLines {
		return code
	}
	afterCursor := strings.Join(lines[:numLines], "\n")
	return strings.Join([]string{pieces[0], cursor, afterCursor}, "")
}

func expansions(sample inspect.Sample, showLayers bool) ([]expansion, error) {
	enc, err := inspect.GetEncoder(sample)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get encoder")
	}

	original := sample.Prediction.Meta

	var exps []expansion
	for _, exp := range sample.Prediction.Meta.Expansions {
		var beams []beam

		for i, rawExts := range exp.RawPredictions {
			if i < len(exp.Before) {
				sample.Prediction.Meta.ContextBefore = exp.Before[i]
			}
			if i < len(exp.Predict) {
				sample.Prediction.Meta.ContextPredict = exp.Predict[i]
			}

			attns, err := inspect.GetAttention(sample)
			if err != nil {
				return nil, errors.New("unable to get attention")
			}

			// TODO: kind of nasty, the prefix suffix model operates on the full context
			// and the masking handles chunking.
			fullContext := append([]int{}, sample.Prediction.Meta.ContextBefore...)
			fullContext = append(fullContext, sample.Prediction.Meta.ContextPredict...)
			fullContext = append(fullContext, sample.Prediction.Meta.ContextAfter...)

			predict, predictLabels := displayAttention(fullContext, enc, attns.Predicts, attns.Predict)

			var before []attentionDisplay
			var beforeLabels [][]string
			if len(predict) == 0 {
				// for prefix suffix model the predict is empty on the first step so we only show the before attention on the first step,
				// after that the before attention doesn't change so we just show predict attention.
				// for the normal model the predict is always empty so we always show the before attention
				before, beforeLabels = displayAttention(fullContext, enc, attns.Befores, attns.Before)
			}

			rawExts, rawExtScores := displayExtensions(rawExts, enc)
			selectedExts, selectedExtScores := displayExtensions(exp.SelectedPredictions[i], enc)

			beams = append(beams, beam{
				ShowLayers:              showLayers,
				Before:                  before,
				BeforeLabels:            beforeLabels,
				Predict:                 predict,
				PredictLabels:           predictLabels,
				RawExtensions:           rawExts,
				RawExtensionScores:      rawExtScores,
				SelectedExtensions:      selectedExts,
				SelectedExtensionScores: selectedExtScores,
			})
		}

		beamsText, err := format(exp.BeamPredictions, sample)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to format beam text")
		}

		exps = append(exps, expansion{
			Depth:     fmt.Sprintf("%d", exp.AtDepth),
			BeamsText: beamsText,
			Beams:     beams,
		})
	}

	sample.Prediction.Meta = original

	return exps, nil
}

func displayPath(path string) string {
	return filepath.Base(path)
}

func displayExtensions(exts []predict.Predicted, enc *lexicalv0.FileEncoder) ([]string, []string) {
	// copy for safety
	exts = append([]predict.Predicted{}, exts...)
	sort.Slice(exts, func(i, j int) bool {
		return exts[i].Prob > exts[j].Prob
	})

	if len(exts) > numExtsToShow {
		exts = exts[:numExtsToShow]
	}

	var labels, scores []string
	for _, ext := range exts {
		labels = append(labels, strings.Join(enc.DecodeToStrings(ext.TokenIDs), " "))
		scores = append(scores, fmt.Sprintf("%.3f", ext.Prob))
	}
	return labels, scores
}
