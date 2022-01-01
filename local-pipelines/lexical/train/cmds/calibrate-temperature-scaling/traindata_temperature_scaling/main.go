package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sync/atomic"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

var (
	maxSizeBytes = 1 << 18 // 256kb

	maxNumSitesPerFile = 10
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

type siteType int

const (
	siteTypeIdent   = siteType(0)
	siteTypeLexical = siteType(1)

	numSiteTypes = 2
)

func (s siteType) String() string {
	switch s {
	case siteTypeIdent:
		return "siteTypeIdent"
	case siteTypeLexical:
		return "siteTypeLexical"
	default:
		panic(fmt.Sprintf("unknown site type %d", s))
	}
}

type site struct {
	Train bool `json:"-"`
	Label int  `json:"label"`

	// [vocab]
	TemperatureTypes []siteType `json:"temperature_types"`
	// [vocab]
	Logits []float32 `json:"logits"`

	Inputs predict.Inputs `json:"-"`
}

func (site) SampleTag() {}

func sites(predictor predict.Predictor, window int, path string, buf sample.ByteSlice) []pipeline.Sample {
	toks, err := predictor.GetEncoder().Lexer.Lex(buf)
	if err != nil {
		return []pipeline.Sample{pipeline.NewError("failed to lex file")}
	}

	var samples [numSiteTypes][]site

	for i, tok := range toks {
		bps := predictor.GetEncoder().EncodeTokens([]lexer.Token{tok})
		if len(bps) == 0 {
			continue
		}

		// we only predict the first bp, if it is a lexical token then this
		// is all we need, for idents the first BP is the most uncertain
		// (see calibration/main.go).
		site := site{
			Inputs: predict.Inputs{
				FilePath:       path,
				Tokens:         toks,
				CursorTokenIdx: i,
			},
			Label: bps[0],
		}
		_, isIdent := predictor.GetEncoder().Lexer.ShouldBPEEncode(tok)
		if isIdent {
			samples[siteTypeIdent] = append(samples[siteTypeIdent], site)
		} else {
			samples[siteTypeLexical] = append(samples[siteTypeLexical], site)
		}
	}

	for i, ss := range samples {
		if len(ss) == 0 {
			return []pipeline.Sample{pipeline.NewError(fmt.Sprintf("no sites found for type %v", siteType(i)))}
		}

		rand.Shuffle(len(ss), func(i, j int) {
			ss[i], ss[j] = ss[j], ss[i]
		})

		if len(ss) > maxNumSitesPerFile {
			samples[i] = ss[:maxNumSitesPerFile]
		}
	}

	// make sure dataset stays balanced
	min := len(samples[0])
	if m := len(samples[1]); m < min {
		min = m
	}

	var batch []pipeline.Sample
	for _, ss := range samples {
		ss = ss[:min]
		for _, s := range ss {
			batch = append(batch, s)
		}
	}

	return batch
}

func main() {
	args := struct {
		Lang          string
		ModelPath     string
		Search        string // TODO: kind of nasty, we need to include this to get the window size
		LocalDataRoot string
		TrainDir      string
		ValidateDir   string
		MaxFiles      int
		Seed          int64
	}{
		TrainDir:    "train_samples",
		ValidateDir: "validate_samples",
		MaxFiles:    10,
		Seed:        47,
	}
	arg.MustParse(&args)

	rand.Seed(args.Seed)

	tensorflow.SetTensorflowThreadpoolSize(runtime.NumCPU())

	fail(os.MkdirAll(args.TrainDir, os.ModePerm))
	fail(os.MkdirAll(args.ValidateDir, os.ModePerm))

	langGroup := lexicalv0.MustLangGroupFromName(args.Lang)

	predictor, err := predict.NewPredictor(args.ModelPath, langGroup)
	fail(err)

	search, err := predict.NewSearchConfig(args.Search)
	fail(err)

	// always shared because we get the logits directly and thus never do any filtering
	var temperatureTypes []siteType
	for i := 0; i < predictor.GetEncoder().Size(); i++ {
		if predictor.GetEncoder().IsLexical(i) {
			temperatureTypes = append(temperatureTypes, siteTypeLexical)
		} else {
			temperatureTypes = append(temperatureTypes, siteTypeIdent)
		}
	}

	src := func() pipeline.Source {
		gen, err := inspect.NewCodeGeneratorWithOpts(langGroup, langGroup.Lexer != lang.Text, "", args.LocalDataRoot, args.Seed)
		fail(err)

		var numFiles int64
		return source.Func("files", func() pipeline.Record {
			code, path, err := gen.Next()
			fail(err)

			if atomic.AddInt64(&numFiles, 1) >= int64(args.MaxFiles) {
				return pipeline.Record{}
			}
			return pipeline.Record{
				Key: path,
				Value: pipeline.Keyed{
					Key:    path,
					Sample: sample.ByteSlice([]byte(code)),
				},
			}
		})
	}()

	sites := transform.NewMap("sites", func(s pipeline.Sample) []pipeline.Sample {
		ks := s.(pipeline.Keyed)
		return sites(predictor, search.Window, ks.Key, ks.Sample.(sample.ByteSlice))
	})

	predictions := transform.NewOneInOneOut("predict", func(s pipeline.Sample) pipeline.Sample {
		site := s.(site)
		site.Inputs.SearchConfig = search
		logits, err := predictor.Logits(site.Inputs)
		fail(err)

		site.Logits = logits
		site.TemperatureTypes = temperatureTypes
		if rand.Float32() < .5 {
			site.Train = true
		}
		return site
	})

	wOpts := aggregator.DefaultWriterOpts
	wOpts.Compress = true
	wOpts.NumGo = 1
	wOpts.SamplesPerFile = 1000

	tw := aggregator.NewJSONWriter(wOpts, "train-sample-writer", args.TrainDir)
	vw := aggregator.NewJSONWriter(wOpts, "validate-sample-writer", args.ValidateDir)

	pm := make(pipeline.ParentMap)
	samples := pm.Chain(src, sites, predictions)
	pm.Chain(
		samples,
		transform.NewFilter("train-filter", func(s pipeline.Sample) bool {
			return s.(site).Train
		}),
		tw,
	)

	pm.Chain(
		samples,
		transform.NewFilter("validate-filter", func(s pipeline.Sample) bool {
			return !s.(site).Train
		}),
		vw,
	)

	pipe := pipeline.Pipeline{
		Name:    "calibrate-temperature-scaling-traindata",
		Parents: pm,
		Sources: []pipeline.Source{src},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: runtime.NumCPU(),
	})
	fail(err)

	_, err = engine.Run()
	fail(err)
}
