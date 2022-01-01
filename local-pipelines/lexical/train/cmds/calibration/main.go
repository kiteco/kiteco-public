package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/lexer"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

const (
	maxSamplesPerFile = 10
	numBins           = 10
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

type bins struct {
	m sync.Mutex
	// Counts is the number of elements in each bin
	Counts []float64
	// Hits is the number of hits in each bin (e.g correct labels)
	Hits []float64
	// Confidences is a sum of the confidences for each bin
	Confidences []float64
}

func newBins(numBins int) *bins {
	return &bins{
		Counts:      make([]float64, numBins),
		Hits:        make([]float64, numBins),
		Confidences: make([]float64, numBins),
	}
}

func (b *bins) Count(confidence float64, isHit bool) {
	b.m.Lock()
	defer b.m.Unlock()

	width := 1. / float64(len(b.Counts))

	bin := int(math.Floor(confidence / width))

	if bin >= len(b.Counts) {
		// if confidence is 1 then we can end up outside our histogram so map
		// to the last bin, e.g if confidence = 1 and numBins = 10
		// then int(1/.1) = 10
		bin = len(b.Counts) - 1
	}

	b.Counts[bin]++
	b.Confidences[bin] += confidence
	if isHit {
		b.Hits[bin]++
	}
}

func (b *bins) ECE() float64 {
	var numSamples float64
	for _, c := range b.Counts {
		numSamples += c
	}

	accs, confs := b.AccuracyAndConfidence()

	var v float64
	for i, count := range b.Counts {
		v += (count / numSamples) * math.Abs(accs[i]-confs[i])
	}
	return v
}

func (b *bins) AccuracyAndConfidence() ([]float64, []float64) {
	var accs, confs []float64
	for i, count := range b.Counts {
		if count == 0 {
			// count is 0 so conf and hits are 0 so just set count to 1
			count = 1
		}
		accs = append(accs, b.Hits[i]/count)
		confs = append(confs, b.Confidences[i]/count)
	}
	return accs, confs
}

func (b *bins) PlotReliability(file string, title string) {

	accs, confs := b.AccuracyAndConfidence()
	width := 1. / float64(len(b.Counts))

	var pts, perfect plotter.XYs
	for i, acc := range accs {
		pts = append(pts, plotter.XY{
			X: confs[i],
			Y: acc,
		})
		perfect = append(perfect, plotter.XY{
			X: float64(i) * width,
			Y: float64(i) * width,
		})
	}

	// append 1,1 to perfect so charts are always
	// go from 0 to 1 on both axes
	perfect = append(perfect, plotter.XY{
		X: 1.,
		Y: 1.,
	})

	p, err := plot.New()
	fail(err)

	p.Title.Text = title
	p.X.Label.Text = "Confidence"
	p.Y.Label.Text = "Accuracy"

	p.Add(plotter.NewGrid())

	fail(plotutil.AddLinePoints(p,
		"Perfect", perfect,
		"Actual", pts,
	))

	fail(p.Save(4*vg.Inch, 4*vg.Inch, file))
}

type siteType int

const (
	siteTypeAny        = siteType(0)
	siteTypeIdentStart = siteType(1)
	siteTypeIdentRest  = siteType(2)
	siteTypeLexical    = siteType(3)

	numSiteTypes = 4
)

func (s siteType) String() string {
	switch s {
	case siteTypeAny:
		return "siteTypeAny"
	case siteTypeIdentStart:
		return "siteTypeIdentStart"
	case siteTypeIdentRest:
		return "siteTypeIdentRest"
	case siteTypeLexical:
		return "siteTypeLexical"
	default:
		return "siteTypeUnknown"
	}
}

type site struct {
	Type   siteType
	Inputs predict.Inputs
	Label  []int
}

func (site) SampleTag() {}

func sites(predictor predict.Predictor, path string, buf sample.ByteSlice) []pipeline.Sample {
	toks, err := predictor.GetEncoder().Lexer.Lex(buf)
	if err != nil {
		return []pipeline.Sample{pipeline.NewError("failed to lex file")}
	}

	// reservoir sampling
	var samples [numSiteTypes][]pipeline.Sample
	var counts [numSiteTypes]int
	maybeAppend := func(t siteType, s site) {
		s.Type = t

		counts[t]++

		if len(samples[t]) < maxSamplesPerFile {
			samples[t] = append(samples[t], s)
		} else {
			idx := rand.Intn(counts[t])
			if idx < maxSamplesPerFile {
				samples[t][idx] = s
			}
		}
	}

	for cursorIdx, tok := range toks {
		bps := predictor.GetEncoder().EncodeTokens([]lexer.Token{tok})

		_, isIdent := predictor.GetEncoder().Lexer.ShouldBPEEncode(tok)
		for bpIdx, bp := range bps {
			var prefix string
			if decoded := predictor.GetEncoder().DecodeToStrings(bps[:bpIdx]); len(decoded) > 0 {
				prefix = decoded[0]
			}

			site := site{
				Inputs: predict.Inputs{
					FilePath:       path,
					Tokens:         toks,
					CursorTokenIdx: cursorIdx,
				},
				Label: []int{bp}, // may be modified below
			}

			switch {
			case isIdent && bpIdx == 0:
				maybeAppend(siteTypeIdentStart, site)
			case isIdent && bpIdx > 0:
				// Give the model the prefix, since we use the prefix
				// as a filter we need to give the model the full label
				// up to the current BP as prefix and then ask it to predict the next bp.
				site.Inputs.Prefix = prefix
				site.Label = bps[:bpIdx+1]
				maybeAppend(siteTypeIdentRest, site)
			case !isIdent:
				maybeAppend(siteTypeLexical, site)
			}
			maybeAppend(siteTypeAny, site)
		}
	}

	var res []pipeline.Sample
	for _, ss := range samples {
		if len(ss) == 0 {
			continue
		}
		res = append(res, ss...)
	}

	if len(res) == 0 {
		res = append(res, pipeline.NewError("no sites found"))
	}

	return res
}

func main() {
	args := struct {
		Lang          string
		ModelPath     string
		Search        string
		OutDir        string
		LocalDataRoot string
		MaxFiles      int
		Seed          int64
	}{
		MaxFiles: 500,
		Seed:     42,
	}
	arg.MustParse(&args)

	rand.Seed(args.Seed)

	tensorflow.SetTensorflowThreadpoolSize(runtime.NumCPU())

	langGroup := lexicalv0.MustLangGroupFromName(args.Lang)

	predictor, err := predict.NewPredictor(args.ModelPath, langGroup)
	fail(err)

	predictor.SetStrictChecking(true)

	search, err := predict.NewSearchConfig(args.Search)
	fail(err)

	files := func() pipeline.Source {
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

	start := time.Now()

	samples := transform.NewMap("samples", func(in pipeline.Sample) []pipeline.Sample {
		ks := in.(pipeline.Keyed)
		buf := ks.Sample.(sample.ByteSlice)

		return sites(predictor, ks.Key, buf)
	})

	var allBins []*bins
	for i := 0; i < numSiteTypes; i++ {
		allBins = append(allBins, newBins(numBins))
	}

	predictions := transform.NewOneInOneOut("predict", func(in pipeline.Sample) pipeline.Sample {
		s := in.(site)

		updatedSearch := search
		updatedSearch.Depth = len(s.Label)
		s.Inputs.SearchConfig = updatedSearch
		res, err := predictor.Predict(kitectx.Background(), s.Inputs)
		if err == predict.ErrUnableToReserveSlots {
			return pipeline.NewError("unable to reserve slots for prediction")
		}
		fail(err)

		if len(res.Preds) == 0 {
			return pipeline.NewError("no predictions")
		}

		bin := allBins[s.Type]
		bin.Count(float64(res.Preds[0].Prob), reflect.DeepEqual(res.Preds[0].TokenIDs, s.Label))
		return nil
	})

	pm := pipeline.ParentMap{}

	pm.Chain(files, samples, predictions)

	pipe := pipeline.Pipeline{
		Name:    "lexical-calibration",
		Parents: pm,
		Sources: []pipeline.Source{files},
	}

	opts := pipeline.DefaultEngineOptions
	opts.NumWorkers = runtime.NumCPU()

	engine, err := pipeline.NewEngine(pipe, opts)
	fail(err)

	_, err = engine.Run()
	fail(err)

	fail(os.MkdirAll(args.OutDir, os.ModePerm))

	outPath := path.Join(args.OutDir, "misc.txt")
	f, err := os.Create(outPath)
	fail(err)
	defer f.Close()

	fmt.Fprintf(f, "model_path: %s\n", args.ModelPath)
	fmt.Fprintf(f, "search params: %+v\n", search)

	printBins := func(prefix string, bins *bins) {
		fmt.Fprintf(f, "%s_expected_calibration_error: %f\n", prefix, bins.ECE())

		accs, confs := bins.AccuracyAndConfidence()
		fmt.Fprintf(f, "%s_accuarcies\n", prefix)
		for _, acc := range accs {
			fmt.Fprintf(f, "%f\n", acc)
		}

		fmt.Fprintf(f, "%s_confidences\n", prefix)
		for _, conf := range confs {
			fmt.Fprintf(f, "%f\n", conf)
		}

		pltPath := path.Join(args.OutDir, fmt.Sprintf("%s_reliability.png", prefix))
		bins.PlotReliability(pltPath, fmt.Sprintf("Reliability for %s", prefix))
	}

	for i := siteType(0); i < numSiteTypes; i++ {
		printBins(i.String(), allBins[i])
	}

	fmt.Println("Done, took", time.Since(start))
}
