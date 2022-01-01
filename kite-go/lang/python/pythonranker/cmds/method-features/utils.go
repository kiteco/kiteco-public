package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncuration"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

const (
	numNegExamples = 100

	curationPriorWeight = 0.3
	githubPriorWeight   = 0.6
	soPriorWeight       = 0.1
)

// loadRawPackageStats loads the raw package stats we gather from github.
func loadRawPackageStats(path string) map[string]pythoncode.PackageStats {
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
	return packageData
}

// loadGithub load package stats from the stats gathered from github.
func loadGithub(path string, packageToData map[string]*packageData, graph *pythonimports.Graph) (map[string]string, error) {
	// canonical packages are all lower-case
	canonicalPackages := make(map[string]string)
	packageStats := loadRawPackageStats(path)

	for p, stats := range packageStats {
		prior, err := pythoncode.NewPackagePrior(graph, stats)
		if err != nil {
			log.Println(err)
			continue
		}
		cn := strings.ToLower(p)
		canonicalPackages[cn] = p

		data := packageToData[cn]
		if data == nil {
			data = newPackageData()
			packageToData[cn] = data
		}
		chainedProb := prior.EntityChainedLogProbs()
		unchainedProb := prior.EntityLogProbs()

		githubChained := make(map[string]float64)
		githubUnchained := make(map[string]float64)

		// register names we've seen
		for m, logP := range chainedProb {
			id, err := data.findID(m, graph)
			if err != nil {
				continue
			}
			name, err := data.nameByID(id)
			if err != nil {
				return nil, err
			}
			if _, found := githubChained[name]; found {
				githubChained[name] = logSumExp([]float64{githubChained[name], logP})
				githubUnchained[name] = logSumExp([]float64{githubUnchained[name], unchainedProb[m]})
			} else {
				githubChained[name] = logP
				githubUnchained[name] = unchainedProb[m]
			}
		}
		data.combinePriors(githubChained, githubUnchained, githubPriorWeight)
	}
	return canonicalPackages, nil
}

// loadDocs load doc data. We go through the doc corpus and load doc data
// only for entities that were observed in the github, stackoverflow, and
// curation corpuses.
func loadDocs(path string, packageToData map[string]*packageData, graph *pythonimports.Graph) error {
	// load from doc corpus
	modules := make(pythondocs.Modules)
	file, err := fileutil.NewCachedReader(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decomp, err := gzip.NewReader(file)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(decomp)
	err = modules.Decode(dec)
	if err != nil {
		return err
	}

	for p, module := range modules {
		data := packageToData[strings.ToLower(p)]
		if data == nil {
			continue
		}
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

		// only get docs for entities that have been observed
		// in the github, so, or curation datasets.
		for _, cat := range categories {
			for _, entity := range cat {
				node, err := graph.Find(entity.Ident)
				if err != nil {
					continue
				}
				// if we haven't seen this entity, skip it.
				if _, found := data.idToNames[node.ID]; !found {
					continue
				}
				name, err := data.nameByID(node.ID)
				if err != nil {
					return err
				}
				data.docData[name] = append(data.docData[name], entity.Doc)
			}
		}
	}
	return nil
}

// loadDocStrings load doc strings from the import graph.
// Note that we only collect doc strings for the nodes that were seen in the other corpuses.
func loadDocStrings(packageToData map[string]*packageData,
	graph *pythonimports.Graph,
	graphstrings pythonimports.GraphStrings) error {
	for _, data := range packageToData {
		for id := range data.idToNames {
			node, found := graphstrings[id]
			if !found {
				return fmt.Errorf("cannot find node for id %d", id)
			}
			if node.Docstring != "" {
				name, err := data.nameByID(id)
				if err != nil {
					return err
				}
				data.docData[name] = append(data.docData[name], node.Docstring)
			}
		}
	}
	return nil
}

// loadStackoverflow load the stackoverflow output by extract_method_data
func loadStackoverflow(path string, packageToData map[string]*packageData,
	graph *pythonimports.Graph, canonicalPackages map[string]string) error {

	in, err := fileutil.NewCachedReader(path)
	if err != nil {
		return err
	}
	defer in.Close()

	decomp, err := gzip.NewReader(in)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(decomp)

	sodata := make(map[string]map[string][]string)
	err = decoder.Decode(&sodata)

	for p, mdata := range sodata {
		cn := strings.ToLower(p)

		data := packageToData[strings.ToLower(p)]
		if data == nil {
			log.Println("cannot find package in so:", p)
			data = newPackageData()
			packageToData[p] = data
		}
		identCounts := make(map[string]int)
		for m, d := range mdata {
			id, err := data.findID(m, graph)
			if err != nil {
				continue
			}
			name, err := data.nameByID(id)
			if err != nil {
				return err
			}
			data.soData[name] = append(data.soData[name], d...)
			identCounts[name] += len(d)
		}
		prior, err := pythoncode.NewPackagePriorFromUniqueNameCounts(cn, identCounts)
		if err != nil {
			log.Fatal(err)
		}
		data.combinePriors(prior.EntityChainedLogProbs(), prior.EntityLogProbs(), soPriorWeight)
	}
	return nil
}

// loadCuration load training data from the curation corpus
func loadCuration(path string, packageToData map[string]*packageData,
	graph *pythonimports.Graph, canonicalPackages map[string]string) error {

	pkgIdentCounts := make(map[string]map[string]int)

	snippets, attributes := loadAttributes(path)
	for id, atts := range attributes {
		pkg := strings.ToLower(snippets[id].Curated.Snippet.Package)
		if pkg == "beautifulsoup" {
			pkg = "bs4"
		}
		// data are indexed by lower case package names
		data := packageToData[pkg]
		if data == nil {
			log.Println("cannot find package in curation:", pkg)
			data = newPackageData()
			packageToData[pkg] = data
		}

		identCounts, found := pkgIdentCounts[pkg]
		if !found {
			identCounts = make(map[string]int)
			pkgIdentCounts[pkg] = identCounts
		}

		title := snippets[id].Curated.Snippet.Title
		for _, att := range atts {
			if att.Type == "" {
				continue
			}
			tokens := strings.Split(att.Type, ".")
			if pkg != strings.ToLower(tokens[0]) {
				continue
			}
			// find the canonical package name since the given one may be wrong
			cn, found := canonicalPackages[pkg]
			if found {
				tokens[0] = cn
			}
			typ := strings.Join(tokens, ".")

			id, err := data.findID(strings.Join([]string{typ, att.Attribute}, "."), graph)
			if err != nil {
				continue
			}
			name, err := data.nameByID(id)
			if err != nil {
				return err
			}
			data.curationData[name] = append(data.curationData[name], title)
			identCounts[name]++
		}
	}

	// register the prior distributions
	for pkg, identCounts := range pkgIdentCounts {
		data := packageToData[pkg]
		if data == nil {
			log.Fatal("data is nil, which should not happen.")
		}
		prior, err := pythoncode.NewPackagePriorFromUniqueNameCounts(canonicalPackages[pkg], identCounts)
		if err != nil {
			log.Fatal(err)
		}
		data.combinePriors(prior.EntityChainedLogProbs(), prior.EntityLogProbs(), curationPriorWeight)
	}

	return nil
}

// loadAttributes loads code snippets and their attributes.
func loadAttributes(path string) (map[int64]*pythoncuration.Snippet, map[int64][]pythoncuration.Attribute) {
	snippets := make(map[int64]*pythoncuration.Snippet)
	attributes := make(map[int64][]pythoncuration.Attribute)

	s3r, err := fileutil.NewCachedReader(path)
	if err != nil {
		log.Fatalf("error loading curated snippets from %s: %v\n", path, err)
	}

	r := awsutil.NewEMRReader(s3r)
	for {
		_, value, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var cs pythoncuration.AnalyzedSnippet
		err = json.Unmarshal(value, &cs)

		if err != nil {
			log.Fatal(err)
		}
		id := cs.Snippet.Curated.Snippet.SnapshotID
		if _, exists := attributes[id]; !exists {
			snippets[id] = cs.Snippet
			attributes[id] = cs.Attributes
		}
	}
	return snippets, attributes
}

// --

type trainingDatum struct {
	Query   string
	Method  string
	Package string
	Score   float64
}

func loadLabeledData(path string) (map[string][]*trainingDatum, error) {
	in, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	var labelData []*trainingDatum

	decoder := json.NewDecoder(in)
	err = decoder.Decode(&labelData)
	if err != nil {
		return nil, err
	}

	queryToData := make(map[string][]*trainingDatum)
	for _, ld := range labelData {
		queryToData[ld.Query] = append(queryToData[ld.Query], ld)
	}
	return queryToData, nil
}

// addNegativeData adds negative training data.
func addNegativeData(graph *pythonimports.Graph,
	packageToData map[string]*packageData, queryToData map[string][]*trainingDatum) error {

	for q, data := range queryToData {
		// find out what package and methods are observed
		seenPackageIdents := make(map[string]map[string]struct{})
		for _, d := range data {
			pkg := strings.ToLower(d.Package)
			seenIdents, found := seenPackageIdents[pkg]
			if !found {
				seenIdents = make(map[string]struct{})
				seenPackageIdents[pkg] = seenIdents
			}
			packageData, found := packageToData[pkg]
			if !found {
				log.Println("cannot find package data for ", pkg)
				continue
			}
			id, err := packageData.findID(d.Method, graph)
			if err != nil {
				log.Println("cannot find", d.Method, "in the graph:", err)
				continue
			}
			name, err := packageToData[pkg].nameByID(id)
			if err != nil {
				return fmt.Errorf("cannot find name for id: %d when adding negative data", id)
			}
			d.Method = name
			seenIdents[name] = struct{}{}
		}
		// for each package, add negative examples
		for pkg, idents := range seenPackageIdents {
			packageData, found := packageToData[pkg]
			if !found {
				continue
			}
			names := packageData.namespace()
			perm := rand.Perm(len(names))
			for i := 0; i < numNegExamples && i < len(perm); i++ {
				if _, found := idents[names[perm[i]]]; found {
					continue
				}
				queryToData[q] = append(queryToData[q], &trainingDatum{
					Query:   q,
					Method:  names[perm[i]],
					Package: pkg,
					Score:   0,
				})
			}
		}
	}
	return nil
}

func loadSet(path string) map[string]struct{} {
	fin, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	list := make(map[string]struct{})
	scanner := bufio.NewScanner(fin)
	for scanner.Scan() {
		list[scanner.Text()] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return list
}

func buildIndex(packageToData map[string]*packageData) map[string]*pythondocs.TermIndex {
	indexes := make(map[string]*pythondocs.TermIndex)
	for p, data := range packageToData {
		corpus := make(map[string][]string)
		for name, docs := range data.soData {
			corpus[name] = append(corpus[name], docs...)
		}
		for name, docs := range data.docData {
			corpus[name] = append(corpus[name], docs...)
		}
		for name, docs := range data.curationData {
			corpus[name] = append(corpus[name], docs...)
		}
		indexes[p] = pythondocs.NewTermIndex(corpus)
	}
	return indexes
}

// addSelectorName adds selector name to the docs of a method name.
// The number of selector names to insert is proportional to the
// number of docs we've seen for this method name (heuristic).
func addSelectorName(data map[string][]string) {
	for m, docs := range data {
		var pseudoWords []string
		parts := strings.Split(m, ".")
		sel := parts[len(parts)-1]
		for i := 0; i < len(docs); i++ {
			pseudoWords = append(pseudoWords, sel)
		}
		data[m] = append(data[m], pseudoWords...)
	}
}

// logSumExp receives a slice of log scores: log(a), log(b), log(c)...
// and returns log(a + b + c....)
func logSumExp(logs []float64) float64 {
	var max float64
	for _, l := range logs {
		if l > max {
			max = l
		}
	}
	var sum float64
	for _, l := range logs {
		sum += math.Exp(l - max)
	}
	return max + math.Log(sum)
}
