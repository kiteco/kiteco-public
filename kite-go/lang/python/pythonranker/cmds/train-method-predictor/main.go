package main

import (
	"encoding/gob"
	"flag"
	"io"
	"log"
	"math"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonranker"
	"github.com/kiteco/kiteco/kite-golib/text"
)

const (
	defaultGithubPath = "s3://kite-emr/users/tarak/python-code-examples/2015-05-19_10-28-59-PM/merge_count_incantations/output"
	defaultSoPath     = "s3://kite-emr/datasets/method-prediction/so-data.json.gz"
	defaultTestPath   = "s3://kite-emr/datasets/method-prediction/test-data.json.gz"
	pseudoCount       = 0.001
)

var (
	tokenizer = text.Tokenize
	processor = text.NewProcessor(text.RemoveStopWords, text.Stem)
)

func main() {
	var (
		docsPath   string
		githubPath string
		soPath     string
		testPath   string
		output     string

		annealingTemp       float64
		uniformPriorWeight  float64
		matchingModelWeight float64
		backgroundLMWeight  float64

		crossvalidation bool
	)

	flag.StringVar(&docsPath, "docs", pythondocs.DefaultSearchOptions.DocPath, "path to the doc corpus")
	flag.StringVar(&githubPath, "github", defaultGithubPath, "path to the github stats")
	flag.StringVar(&soPath, "so", defaultSoPath, "path to the stack overflow data")
	flag.StringVar(&testPath, "test", defaultTestPath, "path to the file that contains test data")
	flag.StringVar(&output, "out", "", "output file")
	flag.Float64Var(&annealingTemp, "annealing", 0.6, "annealing temp on the method dist gathered from github")
	flag.Float64Var(&uniformPriorWeight, "uniform_prior", 0.1, "weight on uniform prior distribution")
	flag.Float64Var(&matchingModelWeight, "matching_weight", 0.4, "weight for matching model")
	flag.Float64Var(&backgroundLMWeight, "background_weight", 0.1, "weights on the background language model")
	flag.BoolVar(&crossvalidation, "cv", false, "to do cross validation")
	flag.Parse()

	if soPath == "" {
		log.Fatal("must specify path to the so data --so")
	}
	if uniformPriorWeight < 0 || uniformPriorWeight > 1 {
		log.Fatal("weight on uniform prior must be within the range of [0, 1].")
	}
	if matchingModelWeight < 0 || matchingModelWeight > 1 {
		log.Fatal("weight on the matching model must be within the range of [0, 1].")
	}
	if backgroundLMWeight < 0 || backgroundLMWeight > 1 {
		log.Fatal("weight on the background language model must be within the range of [0, 1].")
	}

	if output == "" {
		log.Fatal("must specify path to the output file by --out")
	}

	// parse test data
	var testData []testDatum
	if testPath != "" {
		testData = loadTestData(testPath)
	}
	log.Println("loaded test data")

	// parse the modules and construct structured module representation.
	// we use this to estimate distribution on class methods.
	parsedModules, matchingCandidates := parseDocStruct(docsPath)

	// load training data from doc corpus and stack overflow corpus.
	data := loadDocs(docsPath)
	loadSOData(soPath, data)

	// we construct the map that stores the prior information that we'll
	// get from github by using githubStats because we only care
	// about functions/class methods that we have docs for.
	var counter int
	githubPrior := make(map[string]map[string]float64)
	for p, pdata := range data {
		githubPrior[p] = make(map[string]float64)
		for m := range pdata {
			githubPrior[p][m] = 0.0
		}
		counter += len(pdata)
	}
	// load prior on functions / class methods from github stats
	githubStats(githubPath, githubPrior, parsedModules)

	// smooth the prior with uniform distribution
	prior := smoothPrior(githubPrior, annealingTemp, uniformPriorWeight)

	// instantiate method predictors
	predictors := make(map[string]*pythonranker.MethodPredictor)
	lmPredictors := make(map[string]*pythonranker.LmMethodPredictor)
	matcherPredictors := make(map[string]*pythonranker.MatchPredictor)

	weights := []float64{matchingModelWeight, 1 - matchingModelWeight}
	for p, pdata := range data {
		for m, mdata := range pdata {
			mdata.LogPrior = prior[p][m]
		}
		matcher := pythonranker.NewMatchPredictor(matchingCandidates[p])
		lm := pythonranker.NewLmMethodPredictor(backgroundLMWeight, pdata)
		predictors[p] = pythonranker.NewMethodPredictor(matcher, lm, weights)
		lmPredictors[p] = lm
		matcherPredictors[p] = matcher
	}

	if crossvalidation {
		var best struct {
			annealingTemp       float64
			uniformPriorWeight  float64
			matchingModelWeight float64
			backgroundLMWeight  float64
			accuracy            float64
		}
		for annealingTemp = 0; annealingTemp <= 1.0; annealingTemp += 0.1 {
			for uniformPriorWeight = 0; uniformPriorWeight <= 1; uniformPriorWeight += 0.1 {
				prior := smoothPrior(githubPrior, annealingTemp, uniformPriorWeight)
				for matchingModelWeight = 0; matchingModelWeight <= 1; matchingModelWeight += 0.1 {
					weights := []float64{matchingModelWeight, 1 - matchingModelWeight}
					for backgroundLMWeight = 0; backgroundLMWeight <= 1; backgroundLMWeight += 0.1 {
						for p := range data {
							lmPredictors[p].SetBackgroundWeight(backgroundLMWeight)
							err := lmPredictors[p].SetLogPrior(prior[p])
							if err != nil {
								log.Fatal(err)
							}
							predictors[p] = pythonranker.NewMethodPredictor(matcherPredictors[p], lmPredictors[p], weights)
						}
						acc := evaluate(testData, predictors, prior)
						log.Println("Experiment results....")
						log.Println("annealing temp:", annealingTemp)
						log.Println("uniform prior weight:", uniformPriorWeight)
						log.Println("matching model weight:", matchingModelWeight)
						log.Println("background lm weight:", backgroundLMWeight)
						log.Println("accuracy:", acc)
						log.Println("===========================")
						if acc >= best.accuracy {
							best.annealingTemp = annealingTemp
							best.uniformPriorWeight = uniformPriorWeight
							best.matchingModelWeight = matchingModelWeight
							best.backgroundLMWeight = backgroundLMWeight
							best.accuracy = acc
						}
					}
				}
			}
		}
		log.Println("Training with the following parameters...")
		log.Println("annealing temp:", best.annealingTemp)
		log.Println("uniform prior weight:", best.uniformPriorWeight)
		log.Println("matching model weight:", best.matchingModelWeight)
		log.Println("background lm weight:", best.backgroundLMWeight)
		prior := smoothPrior(githubPrior, best.annealingTemp, best.uniformPriorWeight)
		weights := []float64{best.matchingModelWeight, 1 - best.matchingModelWeight}

		for p := range data {
			lmPredictors[p].SetBackgroundWeight(best.backgroundLMWeight)
			err := lmPredictors[p].SetLogPrior(prior[p])
			if err != nil {
				log.Fatal(err)
			}
			predictors[p] = pythonranker.NewMethodPredictor(matcherPredictors[p], lmPredictors[p], weights)
		}
	}

	acc := evaluate(testData, predictors, prior)
	log.Println("accuracy:", acc)
	out, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	err = dumpModel(out, predictors)
	if err != nil {
		log.Fatal(err)
	}
}

// evaluate returns the accuracy of the method prediction model.
func evaluate(testData []testDatum, predictors map[string]*pythonranker.MethodPredictor, prior map[string]map[string]float64) float64 {
	var total float64
	var success float64
	for _, data := range testData {
		queryTokens := processor.Apply(tokenizer(data.query))
		for _, method := range data.methods {
			tokens := strings.Split(method, ".")
			if len(tokens) <= 1 {
				continue
			}
			pkg := tokens[0]
			var predictor *pythonranker.MethodPredictor
			var exists bool
			if predictor, exists = predictors[pkg]; !exists {
				continue
			}
			mtd := tokens[len(tokens)-1]
			if _, exists = prior[pkg][mtd]; !exists {
				continue
			}
			total++
			predictedMethods := predictor.PredictTopNSels(queryTokens, 3)
			if match(mtd, predictedMethods) {
				success++
			}
		}
	}
	return success / total
}

func dumpModel(writer io.Writer, predictors map[string]*pythonranker.MethodPredictor) error {
	encoder := gob.NewEncoder(writer)
	err := encoder.Encode(predictors)
	if err != nil {
		return err
	}
	return nil
}

// match returns true if the target string if sound in the list of candidate strings.
func match(target string, candidates []string) bool {
	for _, c := range candidates {
		if target == c {
			return true
		}
	}
	return false
}

// smoothPrior smooths the prior distribution we get from github stats
// by annealing the github prior and mixing it with a uniform distribution.
func smoothPrior(prior map[string]map[string]float64, annealingTemp, uniformPriorWeight float64) map[string]map[string]float64 {
	smoothedPrior := make(map[string]map[string]float64)
	for p := range prior {
		smoothedPrior[p] = make(map[string]float64)
		var total float64
		for m, count := range prior[p] {
			smoothedPrior[p][m] = math.Pow(count, annealingTemp) + pseudoCount
			total += smoothedPrior[p][m]
		}
		methodCount := float64(len(smoothedPrior[p]))

		for m, count := range smoothedPrior[p] {
			if methodCount != 0 {
				smoothedPrior[p][m] = math.Log((1-uniformPriorWeight)*(count/total) + uniformPriorWeight*(1/methodCount))
			} else {
				smoothedPrior[p][m] = math.Log(count / total)
			}
		}
	}
	return smoothedPrior
}
