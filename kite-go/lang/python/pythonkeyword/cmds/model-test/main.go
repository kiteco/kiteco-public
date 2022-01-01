//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/token"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"os"
	"sort"
	"strings"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonkeyword"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

var (
	scanOpts = pythonscanner.Options{
		ScanComments:  true,
		ScanNewLines:  true,
		KeepEOFIndent: true,
	}
)

type comparisonStruct struct {
	Features           [][]float32 `json:"features"`
	IsKeywordLogits    [][]float32 `json:"is_keyword_logits"`
	WhichKeywordLogits [][]float32 `json:"which_keyword_logits"`
	BatchFeatures      [][]int64   `json:"batch_features"`
}

func newComparisonStruct() *comparisonStruct {
	result := comparisonStruct{
		Features:           make([][]float32, 0),
		IsKeywordLogits:    make([][]float32, 0),
		WhichKeywordLogits: make([][]float32, 0),
		BatchFeatures:      make([][]int64, 0),
	}
	return &result
}

func init() {
	log.SetOutput(os.Stderr)
}

type keyVal struct {
	Key string
	Val float32
}

type result struct {
	Src      string
	Expected []pythonscanner.Token

	// most likely outputs of the classifier
	IsKeyword    bool
	WhichKeyword pythonscanner.Token // the most likely keyword

	// list of features and their values by name order
	Features []keyVal
	// list of contributions to the keyword class (vs name) ordered by decreasing value
	IsKeywordContributions []keyVal
	// list of contributions to the chosen keyword ordered by decreasing value
	WhichKeywordContributions    []keyVal
	ExpectedKeywordContributions []keyVal

	// the logits
	IsKeywordProb       float32
	KeywordProb         float32
	ExpectedKeywordProb float32

	IsKeywordClass    string
	WhichKeywordClass string

	Words string
}

func contribList(weights [][]float32, x [][]float32, outIdx int) []keyVal {
	var contribs []keyVal
	for feature, contrib := range contributions(weights, x, outIdx) {
		contribs = append(contribs, keyVal{Key: feature, Val: contrib})
	}
	sort.Slice(contribs, func(i, j int) bool { return math.Abs(float64(contribs[i].Val)) > math.Abs(float64(contribs[j].Val)) })
	return contribs
}

func newResult(ex example, model *pythonkeyword.Model, compStruct *comparisonStruct) (result, error) {
	src := []byte(ex.src)
	cursor := int64(len(src)) // cursor is always at the end

	words, err := pythonscanner.Lex([]byte(src), scanOpts)
	if err != nil {
		return result{}, fmt.Errorf("error getting results: %v", err)
	}

	tokCursor := token.Pos(cursor)
	parseOpts := pythonparser.Options{
		Approximate: true,
		Cursor:      &tokCursor,
	}

	ast, _ := pythonparser.ParseWords(kitectx.Background(), src, words, parseOpts)
	if ast == nil {
		return result{}, fmt.Errorf("unable to parse file: %v", err)
	}

	expected := ex.tokens
	if expected == nil {
		expected = []pythonscanner.Token{ex.token}
	}

	nodeCount := pythonast.CountNodes(ast)
	parentMap := pythonast.ConstructParentTable(ast, nodeCount)

	modelInputs := pythonkeyword.ModelInputs{
		Buffer:    src,
		Cursor:    cursor,
		AST:       ast,
		Words:     words,
		ParentMap: parentMap,
	}

	features, err := pythonkeyword.NewFeatures(kitectx.Background(), modelInputs, pythonkeyword.ModelLookback)
	if err != nil {
		log.Printf("problematic cursor: %d, buffer: %s", cursor, string(src))
		return result{}, fmt.Errorf("error creating model features: %v", err)
	}

	fetches := []string{
		"classifiers/is_keyword/logits",
		"classifiers/is_keyword/weights",
		"classifiers/which_keyword/logits",
		"classifiers/which_keyword/weights",
		"features/x",
	}

	out, err := model.GetFetches(features, fetches)

	isLogits := out["classifiers/is_keyword/logits"].([][]float32)
	isWeights := out["classifiers/is_keyword/weights"].([][]float32)
	whichLogits := out["classifiers/which_keyword/logits"].([][]float32)
	whichWeights := out["classifiers/which_keyword/weights"].([][]float32)
	x := out["features/x"].([][]float32)

	if compStruct != nil {
		compStruct.Features = append(compStruct.Features, x[0])
		compStruct.IsKeywordLogits = append(compStruct.IsKeywordLogits, isLogits[0])
		compStruct.WhichKeywordLogits = append(compStruct.WhichKeywordLogits, whichLogits[0])
		compStruct.BatchFeatures = append(compStruct.BatchFeatures, features.Vector())
	}

	var fv []keyVal
	for feature, val := range featureValues(x) {
		fv = append(fv, keyVal{Key: feature, Val: val})
	}
	sort.Slice(fv, func(i, j int) bool { return fv[i].Key < fv[j].Key })

	isKeyword := isLogits[0][1] >= 0.5
	var isCategory int
	if isKeyword {
		isCategory = 1
	}

	var whichCategory int
	var maxProb float32
	for k, p := range whichLogits[0] {
		if p > maxProb {
			whichCategory = k
			maxProb = p
		}
	}
	whichKeyword := pythonkeyword.KeywordCatToToken(whichCategory + 1)

	isContribs := contribList(isWeights, x, isCategory)
	whichContribs := contribList(whichWeights, x, whichCategory)
	var expectedContribs []keyVal

	if expected[0] != pythonscanner.Ident {
		expectedContribs = contribList(whichWeights, x, pythonkeyword.KeywordTokenToCat(expected[0])-1)
	}

	var w []string
	for _, word := range words {
		w = append(w, word.String())
	}
	wstr := strings.Join(w, ", ")

	var ecp float32
	if expected[0] != pythonscanner.Ident {
		ecp = whichLogits[0][pythonkeyword.KeywordTokenToCat(expected[0])-1]
	}

	res := result{
		Src:                          ex.src,
		Expected:                     expected,
		IsKeyword:                    isKeyword,
		WhichKeyword:                 whichKeyword,
		Features:                     fv,
		IsKeywordContributions:       isContribs,
		WhichKeywordContributions:    whichContribs,
		ExpectedKeywordContributions: expectedContribs,

		IsKeywordProb:       isLogits[0][1],
		KeywordProb:         whichLogits[0][whichCategory],
		ExpectedKeywordProb: ecp,

		Words: wstr,
	}

	res.IsKeywordClass = "incorrect"
	if isKeyword == (expected[0] != pythonscanner.Ident) {
		res.IsKeywordClass = "correct"
	}

	if expected[0] != pythonscanner.Ident {
		res.WhichKeywordClass = "incorrect"
		for _, t := range expected {
			if t == res.WhichKeyword {
				res.WhichKeywordClass = "correct"
				break
			}
		}
	}

	return res, nil
}

func main() {

	model, err := pythonkeyword.NewModel(pythonmodels.DefaultOptions.KeywordModelPath)
	targetFile := "/home/moe/test/result.html"
	if err != nil {
		log.Fatalf("couldn't load models: %v", err)
	}
	compResult := newComparisonStruct()
	stats := struct {
		Total        int
		TotalKeyword int
		// number of examples that correctly identified the most likely keyword
		CorrectWhichKeyword int
		// number of examples that correctly identified whether the keyword is a keyword
		CorrectIsKeyword int
	}{}

	var results []result

	for _, ex := range examples {
		res, err := newResult(ex, model, compResult)
		if err != nil {
			log.Printf("%v\n", err)
			continue
		}
		results = append(results, res)

		stats.Total++

		expectKeyword := res.Expected[0] != pythonscanner.Ident

		if res.IsKeyword == expectKeyword {
			stats.CorrectIsKeyword++
		}

		if expectKeyword {
			stats.TotalKeyword++
			for _, tok := range res.Expected {
				if res.WhichKeyword == tok {
					stats.CorrectWhichKeyword++
				}
			}
		}
	}

	content, err := json.MarshalIndent(*compResult, "", " ")
	if err != nil {
		log.Fatalf("error while marshalling results: %v", err)
	}
	_ = ioutil.WriteFile("/data/kite/keywords_model/result_golang.json", content, 0644)

	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	templates := templateset.NewSet(staticfs, "templates", template.FuncMap{})

	var buf bytes.Buffer
	err = templates.Render(&buf, "report.html", map[string]interface{}{
		"Results": results,
		"Stats":   stats,
	})

	if err != nil {
		log.Fatalf("error rendering template: %v", err)
	}

	if targetFile != "" {
		f, err := os.Create(targetFile)
		if err == nil {
			defer f.Close()
			f.WriteString(buf.String())
			f.Sync()
		}
	}
	fmt.Println(stats)
}
