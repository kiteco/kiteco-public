package inspect

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/text"
	"github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/utils"
)

var (
	predictorCache map[string]predict.Predictor
	paramsCache    = map[string]predict.HParams{}
	mutex          sync.Mutex
)

func init() {
	predictorCache = make(map[string]predict.Predictor)
}

// Sample represents a case to be inspected
type Sample struct {
	Query      Query
	Prediction Prediction
	Generation int
}

// Query represents a case where completions can be recommended
type Query struct {
	Path                      string
	Language                  lexicalv0.LangGroup
	Cursor                    string
	Code                      string
	ModelPath                 string
	Config                    predict.SearchConfig
	RemoveNumLinesAfterCursor int
	AllowContextAfterCursor   bool
}

// Prediction describes completions and how they were reached
// Duration isn't saved because it's too dependent on the machine
type Prediction struct {
	FinalPredictions []predict.Predicted
	Meta             predict.PredictionsMeta
	Duration         time.Duration `json:"-"`
}

// Inspect examines a Query by loading a Predictor and making a Prediction
func Inspect(query Query) (Sample, error) {
	if !admissibleCode(query.Path, query.Code, query.Cursor, query.Language) {
		return Sample{}, errors.New("Inadmissible code")
	}

	c, err := construct(query)
	if err != nil {
		return Sample{}, err
	}

	start := time.Now()
	res, err := c.predictor.Predict(kitectx.Background(), c.inputs)
	if err != nil {
		return Sample{}, err
	}
	duration := time.Since(start)

	return Sample{
		Query:      query,
		Generation: generation,
		Prediction: Prediction{
			FinalPredictions: res.Preds,
			Meta:             res.Meta,
			Duration:         duration,
		},
	}, nil
}

func admissibleCode(path, code, cursor string, language lexicalv0.LangGroup) bool {
	if len(code) > maxFileSize {
		return false
	}

	if utils.FilterFile(path, []byte(code)) {
		return false
	}

	if lexicalv0.WebGroup.Equals(language) {
		if strings.Contains(path, "assets") {
			return false
		}
	}

	if cursor == "" {
		return len(code) != 0
	}

	if strings.Count(code, cursor) != 1 {
		return false
	}
	if cursorInComment(code, cursor, path) {
		return false
	}

	if language.Lexer != lang.Text {
		line := cursorLine(code, cursor)
		if !containsLetter(line) {
			return false
		}
		if cursorInString(code, cursor) {
			return false
		}
	}
	_, err := process(path, code, cursor, language, 0)
	if err != nil {
		return false
	}
	return true
}

// GetEncoder returns an encoder to use for decoding a sample
func GetEncoder(sample Sample) (*lexicalv0.FileEncoder, error) {
	predictor, err := getPredictor(sample.Query)
	if err != nil {
		return nil, err
	}
	return predictor.GetEncoder(), nil
}

// GetParams returns the hparams for the sample
func GetParams(sample Sample) (predict.HParams, error) {
	mutex.Lock()
	defer mutex.Unlock()
	if params, ok := paramsCache[sample.Query.ModelPath]; ok {
		return params, nil
	}

	paramsPath := fileutil.Join(sample.Query.ModelPath, "config.json")
	params, err := predict.NewHParams(paramsPath)
	if err != nil {
		return predict.HParams{}, err
	}

	paramsCache[sample.Query.ModelPath] = params
	return params, nil
}

type construction struct {
	predictor predict.Predictor
	inputs    predict.Inputs
}

func construct(query Query) (construction, error) {
	predictor, err := getPredictor(query)
	if err != nil {
		return construction{}, err
	}

	inputs, err := process(query.Path, query.Code, query.Cursor, query.Language, query.RemoveNumLinesAfterCursor)
	if err != nil {
		return construction{}, err
	}

	inputs.SearchConfig = query.Config
	inputs.AllowContextAfterCursor = query.AllowContextAfterCursor

	return construction{
		predictor: predictor,
		inputs:    inputs,
	}, nil
}

func getPredictor(query Query) (predict.Predictor, error) {
	mutex.Lock()
	defer mutex.Unlock()
	serialized, err := json.Marshal(query.Config)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s %s", string(serialized), query.ModelPath)
	if predictor, ok := predictorCache[key]; ok {
		return predictor, nil
	}
	predictor, err := newPredictor(query)
	if err != nil {
		return nil, err
	}
	predictorCache[key] = predictor
	return predictor, nil
}

func newPredictor(query Query) (predict.Predictor, error) {
	predictor, err := predict.NewPredictor(query.ModelPath, query.Language)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting predictor")
	}
	predictor.SetStrictChecking(true)
	return predictor, nil
}

func process(path, code, cursor string, language lexicalv0.LangGroup, removeNumLinesAfterCursor int) (predict.Inputs, error) {
	parts := strings.Split(code, cursor)

	langLexer, err := lexicalv0.NewLexer(language.Lexer)
	if err != nil {
		return predict.Inputs{}, err
	}

	var newlinePos int
	if removeNumLinesAfterCursor > 0 {
		var newlineCount int
		for i, r := range parts[1] {
			if r == '\n' {
				newlinePos = i
				newlineCount++
			}
			if newlineCount > removeNumLinesAfterCursor {
				break
			}
		}
	}

	buffer := parts[0] + parts[1][newlinePos:]
	cursorPos := len(parts[0])
	sb := data.NewBuffer(buffer).Select(data.Selection{Begin: cursorPos, End: cursorPos})

	lexed, err := lexicalv0.LexSelectedBuffer(sb, lang.FromFilename(path), langLexer)
	if err != nil {
		return predict.Inputs{}, err
	}

	cursorContext, err := lexicalv0.FindContext(sb, lexed, langLexer)
	if err != nil {
		return predict.Inputs{}, err
	}

	return predict.Inputs{
		FilePath:        path,
		Tokens:          lexed,
		CursorTokenIdx:  cursorContext.LastTokenIdx + 1,
		Prefix:          cursorContext.Prefix,
		IncludeMetaInfo: true,
		Buffer:          sb,
	}, nil
}

func cursorLine(code string, cursor string) string {
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		if strings.Contains(line, cursor) {
			return strings.Replace(line, cursor, "", 1)
		}
	}
	return ""
}

func containsLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func cursorInString(code, cursor string) bool {
	quotemarks := []string{"`", `"`, "'"}
	beforeCursor := strings.Split(code, cursor)[0]
	for _, quotemark := range quotemarks {
		if strings.Count(beforeCursor, quotemark)%2 == 1 {
			return true
		}
	}
	return false
}

func selectedBufferFor(code, cursor string) data.SelectedBuffer {
	parts := strings.Split(code, cursor)
	switch len(parts) {
	case 2:
		return data.NewBuffer(parts[0] + parts[1]).Select(data.Selection{Begin: len(parts[0]), End: len(parts[1])})
	default:
		return data.NewBuffer(code).Select(data.Selection{Begin: len(code), End: len(code)})
	}
}

func cursorInComment(code, cursor, path string) bool {
	sb := selectedBufferFor(code, cursor)
	return text.CursorInComment(sb, lang.FromFilename(path))
}
