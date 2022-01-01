package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"math"
	"os"
	"path"
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonranker"
	"github.com/kiteco/kiteco/kite-go/ranking"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/hash"
)

const (
	defaultGithubPath = "s3://kite-emr/users/tarak/python-code-examples/2015-05-19_10-28-59-PM/merge_count_incantations/output"
	defaultSoPath     = "s3://kite-emr/datasets/package-prediction/2015-10-28_05-53-21-PM/raw-so-data.json.gz"
	dataRoot          = "s3://kite-emr/datasets/package-prediction/2015-10-28_05-53-21-PM/"
	defaultDocsCorpus = "s3://kite-emr/datasets/documentation/python/2015-07-31_14-22-04-PM/python.json.gz"

	annealingTemp = 0.6

	logPrefix = "[package-predictor] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func main() {
	var (
		in     string
		outdir string
		test   string

		pkgPath    string
		docPath    string
		soPath     string
		selPath    string
		githubPath string
	)
	flag.StringVar(&in, "in", "", "input data")
	flag.StringVar(&test, "test", "", "test data")
	flag.StringVar(&outdir, "outdir", "", "output dir")

	flag.StringVar(&pkgPath, "pkg", fileutil.Join(dataRoot, "packages.txt"), "package list (.txt)")
	flag.StringVar(&docPath, "doc", fileutil.Join(dataRoot, "doc-data.json"), "python doc file (map[string][]string json)")
	flag.StringVar(&selPath, "sel", fileutil.Join(dataRoot, "sel-data.json"), "package to selector file (map[string][]string json)")
	flag.StringVar(&soPath, "so", fileutil.Join(dataRoot, "so-data.json"), "python so file (map[string][]string in json)")
	flag.StringVar(&githubPath, "github", fileutil.Join(dataRoot, "github-data.json"), "path to github data (map[string]float64 in json)")

	flag.Parse()

	// set up logger
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)

	if in == "" || outdir == "" || test == "" {
		log.Fatal("must specify --in, --outdir, --test")
	}

	//
	// load package list
	//
	list := loadPackageList(pkgPath)

	packageSels := make(map[string][]string)
	packageDocs := make(map[string][]string)
	packageSos := make(map[string][]string)
	packagePrior := make(map[string]float64)

	//
	// load doc data
	//
	fdoc, err := fileutil.NewCachedReader(docPath)
	if err != nil {
		// load data locally
		log.Println("loading and converting data from doc corpus...")
		defaultSelectors, defaultDocs := loadDocs(list, defaultDocsCorpus)
		packageSels, packageDocs = loadFromGraph(list, defaultSelectors, defaultDocs)

		err := saveToJSON(docPath, packageDocs)
		if err != nil {
			log.Fatal(err)
		}
		err = saveToJSON(selPath, packageSels)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		decoder := json.NewDecoder(fdoc)
		err := decoder.Decode(&packageDocs)
		if err != nil {
			log.Fatal("error loading doc data:", err)
		}
	}
	defer fdoc.Close()
	log.Println("loaded doc data...")

	//
	// load selector data
	//
	if len(packageSels) == 0 {
		fin, err := fileutil.NewCachedReader(selPath)
		if err != nil {
			// load data locally
			defaultSelectors, defaultDocs := loadDocs(list, pythondocs.DefaultSearchOptions.DocPath)
			packageSels, packageDocs = loadFromGraph(list, defaultSelectors, defaultDocs)
		} else {
			decoder := json.NewDecoder(fin)
			err := decoder.Decode(&packageSels)
			if err != nil {
				log.Fatal("error loading doc data:", err)
			}
		}
		defer fin.Close()
	}
	log.Println("loaded sel data...")

	//
	// load so data
	//
	fso, err := fileutil.NewCachedReader(soPath)
	if err != nil {
		// load data locally
		packageSos = loadSoData(defaultSoPath)
		err := saveToJSON(soPath, packageSos)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		decoder := json.NewDecoder(fso)
		err := decoder.Decode(&packageSos)
		if err != nil {
			log.Fatal("error loading so data:", err)
		}
	}
	defer fso.Close()
	log.Println("loaded so data...")

	//
	// load github data
	//
	fgithub, err := fileutil.NewCachedReader(githubPath)
	if err != nil {
		packagePrior = loadGithub(defaultGithubPath, list, annealingTemp)
		err := saveToJSON(githubPath, packagePrior)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		decoder := json.NewDecoder(fgithub)
		err := decoder.Decode(&packagePrior)
		if err != nil {
			log.Fatal("error loading github data:", err)
		}
	}
	defer fgithub.Close()
	log.Println("loaded github data...")

	// instantiate featurerizer
	featurer, err := pythonranker.NewPackageFeaturer(list, packageSos, packageDocs, packageSels, packagePrior)
	if err != nil {
		log.Fatal(err)
	}

	// load data
	log.Println("loading data")
	queries := loadPackageTestData(in)
	log.Println("loading test data")
	testQueries := loadPackageTestData(test)

	var trainingData []ranking.Entry
	var testData []ranking.Entry

	// convert input data into features
	for q, data := range queries {
		_, isTest := testQueries[q]

		dataPoints := featurer.Features(q)
		for j, dp := range dataPoints {
			var score float64
			var found bool
			if score, found = data[dp.Name]; !found {
				score = 0
			}
			entry := ranking.Entry{
				SnapshotID: int64(j),
				Label:      score,
				QueryHash:  hash.SpookyHash128String([]byte(q)),
				QueryText:  q,
				QueryCode:  dp.Name,
				Features:   dp.Features,
			}
			if isTest {
				testData = append(testData, entry)
			} else {
				trainingData = append(trainingData, entry)
			}
			if math.IsInf(score, 0) {
				log.Fatal("score is inf:", score)
			}
			for _, f := range dp.Features {
				if math.IsInf(f, 0) {
					log.Println(q, dp.Name, dp.Features)
					log.Fatal("feature is inf:", f)
				}
			}
		}
	}

	// write data
	trainPayload := map[string]interface{}{
		"FeatureLabels":   featurer.Labels(),
		"Data":            trainingData,
		"FeaturerOptions": nil,
	}

	testPayload := map[string]interface{}{
		"FeatureLabels":   featurer.Labels(),
		"Data":            testData,
		"FeaturerOptions": nil,
	}

	if err = writeData(trainPayload, testPayload, featurer, outdir); err != nil {
		log.Fatal(err)
	}

}

func writeData(train, test map[string]interface{}, featurer *pythonranker.PackageFeaturer, outdir string) error {
	// check if the directory exists
	if _, err := os.Stat(outdir); os.IsNotExist(err) {
		if err = os.MkdirAll(outdir, os.ModePerm); err != nil {
			return err
		}
	}

	ftrain, err := os.Create(path.Join(outdir, "train.json"))
	if err != nil {
		return err
	}
	defer ftrain.Close()

	encoder := json.NewEncoder(ftrain)
	err = encoder.Encode(train)
	if err != nil {
		return err
	}

	ftest, err := os.Create(path.Join(outdir, "test.json"))
	if err != nil {
		return err
	}
	defer ftest.Close()

	encoder = json.NewEncoder(ftest)
	err = encoder.Encode(test)
	if err != nil {
		return err
	}

	// write featurer
	f, err := os.Create(path.Join(outdir, "featurer.json"))
	if err != nil {
		return err
	}
	defer f.Close()

	encoder = json.NewEncoder(f)
	err = encoder.Encode(featurer)
	if err != nil {
		return err
	}
	return nil
}

type rawPackageData struct {
	Query   string
	Package string
	Value   float64
}

type byValue []*rawPackageData

func (bv byValue) Len() int           { return len(bv) }
func (bv byValue) Swap(i, j int)      { bv[i], bv[j] = bv[j], bv[i] }
func (bv byValue) Less(i, j int) bool { return bv[i].Value < bv[j].Value }

func loadPackageTestData(path string) map[string]map[string]float64 {
	in, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	rawdata := make(map[string][]*rawPackageData)

	decoder := json.NewDecoder(in)
	for {
		var datum rawPackageData
		err := decoder.Decode(&datum)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		rawdata[datum.Query] = append(rawdata[datum.Query], &datum)
	}

	data := make(map[string]map[string]float64)

	for q := range rawdata {
		data[q] = make(map[string]float64)
		sort.Sort(sort.Reverse(byValue(rawdata[q])))
		var scale float64
		for i, d := range rawdata[q] {
			if i == 0 {
				// we scale the score to be capped at 4
				scale = d.Value / 4.0
			}
			if scale != 0 {
				data[q][d.Package] = d.Value / scale
			}
		}
	}
	return data
}
