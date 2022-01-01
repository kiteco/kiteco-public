package main

import (
	"io/ioutil"
	"log"
	"math/rand"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/lexicalproviders"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/javascript"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/render"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

func fail(process string, err error) {
	if err != nil {
		log.Fatalf("Error %s while %s\n", err, process)
	}
}

var defaultMatchOption = render.MatchEnd

func main() {
	args := struct {
		TestFile         string
		SampleRate       float64
		PredictionLength int
		LogFile          string
	}{
		SampleRate:       0.1,
		PredictionLength: 5,
	}

	arg.MustParse(&args)
	buf, err := ioutil.ReadFile(args.TestFile)
	fail("reading file", err)
	lxr, err := lexicalv0.NewLexer(lang.JavaScript)
	fail("creating lexer", err)
	tokens, err := lxr.Lex(buf)
	fail("lexing", err)

	models, err := lexicalmodels.NewModels(lexicalmodels.DefaultModelOptions)
	fail("initializing models", err)
	global := lexicalproviders.Global{
		Models:   models,
		FilePath: args.TestFile,
		Product:  licensing.Pro,
	}

	log, err := os.Create(args.LogFile)
	fail("opening log file", err)
	defer log.Close()

	for i := range tokens {
		if i > len(tokens)-args.PredictionLength-1 || rand.Float64() >= args.SampleRate {
			continue
		}
		current := tokens[i]
		context := string(buf[:current.End])
		prediction := tokens[i+1 : i+1+args.PredictionLength]

		for i, p := range prediction {
			if _, ok := lxr.ShouldBPEEncode(p); ok {
				prediction[i].Token = lexer.BPEEncodedTok
			}
		}
		truth := string(buf[tokens[i+1].Start:tokens[i+args.PredictionLength].End])
		b := data.NewBuffer(string(buf)).Select(data.Selection{
			Begin: tokens[i+1].Start,
			End:   tokens[i+args.PredictionLength].End,
		})

		in, err := lexicalproviders.NewInputs(kitectx.Background(), global, b, false)
		fail("generating inputs", err)
		// Comment out the auto-closing parens in Render() before applying this - makes life easier
		snippet, ok := javascript.Render(in.LineContext, prediction, false, current.End < tokens[i+1].Start)

		if !ok {
			continue
		}
		c := data.Completion{
			Snippet: snippet,
			Replace: data.Selection{
				Begin: in.Selection.Begin - len(in.PredictInputs.Prefix),
				End:   in.Selection.End,
			},
		}

		formatted := javascript.FormatCompletion(in.Text(), c, javascript.DefaultPrettifyConfig, defaultMatchOption)
		if formatted.Text != truth {
			_, err := log.WriteString("================= CONTEXT ==============\n")
			if len(context) > 100 {
				context = context[len(context)-100:]
			}
			_, err = log.WriteString(context)
			_, err = log.WriteString("\n================ EXPECTED ==============\n")
			_, err = log.WriteString(truth)
			_, err = log.WriteString("\n=================== GOT ================\n")
			_, err = log.WriteString(formatted.Text)
			_, err = log.WriteString("\n")
			fail("writing log", err)
		}
	}
}
