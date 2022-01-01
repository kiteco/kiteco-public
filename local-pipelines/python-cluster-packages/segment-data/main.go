package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/servercontext"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type appearance struct {
	Packages []string
}

// SampleTag implements pipeline.Sample
func (co appearance) SampleTag() {}

type coOccurrence struct {
	Pkg1 string
	Pkg2 string
}

type scoredCoOccurrence struct {
	Pkg1  string `json:"pkg1"`
	Pkg2  string `json:"pkg2"`
	Score int    `json:"score"`
}

type pkgFreq struct {
	Pkg  string
	Freq int
}

func extract(s pipeline.Sample) pipeline.Sample {
	ev := s.(pythonpipeline.AnalyzedEvent)
	imports := make(map[string]bool)

	pythonast.Inspect(ev.Context.AST, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}
		switch node := n.(type) {
		case *pythonast.ImportNameStmt:
			for _, n := range node.Names {
				pkg := n.External.Names[0].Ident.Literal
				imports[pkg] = true
			}
			return true
		case *pythonast.ImportFromStmt:
			// No leading dots, has package names
			if len(node.Dots) == 0 && node.Package != nil && len(node.Package.Names) > 0 {
				pkg := node.Package.Names[0].Ident.Literal
				imports[pkg] = true
			}
			return true
		default:
			return true
		}
	})
	importList := make([]string, 0, len(imports))
	for i := range imports {
		importList = append(importList, i)
	}

	return appearance{importList}
}

func main() {
	args := struct {
		Out         string
		Freq        string
		MaxEvents   int
		NumAnalysis int
		MinFreq     int
	}{
		MaxEvents:   500,
		NumAnalysis: 2,
		MinFreq:     10,
	}

	arg.MustParse(&args)

	startDate, err := analyze.ParseDate("2018-01-01")
	fail(err)
	endDate, err := analyze.ParseDate("2019-01-20")
	fail(err)

	recreator, err := servercontext.NewRecreator(servercontext.DefaultBucketsByRegion)
	fail(err)

	trackOpts := pythonpipeline.DefaultTrackingEventsOpts
	trackOpts.MaxEvents = args.MaxEvents
	trackOpts.ShardByUMF = true
	trackOpts.NumReaders = 2
	trackOpts.Logger = os.Stdout

	events := pythonpipeline.NewTrackingEvents(startDate, endDate, pythontracking.ServerSignatureFailureEvent, trackOpts)
	deduped := transform.NewFilter("deduped", pythonpipeline.DedupeEvents())
	analyzed := transform.NewOneInOneOut("analyzed", pythonpipeline.AnalyzeEvents(recreator, false))
	extracted := transform.NewOneInOneOut("extracted", func(s pipeline.Sample) pipeline.Sample {
		return extract(s)
	})

	var m sync.Mutex
	cooccurs := make(map[coOccurrence]int)
	freq := make(map[string]int)
	merged := dependent.NewFromFunc("merged", func(s pipeline.Sample) {
		m.Lock()
		defer m.Unlock()

		if s == nil {
			return
		}
		app := s.(appearance)
		for _, p1 := range app.Packages {
			freq[p1]++
			for _, p2 := range app.Packages {
				if p1 == p2 {
					continue
				}
				cooccurs[coOccurrence{p1, p2}]++
			}
		}
	})

	pm := pipeline.ParentMap{}
	pm.Chain(
		events,
		deduped,
		analyzed,
		extracted,
		merged,
	)

	pipe := pipeline.Pipeline{
		Name:    "python-type-induction-validation",
		Parents: pm,
		Sources: []pipeline.Source{events},
	}

	start := time.Now()
	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: args.NumAnalysis,
	})
	fail(err)

	_, err = engine.Run()
	fail(err)

	var results []scoredCoOccurrence
	for c, s := range cooccurs {
		results = append(results, scoredCoOccurrence{
			Pkg1:  c.Pkg1,
			Pkg2:  c.Pkg2,
			Score: s,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Pkg1 == results[j].Pkg1 {
			return results[i].Pkg2 < results[j].Pkg2
		}
		return results[i].Pkg1 < results[j].Pkg1
	})

	out, err := os.Create(args.Out)
	fail(err)
	defer out.Close()

	fail(json.NewEncoder(out).Encode(results))

	var p []pkgFreq

	for k, v := range freq {
		if v < args.MinFreq {
			delete(freq, k)
		} else {
			p = append(p, pkgFreq{k, v})
		}
	}

	sort.Slice(p, func(i, j int) bool {
		return p[i].Freq > p[j].Freq
	})

	ff, err := os.Create(args.Freq)
	fail(err)
	defer out.Close()
	for _, entry := range p {
		_, err := fmt.Fprintln(ff, entry)
		fail(err)
	}

	fmt.Printf("Done! took %v, extracted co-occurrence in segment data in %s.\n", time.Since(start), args.Out)
}
