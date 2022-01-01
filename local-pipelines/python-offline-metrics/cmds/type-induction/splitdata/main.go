package main

import (
	"fmt"
	"log"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/serialization"
	"github.com/kiteco/kiteco/local-pipelines/python-offline-metrics/cmds/type-induction/data"
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func saveSamples(dir string, pkg string, samples []data.Sample) {
	outf := fileutil.Join(dir, fmt.Sprintf("%s.json", pkg))
	e, err := serialization.NewEncoder(outf)
	fail(err)
	defer e.Close()
	for _, s := range samples {
		fail(e.Encode(s))
	}
}

func getSamples(files []string, maxSamples int, saveDir string, minSamples int) map[string][]data.Sample {
	samplesByPkg := make(map[string][]data.Sample)
	saved := make(map[string]bool)
	for _, file := range files {
		err := serialization.Decode(file, func(sample *data.Sample) {
			pkg := sample.Pkg
			if saved[pkg] {
				return
			}
			if len(samplesByPkg[pkg]) < maxSamples {
				samplesByPkg[pkg] = append(samplesByPkg[pkg], *sample)
			} else {
				saveSamples(saveDir, pkg, samplesByPkg[pkg])
				saved[pkg] = true
				delete(samplesByPkg, pkg)
			}
		})
		fail(err)
	}

	for p, ss := range samplesByPkg {
		if len(ss) < minSamples {
			continue
		} else {
			saveSamples(saveDir, p, ss)
		}
	}

	return samplesByPkg
}

func main() {
	args := struct {
		SamplesAll string
		SamplesOut string
		MaxSamples int
		MinSamples int
	}{
		MaxSamples: 300000,
		MinSamples: 50,
		SamplesAll: pythoncode.TypeInductionTrainData,
	}
	arg.MustParse(&args)

	sampleFiles, err := aggregator.ListDir(args.SamplesAll)
	fail(err)

	getSamples(sampleFiles, args.MaxSamples, args.SamplesOut, args.MinSamples)
}
