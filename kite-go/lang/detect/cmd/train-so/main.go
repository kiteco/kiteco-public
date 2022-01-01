package main

import (
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/detect"
	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/languagemodel"
	"github.com/kiteco/kiteco/kite-golib/text"
)

const (
	defaultSOSynonymsPath = "s3://kite-data/stackoverflow/synonyms.json"
	maxTriesExamples      = 1e7
)

var supportedLanguages = []string{
	"objective-c",
	"php",
	"ruby",
	"javascript",
	"python",
	"go",
	"java",
	"c++",
	"c",
	"bash",
}

// splitTags splits a string of SO tags into the individual tags.
func splitTags(str string) []string {
	var (
		tag  string
		tags []string
	)
	for _, ch := range str {
		c := string(ch)
		if c != "" && c != "<" && c != ">" && c != " " {
			tag = tag + c
		}
		if c == ">" && tag != "" {
			tags = append(tags, tag)
			tag = ""
		}
	}
	return tags
}

// langSynonyms creates a map from a language synonym to the canonical
// name for that language, see languageDetector.SupportedLanguages.
// synPath specifies the path to the stackoverflow tag synonyms map.
func langSynonyms(synPath string) map[string]string {
	f, err := fileutil.NewCachedReader(synPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	var syns map[string][]string
	err = decoder.Decode(&syns)
	if err != nil {
		log.Fatal(err)
	}

	langSyns := make(map[string]string)
	for _, lang := range supportedLanguages {
		names := syns[lang]

		// make sure each lang is its own synonym
		names = append(names, lang)

		// manually add some synonyms
		switch lang {
		case "go":
			names = append(names, "golang")
		}

		fmt.Printf("Synonyms for lang %s: %v \n", lang, names)

		for _, name := range names {
			langSyns[name] = lang
		}
	}
	return langSyns
}

func examplesForClass(nNeg, nPos int, classes [][]string, class string) map[int]struct{} {
	exampleIdxs := make(map[int]struct{})
	var nTries int
	// find negative examples
NegLoop:
	for len(exampleIdxs) <= nNeg {
		nTries++
		if nTries > maxTriesExamples {
			log.Printf("only able to get %d negative examples for class %s\n", len(exampleIdxs), class)
			break
		}
		idx := rand.Intn(len(classes))
		for _, c := range classes[idx] {
			if c == class {
				continue NegLoop
			}
		}
		exampleIdxs[idx] = struct{}{}
	}

	// find positive examples
	nTries = 0
	var pos int
	for len(exampleIdxs) <= nNeg+nPos {
		nTries++
		if nTries > maxTriesExamples {
			log.Printf("only able to get %d positive examples for class %s\n", pos, class)
			break
		}
		idx := rand.Intn(len(classes))
		for _, c := range classes[idx] {
			if c == class {
				if _, found := exampleIdxs[idx]; !found {
					exampleIdxs[idx] = struct{}{}
					pos++
				}
				break
			}
		}
	}

	return exampleIdxs
}

// This binary trains a language detector using StackOverflow pages.
func main() {
	var (
		pagesPath      string
		synPath        string
		outPath        string
		nTestData      int
		pctNegTestData float64
	)
	flag.StringVar(&pagesPath, "pages", "", "path to GOB pages dump (REQUIRED)")
	flag.StringVar(&synPath, "syns", defaultSOSynonymsPath, "path to SO synonyms")
	flag.StringVar(&outPath, "out", "", "out path to write language scorer to (REQUIRED)")
	flag.IntVar(&nTestData, "nTest", 1000, "num test examples to use to threshold each language detector (nonnegative)")
	flag.Float64Var(&pctNegTestData, "pctNeg", 0.5, "pct, in [0,1], of negative test examples to use to threshold each language detector")
	flag.Parse()

	if pagesPath == "" || outPath == "" {
		flag.Usage()
		log.Fatal("pages, out REQUIRED")
	}

	if pctNegTestData < 0 || pctNegTestData > 1 {
		flag.Usage()
		log.Fatal("pctNeg must be in [0,1]")
	}

	if nTestData < 0 {
		flag.Usage()
		log.Fatal("nTestData must be non negative")
	}

	start := time.Now()

	// map from synonym of a language to canonical name for language
	langSyns := langSynonyms(synPath)

	f, err := os.Open(pagesPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := gob.NewDecoder(f)

	tokenizer := text.NewHTMLTokenizer()

	// extract docs and classes
	var docs []string
	// languages present in each doc
	var classes [][]string
	for {
		var page stackoverflow.StackOverflowPage
		err = decoder.Decode(&page)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		// use tags for each post to extract languages in so page
		tags := text.Uniquify(splitTags(page.GetQuestion().GetPost().GetTags()))
		var class []string
		for _, tag := range tags {
			if lang, exists := langSyns[tag]; exists {
				class = append(class, lang)
			}
		}

		// no explicit language tags in page
		if len(class) == 0 {
			continue
		}

		doc := page.GetQuestion().GetPost().GetTitle()

		docs = append(docs, doc)
		classes = append(classes, class)
	}

	// train LM scorer
	scorer, err := languagemodel.TrainScorer(docs, classes, tokenizer.Tokenize)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Done training detectors, took %v\n", time.Since(start))

	if nTestData > len(docs) {
		log.Printf("%d documents in set, adjusting number of test examples from %d to %d \n", len(docs), nTestData, len(docs))
		nTestData = len(docs)
	}

	// establish detection thresholds for each language by doing a grid search over threshold values
	// to maximize the balanced accuracy = (.5*TP / (TP + FN)) + (0.5 *TN / (TN + FP)) for the detecor
	// TOOD(juan): dont train on "test" set?
	startThresh := time.Now()
	nNeg := int(pctNegTestData * float64(nTestData))
	nPos := nTestData - nNeg
	thresholds := make(map[string]float64)
	for _, lang := range supportedLanguages {
		exIdxs := examplesForClass(nNeg, nPos, classes, lang)
		var maxAcc, maxThresh float64
		for thresh := 0.0; thresh <= 1.0; thresh += .01 {
			var TP, TN, FP, FN float64
			for idx := range exIdxs {
				// check if positive or negative example
				var langPresent bool
				for _, class := range classes[idx] {
					if class == lang {
						langPresent = true
						break
					}
				}

				// run detector higher prob -> more likely doc contains lang
				tokens := tokenizer.Tokenize(docs[idx])
				estLangPresent := scorer.Posterior(tokens)[lang] > thresh

				switch {
				case langPresent && estLangPresent:
					// estimated lang was present and lang was present -> true positive
					TP += 1.0
				case langPresent && !estLangPresent:
					// estimated lang was not present and lang was present -> false negative
					FN += 1.0
				case !langPresent && estLangPresent:
					// estimated lang was present and lang was not present -> false positive
					FP += 1.0
				case !langPresent && !estLangPresent:
					// estimated lang was not present and lang was not present -> true negative
					TN += 1.0
				}
			}
			acc := 0.5*(TP/(TP+FN)) + 0.5*(TN/(TN+FP))
			if acc > maxAcc {
				maxAcc = acc
				maxThresh = thresh
			}
		}
		thresholds[lang] = maxThresh
		fmt.Printf("for lang %s, selected thresh %f with balanced accuracy %f \n", lang, maxThresh, maxAcc)
	}

	fmt.Printf("Done thresholding detectors, took %v\n", time.Since(startThresh))

	fout, err := os.Create(outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer fout.Close()

	encoder := gob.NewEncoder(fout)
	err = encoder.Encode(detect.Detector{
		Scorer:     scorer,
		LangSyns:   langSyns,
		Thresholds: thresholds,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Done! took %v \n", time.Since(start))
}
