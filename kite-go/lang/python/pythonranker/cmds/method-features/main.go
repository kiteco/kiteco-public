package main

import (
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonranker"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonranker/internal/precompute"
	"github.com/kiteco/kiteco/kite-go/ranking"
	"github.com/kiteco/kiteco/kite-golib/hash"
	"github.com/kiteco/kiteco/kite-golib/text"
)

// This binary builds a method featurizer for each package we have data for.
// We collect data from the github, curation, so, doc corpuses.

var (
	defaultDocPath              = "s3://kite-emr/datasets/documentation/python/2015-07-31_14-22-04-PM/python.json.gz"
	defaultGithubPath           = "s3://kite-emr/users/tarak/python-code-examples/2015-10-21_13-13-06-PM/merge_package_stats/output"
	defaultSoPath               = "s3://kite-emr/datasets/stackoverflow/2015-10-25_21-18-03-PM/package-method-data.json.gz"
	defaultGraphPath            = pythonimports.DefaultImportGraph
	defaultGraphStringsPath     = pythonimports.DefaultImportGraphStrings
	defaultArgSpecsPath         = pythonimports.DefaultImportGraphArgSpecs
	defaultTypeshedArgSpecsPath = pythonimports.DefaultTypeshedArgSpecs
	defaultCurationPath         = "s3://kite-emr/datasets/curated-snippets/2015-09-29_15-30-27-PM/curated-snippets-attributes.emr"
)

func main() {
	var (
		curationPath         string
		docPath              string
		graphPath            string
		graphStringsPath     string
		graphArgSpecsPath    string
		typeshedArgSpecsPath string
		githubPath           string
		soPath               string

		dir         string
		testQueries string
		input       string
	)

	flag.StringVar(&docPath, "doc", defaultDocPath, "default path to the doc corpus")
	flag.StringVar(&githubPath, "github", defaultGithubPath, "default path to the file that contains github stats")
	flag.StringVar(&soPath, "so", defaultSoPath, "default path to the so data")
	flag.StringVar(&graphPath, "graph", defaultGraphPath, "default path to the import graph")
	flag.StringVar(&graphStringsPath, "graphstrings", defaultGraphStringsPath, "default path to the import graph strings")
	flag.StringVar(&graphArgSpecsPath, "gaphargspecs", defaultArgSpecsPath, "default path to the import graph arg specs")
	flag.StringVar(&typeshedArgSpecsPath, "typeshedargspecs", defaultTypeshedArgSpecsPath, "default path to the typeshed arg specs")
	flag.StringVar(&curationPath, "curation", defaultCurationPath, "default path to the curated examples")

	flag.StringVar(&input, "input", "", "input data that contains raw queries (output of data-prep)")
	flag.StringVar(&dir, "dir", "", "directory where the output file should go")
	flag.StringVar(&testQueries, "test", "", "list of queries for the test set")

	flag.Parse()

	if dir == "" || testQueries == "" || input == "" {
		flag.Usage()
		log.Fatal("must specify --dir --input --test")
	}

	// build the import graph client. We use the graph to
	// unify the reference names of methods in a package.
	graph, err := pythonimports.NewGraph(graphPath)
	if err != nil {
		log.Fatal(err)
	}

	graphstrings, err := pythonimports.LoadGraphStrings(graphStringsPath)
	if err != nil {
		log.Fatal(err)
	}

	argspecs, err := pythonimports.LoadArgSpecs(graph, graphArgSpecsPath, typeshedArgSpecsPath)
	if err != nil {
		log.Fatal(err)
	}

	// place holder for the data
	packageToData := make(map[string]*packageData)

	// load github stats
	log.Println("loading github stats...")
	canonicalPackages, err := loadGithub(githubPath, packageToData, graph)
	if err != nil {
		log.Fatal(err)
	}

	// load data from the curation corpus
	log.Println("loading curation corpus...")
	err = loadCuration(curationPath, packageToData, graph, canonicalPackages)
	if err != nil {
		log.Fatal(err)
	}

	// load data from the stackoverflow corpus
	log.Println("loading so data...")
	err = loadStackoverflow(soPath, packageToData, graph, canonicalPackages)
	if err != nil {
		log.Fatal(err)
	}

	// load documentation from the docs corpus
	log.Println("loading documentation...")
	err = loadDocs(docPath, packageToData, graph)
	if err != nil {
		log.Fatal(err)
	}

	// load doc strings from the graph for entities that
	// we've observed so far.
	log.Println("loading doc strings...")
	err = loadDocStrings(packageToData, graph, graphstrings)
	if err != nil {
		log.Fatal(err)
	}

	// build indices
	indices := buildIndex(packageToData)

	// load training set
	testSet := loadSet(testQueries)

	// with the data loaded, we can start training featurers
	// for each package.
	featurers := make(map[string]*pythonranker.MethodFeaturer)

	for p, data := range packageToData {
		alternatives := make(map[string][]string)
		for id, names := range data.idToNames {
			name, err := data.nameByID(id)
			if err != nil {
				log.Fatal(err)
			}
			alternatives[name] = text.Uniquify(names)
		}

		addSelectorName(data.docData)
		addSelectorName(data.soData)
		addSelectorName(data.curationData)

		data.alternativeNames = alternatives
		data.buildKeywordArgs(graph, argspecs)

		featurer := pythonranker.NewMethodFeaturer(data.githubData,
			data.githubDataChained, data.alternativeNames,
			data.docData, data.soData, data.curationData,
			data.kwargs, p)
		featurers[p] = featurer
	}
	log.Printf("built featurers for %d packages\n", len(packageToData))

	// load raw data for training the ranker
	log.Println("loading labeled data...")
	queryToData, err := loadLabeledData(input)
	if err != nil {
		log.Fatal(err)
	}

	// augment with negative training samples.
	log.Println("adding negative data...")
	err = addNegativeData(graph, packageToData, queryToData)
	if err != nil {
		log.Fatal(err)
	}

	// convert to format that the training script takes
	var numTrain int
	var numTest int

	var trainingData []ranking.Entry
	var testData []ranking.Entry

	for q, data := range queryToData {
		var rankingData *[]ranking.Entry
		if _, found := testSet[q]; found {
			numTest++
			rankingData = &testData
		} else {
			numTrain++
			rankingData = &trainingData
		}
		seenPackageIdents := make(map[string][]string)
		for _, d := range data {
			seenPackageIdents[d.Package] = append(seenPackageIdents[d.Package], d.Method)
		}
		for i, d := range data {
			featurer, found := featurers[d.Package]
			if !found {
				continue
			}
			queryStats := precompute.NewQueryStats(q, d.Package)
			targetStats := precompute.NewTargetStats(seenPackageIdents[d.Package])

			feats := featurer.Features(d.Method, queryStats, targetStats)
			for _, f := range feats {
				if math.IsInf(f, 0) {
					log.Fatal("feature is inf:", f)
				}
			}

			*rankingData = append(*rankingData, ranking.Entry{
				SnapshotID: int64(i),
				Label:      d.Score,
				QueryHash:  hash.SpookyHash128String([]byte(q + " " + d.Package)),
				QueryCode:  d.Method,
				QueryText:  q,
				Features:   feats,
			})
		}
	}

	log.Printf("Loaded %d training queries.\n", numTrain)
	log.Printf("Loaded %d test queries.\n", numTest)

	trainPayload := map[string]interface{}{
		"FeatureLabels":   featurers["numpy"].Labels(),
		"Data":            trainingData,
		"FeaturerOptions": nil,
	}
	testPayload := map[string]interface{}{
		"FeatureLabels":   featurers["numpy"].Labels(),
		"Data":            testData,
		"FeaturerOptions": nil,
	}

	ftrain, err := os.Create(path.Join(dir, "train-data.json"))
	if err != nil {
		log.Fatal(err)
	}
	defer ftrain.Close()

	jsonEncoder := json.NewEncoder(ftrain)
	err = jsonEncoder.Encode(trainPayload)
	if err != nil {
		log.Fatal(err)
	}

	ftest, err := os.Create(path.Join(dir, "test-data.json"))
	if err != nil {
		log.Fatal(err)
	}
	defer ftest.Close()

	jsonEncoder = json.NewEncoder(ftest)
	err = jsonEncoder.Encode(testPayload)
	if err != nil {
		log.Fatal(err)
	}

	// save the featurers
	ffeat, err := os.Create(path.Join(dir, "featurer.gob.gz"))
	if err != nil {
		log.Fatal(err)
	}
	defer ffeat.Close()

	compressor := gzip.NewWriter(ffeat)
	encoder := gob.NewEncoder(compressor)
	err = encoder.Encode(featurers)
	if err != nil {
		log.Fatal(err)
	}
	compressor.Flush()
	compressor.Close()

	// save the indexes
	findex, err := os.Create(path.Join(dir, "index.json"))
	if err != nil {
		log.Fatal(err)
	}
	defer findex.Close()

	jsonEncoder = json.NewEncoder(findex)
	err = jsonEncoder.Encode(indices)
	if err != nil {
		log.Fatal(err)
	}

}

// packageData contains all data relevant to a package
type packageData struct {
	idToNames map[int64][]string
	nameToID  map[string]int64

	alternativeNames  map[string][]string
	githubData        map[string]float64
	githubDataChained map[string]float64
	soData            map[string][]string
	docData           map[string][]string
	curationData      map[string][]string
	kwargs            map[string][]string
}

// newPackageData returns a pointer to a new packageData object.
func newPackageData() *packageData {
	return &packageData{
		idToNames:         make(map[int64][]string),
		nameToID:          make(map[string]int64),
		alternativeNames:  make(map[string][]string),
		githubData:        make(map[string]float64),
		githubDataChained: make(map[string]float64),
		soData:            make(map[string][]string),
		docData:           make(map[string][]string),
		curationData:      make(map[string][]string),
		kwargs:            make(map[string][]string),
	}
}

// combinePriors combines the given priors with the current prior that packageData
// holds.
func (pd *packageData) combinePriors(chainedLogProb, logProb map[string]float64, weight float64) {
	if len(chainedLogProb) != len(logProb) {
		log.Fatal("len of chained log prob is not equal to len of log prob")
	}
	logWeight := math.Log(weight)
	for m, clp := range chainedLogProb {
		if currentLP, found := pd.githubDataChained[m]; found {
			pd.githubDataChained[m] = logSumExp([]float64{currentLP, clp + logWeight})
			pd.githubData[m] = logSumExp([]float64{pd.githubData[m], logProb[m] + logWeight})
		} else {
			pd.githubDataChained[m] = clp + logWeight
			pd.githubData[m] = logProb[m] + logWeight
		}
	}
}

// namespace returns the identifier names observed for the package in the data.
func (pd *packageData) namespace() []string {
	var names []string
	for name := range pd.alternativeNames {
		names = append(names, name)
	}
	return names
}

// nameByID returns the name represented by the given id.
func (pd *packageData) nameByID(id int64) (string, error) {
	names, found := pd.idToNames[id]
	if found {
		if len(names) == 0 {
			return "", fmt.Errorf("len(names) for id is 0")
		}
		return names[0], nil
	}
	return "", fmt.Errorf("cannot find name for id %d", id)
}

// findID returns the id of a identifier name and fill in entries in data.nameToID
// and data.idToNames.
func (pd *packageData) findID(ident string, graph *pythonimports.Graph) (int64, error) {
	id, found := pd.nameToID[ident]
	if found {
		return id, nil
	}
	node, err := graph.Find(ident)
	if err != nil {
		return -1, err
	}

	id = node.ID
	pd.nameToID[ident] = id
	pd.idToNames[id] = append(pd.idToNames[id], ident)
	canonicalName, err := graph.CanonicalName(ident)
	if err == nil {
		pd.idToNames[id] = append(pd.idToNames[id], canonicalName)
		pd.nameToID[canonicalName] = id
	}
	return id, nil
}

// buildKeywordArgs builds a map from a method name to its keyword args
func (pd *packageData) buildKeywordArgs(graph *pythonimports.Graph, argSpecs *pythonimports.ArgSpecs) {
	for id := range pd.idToNames {
		node, found := graph.FindByID(id)
		if !found {
			log.Fatal("should have found node for id", id)
		}
		name, err := pd.nameByID(id)
		if err != nil {
			log.Fatal(err)
		}
		var args []string
		argSpec := argSpecs.Find(node)
		if argSpec != nil {
			for _, arg := range argSpec.Args {
				if arg.Name != "self" {
					args = append(args, text.TokenizeWithoutCamelPhrases(arg.Name)...)
				}
			}
		}
		pd.kwargs[name] = text.Uniquify(args)
	}
}
