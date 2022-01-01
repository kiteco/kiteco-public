package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
)

func main() {
	args := struct {
		Lang               string `json:"-"`
		VocabSize          int    `json:"n_vocab"`
		EmbeddingSize      int    `json:"n_embd"`
		ContextSize        int    `json:"n_ctx"`
		NumHeads           int    `json:"n_head"`
		NumLayers          int    `json:"n_layer"`
		NumPredictionSlots int    `json:"n_prediction_slots"`
		ModelType          string `json:"model_type"`
		NLangs             int    `json:"n_langs"`
		FullEmbdSize       int    `json:"n_full_embd"`
		Output             string `json:"-"`
	}{}
	arg.MustParse(&args)

	langGroup := lexicalv0.MustLangGroupFromName(args.Lang)
	// TODO: gross
	args.NLangs = len(langGroup.Langs)

	// TODO: hacky
	if args.NLangs > 1 && args.EmbeddingSize == 180 {
		args.FullEmbdSize = 3 * args.EmbeddingSize
	} else {
		args.FullEmbdSize = args.EmbeddingSize
	}

	langLexer, err := lexicalv0.NewLexer(langGroup.Lexer)
	if err != nil {
		log.Fatalln(err)
	}

	args.VocabSize += langLexer.NumTokens() + len(lexicalv0.ExtraTokens(langGroup))

	buf, err := json.Marshal(args)
	if err != nil {
		log.Fatalln(err)
	}

	err = ioutil.WriteFile(args.Output, buf, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
}
