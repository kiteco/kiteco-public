package minp

import (
	"math"
	"math/rand"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

const (
	cursor = "$"
	seed   = 0
)

var (
	// Args is used for collecting data
	Args = struct {
		Language   string
		Local      bool
		Iters      int
		MaxDepth   int
		CheckIdent bool
	}{
		Language:   "",
		Local:      false,
		Iters:      1000,
		MaxDepth:   1,
		CheckIdent: false,
	}

	language lexicalv0.LangGroup
)

// Data holds probabilities, grouped by depth
type Data map[int][]float32

// Collect Data from random samples
func Collect(modelpath string, config predict.SearchConfig) (Data, error) {
	rand.Seed(seed)

	language = lexicalv0.MustLangGroupFromName(Args.Language)
	codeGenerator, err := inspect.NewCodeGenerator(language, Args.Local, cursor)
	if err != nil {
		return Data{}, err
	}
	defer codeGenerator.Close()
	d := make(Data)
	for i := 0; i < Args.Iters; i++ {
		code, path, err := codeGenerator.Next()
		if err != nil {
			return Data{}, err
		}
		for depth := 1; depth <= Args.MaxDepth; depth++ {
			query := createQuery(code, path, depth, modelpath, config)
			sample, err := inspect.Inspect(query)
			if err != nil {
				return Data{}, err
			}
			err = d.addHit(depth, sample)
			if err != nil {
				return Data{}, err
			}
		}
	}
	for k := range d {
		sort.Slice(d[k], func(i, j int) bool { return d[k][i] < d[k][j] })
	}
	return d, nil
}

func createQuery(code, path string, depth int, modelpath string, config predict.SearchConfig) inspect.Query {
	config.Depth = depth
	return inspect.Query{
		Path:      path,
		Cursor:    cursor,
		ModelPath: modelpath,
		Code:      code,
		Config:    config,
		Language:  language,
	}
}

func (d Data) addHit(depth int, sample inspect.Sample) error {
	matches, err := inspect.Matches(sample)
	if err != nil {
		return err
	}
	encoder, err := inspect.GetEncoder(sample)
	if err != nil {
		return err
	}
	for i, pred := range sample.Prediction.FinalPredictions {
		decoded := encoder.Decode(pred.TokenIDs)
		firstIsIdent := len(decoded) > 0 && encoder.Lexer.IsType(lexer.IDENT, decoded[0])
		if Args.CheckIdent && !firstIsIdent {
			continue
		}
		if matches[i] {
			d[depth] = append(d[depth], pred.Prob)
			return nil
		}
	}
	return nil
}

// GetPercentile computes an approximate percentile
func GetPercentile(values []float32, pct float64) float32 {
	if pct == 1 {
		return values[len(values)-1]
	}
	idx := int(math.Round(pct * float64(len(values))))
	return values[idx]
}
