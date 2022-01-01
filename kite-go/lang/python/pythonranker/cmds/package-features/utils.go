package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sync"

	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/text"
)

const (
	alpha = 0.1
)

// loadPackageList expects a list that contains the list of packages
// that will be considered in package ranking.
func loadPackageList(path string) map[string]struct{} {
	log.Println(path)
	file, err := fileutil.NewCachedReader(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	list := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		list[scanner.Text()] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return list
}

// loadDocs loads data from the doc corpus.
func loadDocs(list map[string]struct{}, path string) (packageSels, packageDocs map[string][]string) {
	packageSels = make(map[string][]string)
	packageDocs = make(map[string][]string)

	modules := make(pythondocs.Modules)
	file, err := fileutil.NewCachedReader(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	decomp, err := gzip.NewReader(file)
	if err != nil {
		log.Fatal(err)
	}

	dec := json.NewDecoder(decomp)
	err = modules.Decode(dec)
	if err != nil {
		log.Fatal(err)
	}

	for p := range list {
		module, found := modules[p]
		if !found {
			log.Println("cannot find", p, "in doc modules")
			continue
		}
		var docs []string
		var sels []string
		categories := [][]*pythondocs.LangEntity{
			module.Classes,
			module.ClassMethods,
			module.ClassAttributes,
			module.Funcs,
			module.Vars,
			module.Exceptions,
			module.Unknown,
			[]*pythondocs.LangEntity{
				module.Documentation,
			},
		}
		// Go through docs for each category
		for _, cat := range categories {
			for _, d := range cat {
				sels = append(sels, d.Sel)
				docs = append(docs, d.Doc)
			}
		}
		if len(docs) == 0 {
			continue
		}
		packageDocs[p] = docs
		packageSels[p] = sels
	}
	return packageSels, packageDocs
}

// loadFromGraph loads the name of modules/methods in a package by exploring import graph.
// defaultSelectors are selector names found in the doc corpus.
func loadFromGraph(packages map[string]struct{}, defaultSelectors, defaultDocs map[string][]string) (map[string][]string, map[string][]string) {
	// load import graph
	graph, err := pythonimports.NewGraph(pythonimports.DefaultImportGraph)
	if err != nil {
		log.Fatal(err)
	}
	graphstrings, err := pythonimports.LoadGraphStrings(pythonimports.DefaultImportGraphStrings)
	if err != nil {
		log.Fatal(err)
	}
	packageSelectors := make(map[string][]string)
	packageDocs := make(map[string][]string)

	for p := range packages {
		var selectors []string
		// allocate space first so that it doesn't increase the space along the way
		// which may be time consuming.
		docs := make([]string, 0, 10000)

		err := graph.Walk(p, func(name string, node *pythonimports.Node) bool {
			if aux, ok := graphstrings[node.ID]; ok {
				docs = append(docs, aux.Docstring)
			}
			if node.Classification == pythonimports.Object || node.Classification == pythonimports.Function {
				tokens := node.CanonicalName.Parts
				if len(tokens) > 0 && tokens[0] == p {
					selectors = append(selectors, tokens[len(tokens)-1])
				}
			}
			return true
		})

		if err != nil {
			log.Println(err)
			selectors = defaultSelectors[p]
		}
		docs = append(docs, defaultDocs[p]...)
		packageSelectors[p] = text.Uniquify(selectors)
		packageDocs[p] = docs
	}
	return packageSelectors, packageDocs
}

// loadSoData loads stackoverflow data.
func loadSoData(path string) map[string][]string {
	in, err := fileutil.NewCachedReader(path)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	decomp, err := gzip.NewReader(in)
	if err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(decomp)
	var packageSuggestions map[string][]*curation.SuggestionScore
	err = decoder.Decode(&packageSuggestions)
	if err != nil {
		log.Fatal(err)
	}
	data := make(map[string][]string)
	for p, suggestions := range packageSuggestions {
		var docs []string
		for _, suggestion := range suggestions {
			docs = append(docs, suggestion.Query)
		}
		data[p] = docs
	}
	return data
}

// loadGithub loads the raw package stats we gather from github and computes
// the prior probability of each package.
// It anneals the package counts we get from github to smooth the prior distribution.
func loadGithub(path string, list map[string]struct{}, annealingTemp float64) map[string]float64 {
	f, err := awsutil.NewShardedFile(path)
	if err != nil {
		log.Fatalln("cannot open completions dataset:", err)
	}
	packageData := make(map[string]pythoncode.PackageStats)
	var m sync.Mutex
	err = awsutil.EMRIterateSharded(f, func(key string, value []byte) error {
		var stats pythoncode.PackageStats
		err := json.Unmarshal(value, &stats)
		if err != nil {
			return err
		}
		m.Lock()
		packageData[stats.Package] = stats
		m.Unlock()
		return nil
	})
	if err != nil {
		log.Fatalln("error reading completions:", err)
	}

	total := 0.0
	packagePrior := make(map[string]float64)
	for p := range list {
		if stats, exists := packageData[p]; exists && stats.Count != 0 {
			packagePrior[p] += float64(stats.Count)
		} else {
			packagePrior[p] += alpha
		}
	}
	for p := range list {
		packagePrior[p] = math.Pow(packagePrior[p], annealingTemp)
		total += packagePrior[p]
	}

	for p, count := range packagePrior {
		packagePrior[p] = count / total
	}
	return packagePrior
}

func saveToJSON(path string, data interface{}) error {
	fout, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fout.Close()

	encoder := json.NewEncoder(fout)
	err = encoder.Encode(data)
	if err != nil {
		return fmt.Errorf("not saving to path: %v", err)
	}
	return nil
}
