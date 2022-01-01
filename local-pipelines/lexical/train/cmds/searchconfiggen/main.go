package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
)

func main() {
	args := struct {
		Lang                 string
		Window               int
		TopK                 int
		TopP                 float32
		MinP                 float32
		BeamWidth            int
		Depth                int
		PrefixRegularization float32
		Out                  string
		ModelType            predict.ModelType
	}{
		Window:               400,
		TopK:                 10,
		TopP:                 1,
		MinP:                 0.02,
		BeamWidth:            5,
		Depth:                5,
		PrefixRegularization: 0.05,
		Out:                  "searchconfig.json",
	}
	arg.MustParse(&args)

	group := lexicalv0.MustLangGroupFromName(args.Lang)
	langLexer, err := lexicalv0.NewLexer(group.Lexer)
	if err != nil {
		log.Fatalln(err)
	}

	if group.Lexer == lang.Text && group.IsMultiLingual() {
		args.PrefixRegularization = 0.005
	}

	f, err := os.Create(args.Out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	config := predict.SearchConfig{
		Window:               args.Window,
		TopK:                 args.TopK,
		TopP:                 args.TopP,
		MinP:                 args.MinP,
		BeamWidth:            args.BeamWidth,
		Depth:                args.Depth,
		PrefixRegularization: args.PrefixRegularization,
		IdentTemperature:     1.000001,                  // default value, no-op
		LexicalTemperature:   1.000001,                  // default value, no-op
		NumLexicalTokens:     langLexer.NumTokens() + 1, // +1 for SOF token
	}

	if args.ModelType == predict.ModelTypePrefixSuffix {
		config.Window = 200
	}

	if err := json.NewEncoder(f).Encode(config); err != nil {
		log.Fatal(err)
	}
}
