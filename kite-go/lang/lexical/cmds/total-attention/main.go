package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/alexflint/go-arg"

	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

var (
	language  lexicalv0.LangGroup
	modelpath string
	params    predict.HParams
	config    = predict.SearchConfig{
		Window:    64,
		TopK:      10,
		TopP:      1,
		MinP:      0.02,
		BeamWidth: 5,
		Depth:     5,
	}

	cursor = "$"
)

func main() {
	args := struct {
		Language    string
		Local       bool
		Iters       int
		Context     int
		ModelConfig bool
		Shard       bool
	}{
		Language:    "",
		Local:       false,
		Iters:       10,
		Context:     0,
		ModelConfig: false,
	}
	arg.MustParse(&args)

	language = lexicalv0.MustLangGroupFromName(args.Language)

	modelOptions, err := lexicalmodels.GetDefaultModelOptions(language)
	if err != nil {
		log.Fatal(err)
	}
	modelpath = modelOptions.ModelPath
	if args.ModelConfig {
		config, err = predict.NewSearchConfigFromModelPath(modelpath)
		if err != nil {
			log.Fatal(err)
		}
	}
	if args.Context != 0 {
		config.Window = args.Context
	}
	paramsPath := fileutil.Join(modelpath, "config.json")
	params, err = predict.NewHParams(paramsPath)
	if err != nil {
		log.Fatal(err)
	}
	total := computeTotal(args.Local, args.Iters)
	showTotal(total)
}

func computeTotal(local bool, iters int) [][]inspect.Attention {
	// returned value has shape (NumLayers, NumHeads, Window, Window)
	codeGenerator, err := inspect.NewCodeGenerator(language, local, cursor)
	if err != nil {
		log.Fatal(err)
	}
	defer codeGenerator.Close()
	var total [][]inspect.Attention
	var samples float32
	for i := 0; i < iters; i++ {
		code, path, err := codeGenerator.Next()
		if err != nil {
			log.Fatal(err)
		}
		query := inspect.Query{
			Path:      path,
			Cursor:    cursor,
			ModelPath: modelpath,
			Code:      code,
			Config:    config,
			Language:  language,
		}
		sample, err := inspect.Inspect(query)
		if err != nil {
			log.Fatal(err)
		}
		attention, err := inspect.GetAttention(sample)
		if err != nil {
			log.Fatal(err)
		}
		samples++
		if samples == 1 {
			total = attention.Befores
			continue
		}
		addInPlace(total, attention.Befores)
	}
	normalizeInPlace(total, samples)
	return total
}

func showTotal(total [][]inspect.Attention) {
	var layerLabels, headLabels []string
	for a, layer := range total {
		if a == 0 {
			layerLabels = append(layerLabels, "")
			headLabels = append(headLabels, "tokens back")
		}
		layerLabels = append(layerLabels, "")
		headLabels = append(headLabels, "")
		for b := range layer {
			headLabels = append(headLabels, fmt.Sprintf("H%d", b))
			if b == 0 {
				layerLabels = append(layerLabels, fmt.Sprintf("L%d", a))
				continue
			}
			layerLabels = append(layerLabels, "")
		}
	}
	fmt.Println(strings.Join(layerLabels, ","))
	fmt.Println(strings.Join(headLabels, ","))
	for i := config.Window - 1; i >= 0; i-- {
		row := []string{fmt.Sprintf("%d", config.Window-i), ""}
		for a, layer := range total {
			if a != 0 {
				row = append(row, "")
			}
			for _, head := range layer {
				val := head[len(head)-1][i]
				rep := fmt.Sprintf("%.3f", val)
				row = append(row, rep)
			}
		}
		fmt.Println(strings.Join(row, ","))
	}
}

func valid(attention [][][][]float32) bool {
	a, b, c, d, err := getDimensions(attention)
	if err != nil {
		log.Fatal(err)
	}
	return a == params.NumLayers && b == params.NumHeads && c == config.Window && d == config.Window
}

func getDimensions(attention [][][][]float32) (int, int, int, int, error) {
	if empty(attention) {
		return 0, 0, 0, 0, errors.New("empty")
	}
	a := len(attention)
	b := len(attention[0])
	c := len(attention[0][0])
	d := len(attention[0][0][0])
	for _, layer := range attention {
		if len(layer) != b {
			return 0, 0, 0, 0, errors.New("non-uniform shape")
		}
		for _, head := range layer {
			if len(head) != c {
				return 0, 0, 0, 0, errors.New("non-uniform shape")
			}
			for _, row := range head {
				if len(row) != d {
					return 0, 0, 0, 0, errors.New("non-uniform shape")
				}
			}
		}
	}
	return a, b, c, d, nil
}

func empty(attention [][][][]float32) bool {
	if len(attention) == 0 {
		return true
	}
	if len(attention[0]) == 0 {
		return true
	}
	if len(attention[0][0]) == 0 {
		return true
	}
	if len(attention[0][0][0]) == 0 {
		return true
	}
	return false
}

func addInPlace(x, y [][]inspect.Attention) {
	for a, layer := range x {
		for b, head := range layer {
			for c, row := range head {
				for d := range row {
					x[a][b][c][d] += y[a][b][c][d]
				}
			}
		}
	}
}

func normalizeInPlace(x [][]inspect.Attention, y float32) {
	for a, layer := range x {
		for b, head := range layer {
			for c, row := range head {
				for d := range row {
					x[a][b][c][d] /= y
				}
			}
		}
	}
}

func sum(row []float32) float32 {
	var ans float32
	for _, v := range row {
		ans += v
	}
	return ans
}
