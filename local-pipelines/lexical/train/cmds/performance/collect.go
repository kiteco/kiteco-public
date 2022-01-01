package main

import (
	"path/filepath"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/lexicalv0/inspect"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/performance"
)

type extToPredictionSites map[string]performance.PredictionSites
type pathToPredictionSites map[string]performance.PredictionSites
type minNumSites map[performance.Measurement]int

func enoughSamples(allExts map[string]bool, minNum minNumSites, eps extToPredictionSites) bool {
	for ext := range allExts {
		for m, n := range minNum {
			if len(eps[ext][m]) < n {
				return false
			}
		}
	}
	return true
}

func useFile(ext string, minNum minNumSites, eps extToPredictionSites) bool {
	for m, n := range minNum {
		if len(eps[ext][m]) < n {
			return true
		}
	}
	return false
}

func collectSites(gen inspect.CodeGenerator, extractor performance.Extractor, minNum minNumSites, allExts map[string]bool) pathToPredictionSites {
	eps := make(extToPredictionSites)
	for e := range allExts {
		eps[e] = make(performance.PredictionSites)
	}
	for {
		if enoughSamples(allExts, minNum, eps) {
			break
		}
		code, path, err := gen.Next()
		fail(err)
		ext := filepath.Ext(path)
		if len(ext) == 0 {
			continue
		}
		ext = ext[1:]
		if !allExts[ext] {
			continue
		}
		if !useFile(ext, minNum, eps) {
			continue
		}

		currentSites, err := extractor.ExtractPredictionSites([]byte(code), path, 8)
		if err != nil {
			continue
		}
		for m, pss := range currentSites {
			eps[ext][m] = append(eps[ext][m], pss...)
		}
	}
	// Shuffle and truncate prediction sites for each extension and measurement to be the minimum
	// Then group them by path to feed into evaluator
	pathToSites := make(pathToPredictionSites)
	for _, pss := range eps {
		// Sort the measurements to get deterministic results
		var ms []performance.Measurement
		for m := range pss {
			ms = append(ms, m)
		}
		sort.Slice(ms, func(i, j int) bool {
			return string(ms[i]) < string(ms[j])
		})
		for _, m := range ms {
			ps := pss[m]
			extractor.Rand.Shuffle(len(pss), func(i, j int) {
				ps[i], ps[j] = ps[j], ps[i]
			})
			truncated := ps[:minNum[m]]
			for _, p := range truncated {
				if pathToSites[p.Path] == nil {
					pathToSites[p.Path] = make(performance.PredictionSites)
				}
				pathToSites[p.Path][m] = append(pathToSites[p.Path][m], p)
			}
		}
	}
	return pathToSites
}
