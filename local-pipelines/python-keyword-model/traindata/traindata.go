package main

import (
	"encoding/json"
	"fmt"
	"go/token"
	"hash/fnv"
	"io/ioutil"
	"log"
	"math/rand"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonkeyword"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/local-pipelines/python-keyword-model/traindata/internal/data"
)

const (
	s3Region          = "us-west-1"
	datasetPath       = pythoncode.DedupedCodeDumpPath
	pickCountMaxTries = 1000
)

var (
	// scanOpts and parseOpts should match the options in the driver (or whatever is running inference with the model)
	scanOpts = pythonscanner.Options{
		ScanComments:  true,
		ScanNewLines:  true,
		KeepEOFIndent: true,
	}

	frequenciesMaxFiles int
	collectionMaxFiles  int
	maxSampleCount      uint64
	examplesPerFile     int
	prefixRate          = 0.2
	exampleOutputFile   string

	// keywordTokens has a key for each token that is a keyword
	keywordTokens map[pythonscanner.Token]struct{}
)

var collectCmd = &cobra.Command{
	Use:   "collect OUT_DIRECTORY FREQ_FILE",
	Short: "produce training data for the keyword model",
	Args:  cobra.ExactArgs(2),
	Run:   traindata,
}

var frequencyCmd = &cobra.Command{
	Use:   "freq OUT_FILE",
	Short: "Scan python files to compute keyword frequencies for later subsampling",
	Args:  cobra.ExactArgs(1),
	Run:   computeFrequenciesTable,
}

func maybeQuit(err error) {
	if err != nil {
		panic(err)
	}
}

func init() {
	collectCmd.PersistentFlags().IntVar(&collectionMaxFiles, "n_files", 10000000, "maximum Python source files to attempt reading for computing keyword frequencies")
	frequencyCmd.PersistentFlags().IntVar(&frequenciesMaxFiles, "n_files", 100000, "maximum Python source files to process to generate training dataset")
	collectCmd.PersistentFlags().Uint64Var(&maxSampleCount, "max_samples", 10000, "Maximum number of sample per keyword")
	collectCmd.PersistentFlags().IntVar(&examplesPerFile, "examples_per_file", 5, "number of training samples to generate for each file")
	collectCmd.PersistentFlags().StringVar(&exampleOutputFile, "examples_file", "", "Output file to store example to use with model-test")

	keywordTokens = make(map[pythonscanner.Token]struct{})
	for _, k := range pythonscanner.KeywordTokens {
		keywordTokens[k] = struct{}{}
	}
}

func bucketAndKeys(path string) []string {
	uri, err := awsutil.ValidateURI(path)
	if err != nil {
		log.Fatalln(err)
	}

	bucket := uri.Host
	prefix := uri.Path[1:]

	keys, err := awsutil.S3ListObjects(s3Region, bucket, prefix)
	if err != nil {
		log.Fatalln(err)
	}
	result := make([]string, 0, len(keys))
	for _, s := range keys {
		if !strings.HasSuffix(s, "DONE") {
			result = append(result, fmt.Sprintf("s3://%s/%s", bucket, s))
		}
	}

	return result
}

func getHash(s string) int64 {
	fnvHash := fnv.New32a()
	fnvHash.Write([]byte(s))
	return int64(fnvHash.Sum32())
}

func recordExtractor() func(s pipeline.Sample) []pipeline.Sample {
	return func(s pipeline.Sample) []pipeline.Sample {
		k := s.(pipeline.Keyed)
		hash := getHash(k.Key)
		randomSource := rand.NewSource(hash)
		r := rand.New(randomSource)
		records, err := getRecords(k.Sample.(sample.ByteSlice), r)
		if err != nil {
			return nil
		}
		result := make([]pipeline.Sample, 0, len(records))
		for i, r := range records {
			result = append(result, pipeline.Keyed{Key: fmt.Sprintf("%s%d", k.Key, i), Sample: r})
		}
		return result
	}
}

func countKeywords(ic *data.ItemCounter) func(s pipeline.Sample) pipeline.Sample {
	return func(s pipeline.Sample) pipeline.Sample {
		ic.Mutex.Lock()
		defer ic.Mutex.Unlock()
		record := s.(pipeline.Keyed).Sample.(data.Record)
		if record.IsKeyword {
			ic.Keyword++
			ic.Keywords[record.KeywordCategory]++
		} else {
			ic.Name++
		}
		return s
	}
}

// TODO(Moe): We could skip the frequency table step by using a reservoir sampling method
// See https://en.wikipedia.org/wiki/Reservoir_sampling
// That allows to do a uniform sampling inline without having to store all records in memory

func computeFrequenciesTable(cmd *cobra.Command, args []string) {
	start := time.Now()
	result := data.NewItemCounter(uint64(frequenciesMaxFiles))

	sourceOpts := source.DefaultEMRDatasetOpts
	sourceOpts.MaxRecords = frequenciesMaxFiles
	keys, err := aggregator.ListDir(datasetPath)
	maybeQuit(err)

	emrSource := source.NewEMRDataset("dedupe_files", sourceOpts, keys)
	recordExtract := transform.NewMap("record_extractor", recordExtractor())
	recordCounter := transform.NewOneInOneOut("record_counter", countKeywords(result))

	pm := make(pipeline.ParentMap)

	pm.Chain(emrSource, recordExtract, recordCounter)

	pipe := pipeline.Pipeline{
		Name:    "keyword-model-traindata",
		Parents: pm,
		Sources: []pipeline.Source{emrSource},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: 10,
	})
	maybeQuit(err)
	_, err = engine.Run()
	maybeQuit(err)

	fmt.Printf("Time elapsed for computing frequencies : %v\n", time.Since(start))
	fmt.Printf("Frequencies:\n%v\n", result)
	saveFrequencyTable(result, args[0])
}

func recordSubsampler(sr *data.SamplingRates) func(s pipeline.Sample) bool {
	return func(s pipeline.Sample) bool {
		record := s.(pipeline.Keyed).Sample.(data.Record)
		rSource := rand.NewSource(getHash(s.(pipeline.Keyed).Key))
		r := rand.New(rSource)
		if record.IsKeyword {
			return r.Float64() < sr.Keywords[record.KeywordCategory]
		}
		return r.Float64() < sr.NameKeyword
	}
}

func exampleExtractor(s pipeline.Sample) pipeline.Sample {
	r := s.(pipeline.Keyed).Sample.(data.Record)
	snippet := r.Features.CodeSnippet
	index := strings.Index(snippet, "#$#")
	snippet = snippet[:index]
	return data.KeywordExample{CodeSnippet: snippet, KeywordCategory: r.KeywordCategory}
}

func computeSamplingRates(ic *data.ItemCounter) *data.SamplingRates {
	sr := data.SamplingRates{Keywords: make(map[int]float64)}
	ratio := float64(ic.FilesScanned) / float64(collectionMaxFiles)

	maxCount := uint64(float64(maxSampleCount) * ratio)
	var normalizedKeywords uint64
	for k := range ic.Keywords {
		c := ic.Keywords[k]
		if c < maxCount {
			sr.Keywords[k] = 1
			normalizedKeywords += c
		} else {
			sr.Keywords[k] = float64(maxCount) / float64(c)
			normalizedKeywords += maxCount
		}
	}
	sr.NameKeyword = float64(normalizedKeywords) / float64(ic.Name)
	return &sr
}

func saveFrequencyTable(ic *data.ItemCounter, path string) {
	content, _ := json.MarshalIndent(ic, "", " ")
	_ = ioutil.WriteFile(path, content, 0644)
}

func loadFrequencyTable(path string) *data.ItemCounter {
	content, _ := ioutil.ReadFile(path)
	var result data.ItemCounter
	err := json.Unmarshal(content, &result)
	maybeQuit(err)
	return &result
}

func traindata(cmd *cobra.Command, args []string) {
	start := time.Now()
	outFilename := args[0]

	outDir := args[0]
	log.Printf("will write to %s", outDir)

	keys, err := aggregator.ListDir(datasetPath)
	maybeQuit(err)
	freqTable := loadFrequencyTable(args[1])
	samplingRates := computeSamplingRates(freqTable)
	log.Printf("%d keys found in dataset", len(keys))
	afterSamplingFreq := data.ItemCounter{Keywords: make(map[int]uint64)}
	sourceOpts := source.DefaultEMRDatasetOpts
	sourceOpts.MaxRecords = collectionMaxFiles

	emrSource := source.NewEMRDataset("dedupe_files", sourceOpts, keys)
	recordExtract := transform.NewMap("record_extractor", recordExtractor())
	filterRecords := transform.NewFilter("record_subsampling", recordSubsampler(samplingRates))
	recordCounter := transform.NewOneInOneOut("record_counter", countKeywords(&afterSamplingFreq))
	keyRemover := transform.NewOneInOneOut("key_remover", func(s pipeline.Sample) pipeline.Sample {
		return s.(pipeline.Keyed).Sample
	})
	writer := aggregator.NewJSONWriter(aggregator.WriterOpts{
		FilePrefix: "keyword_traindata",
	}, "json_writer", outDir)

	pm := make(pipeline.ParentMap)

	recordExtraction := pm.Chain(emrSource, recordExtract, filterRecords, recordCounter)

	pm.Chain(recordExtraction, keyRemover, writer)
	if exampleOutputFile != "" {
		exampleWriter := aggregator.NewJSONWriter(aggregator.WriterOpts{
			FilePrefix: "keyword_examples_comparison",
		}, "example_writer", path.Dir(outFilename))
		pm.Chain(recordExtraction, transform.NewOneInOneOut("Example extractor", exampleExtractor), exampleWriter)
	}

	pipe := pipeline.Pipeline{
		Name:    "keyword-model-traindata",
		Parents: pm,
		Sources: []pipeline.Source{emrSource},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers: 10,
	})
	maybeQuit(err)
	_, err = engine.Run()
	maybeQuit(err)
	fmt.Printf("Time elapsed : %v\n", time.Since(start))
	fmt.Printf("Frequencies after subsampling: \n%v\n", afterSamplingFreq.Keywords)
}

func getRecords(src []byte, r *rand.Rand) ([]data.Record, error) {
	words, err := pythonscanner.Lex(src, scanOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to lex file: %v", err)
	}

	chosen, err := chooseWords(words, r, examplesPerFile)
	if err != nil {
		return nil, err
	}

	var records []data.Record

	for _, word := range chosen {
		// Choose some random offset into the current word that the user is in
		literal := word.Literal
		if len(literal) == 0 {
			literal = word.Token.String()
		}
		cursor := int64(word.Begin)
		if r.Float64() < prefixRate {
			// We add a 1 letter prefix in 20% of the cases
			_, inc := utf8.DecodeRune(src[cursor:])
			cursor += int64(inc)
		}

		truncatedSrc := src[:cursor]
		truncatedWords, err := pythonscanner.Lex(truncatedSrc, scanOpts)
		if err != nil {
			return nil, fmt.Errorf("unable to lex truncated file: %v", err)
		}
		cursorPos := token.Pos(cursor)
		parseOpts := pythonparser.Options{
			Approximate: true,
			Cursor:      &cursorPos,
		}
		truncatedAST, err := pythonparser.ParseWords(kitectx.Background(), truncatedSrc, truncatedWords, parseOpts)
		if truncatedAST == nil {
			return nil, fmt.Errorf("unable to parse truncated file: %v", err)
		}

		inputs := modelInputs(truncatedSrc, cursor, truncatedAST, truncatedWords)
		features, err := pythonkeyword.NewFeatures(kitectx.Background(), inputs, pythonkeyword.ModelLookback)
		if err != nil {
			log.Printf("error getting features: %v", err)
			return nil, fmt.Errorf("error getting features: %v", err)
		}

		var keywordCat int
		var isKeyword bool
		if word.Token != pythonscanner.Ident {
			isKeyword = true
			keywordCat = pythonkeyword.KeywordTokenToCat(word.Token)
		}
		if !isKeyword || keywordCat > 0 {
			records = append(records, data.Record{
				Features:        features,
				IsKeyword:       isKeyword,
				KeywordCategory: keywordCat,
				Literal:         word.Literal,
			})
		}
	}

	return records, nil
}

// Randomly choose `count` idents and `count` keyword words from the file for use as training examples. Each ident/keyword
// is as likely to be chosen as another.
func chooseWords(words []pythonscanner.Word, r *rand.Rand, count int) ([]pythonscanner.Word, error) {
	var keywords []pythonscanner.Word
	var idents []pythonscanner.Word

	for _, w := range words {
		if _, ok := keywordTokens[w.Token]; ok {
			keywords = append(keywords, w)
		} else if w.Token == pythonscanner.Ident {
			idents = append(idents, w)
		}
	}

	ni := len(idents)
	nk := len(keywords)

	if ni == 0 {
		return nil, fmt.Errorf("no idents found (keywords = %d)", ni)
	}
	if nk == 0 {
		return nil, fmt.Errorf("no keywords found (idents = %d)", nk)
	}

	if ni < count || nk < count {
		return nil, fmt.Errorf("insufficient words found (%d)", ni+nk)
	}
	res := make([]pythonscanner.Word, 0, count*2)
	res = append(res, pickCount(count, keywords, r)...)
	res = append(res, pickCount(count, idents, r)...)
	return res, nil
}

func pickCount(count int, words []pythonscanner.Word, r *rand.Rand) []pythonscanner.Word {
	res := make([]pythonscanner.Word, 0, count)
	chosen := make(map[pythonscanner.Word]struct{})

	for maxTries := 0; len(chosen) < count && maxTries < pickCountMaxTries; maxTries++ {
		idx := r.Intn(len(words))

		var w pythonscanner.Word
		w = words[idx]

		if _, found := chosen[w]; found {
			continue
		}
		chosen[w] = struct{}{}
	}
	for w := range chosen {
		res = append(res, w)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Begin < res[j].Begin
	})
	return res
}

func modelInputs(src []byte, cursor int64, ast *pythonast.Module, words []pythonscanner.Word) pythonkeyword.ModelInputs {
	nodeCount := pythonast.CountNodes(ast)
	parentMap := pythonast.ConstructParentTable(ast, nodeCount)

	return pythonkeyword.ModelInputs{
		Buffer:    src,
		Cursor:    cursor,
		AST:       ast,
		Words:     words,
		ParentMap: parentMap,
	}
}
