package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonkeyword"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncompletions"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmixing"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

const (
	s3Region    = "us-west-1"
	datasetPath = "s3://kite-emr/users/juan/python-dedupe-code/2018-07-26_13-53-43-PM/dedupe/output/"
)

var (
	// scanOpts and parseOpts should match the options in the driver (or whatever is running inference with the model)
	scanOpts = pythonscanner.Options{
		ScanComments: true,
		ScanNewLines: true,
	}

	parseOpts = pythonparser.Options{
		ScanOptions: pythonscanner.Options{
			ScanComments: false,
			ScanNewLines: false,
		},
		ErrorMode: pythonparser.Recover,
	}
)

// need this to communicate with S3. This is to get keys associates with data partitions.
func bucketAndKeys(path string) (string, []string) {
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

	return bucket, keys
}

func findRandomNode(ast *pythonast.Module, r *rand.Rand) *pythonast.AttributeExpr {
	var nodeList []*pythonast.AttributeExpr
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}
		switch n := n.(type) {
		case *pythonast.AttributeExpr:
			nodeList = append(nodeList, n)
		}

		return true
	})

	if len(nodeList) == 0 {
		return nil
	}

	return nodeList[r.Intn(len(nodeList))]
}

func getState(src []byte, rm pythonresource.Manager, r *rand.Rand) (State, error) {

	words, err := pythonscanner.Lex(src, scanOpts)
	if err != nil {
		return State{}, fmt.Errorf("unable to lex file: %v", err)
	}

	ast, err := pythonparser.ParseWords(kitectx.Background(), src, words, parseOpts)
	if err != nil {
		return State{}, fmt.Errorf("unable to parse file: %v", err)
	}

	// select a random node
	expr := findRandomNode(ast, r)
	if expr == nil {
		return State{}, errors.New("no attribute node")
	}

	userTyped := src[expr.Dot.End:]

	cursor := expr.Dot.End

	// truncate the source.
	src = src[0:cursor]

	// second pass for the source

	words2, err := pythonscanner.Lex(src, scanOpts)
	if err != nil {
		return State{}, fmt.Errorf("unable to lex file: %v", err)
	}

	withCursor := parseOpts
	withCursor.Cursor = &cursor
	ast2, err := pythonparser.ParseWords(kitectx.Background(), src, words2, withCursor)
	if ast2 == nil {
		return State{}, fmt.Errorf("unable to parse file: %v", err)
	}

	var expr2 *pythonast.AttributeExpr
	pythonast.Inspect(ast2, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}
		switch n := n.(type) {
		case *pythonast.AttributeExpr:
			if n.Dot.End == expr.Dot.End {
				expr2 = n
				return true
			}
		}

		return true
	})

	if pythonast.IsNil(expr2) {
		return State{}, errors.New("can't find the same attribute node again")
	}

	var rast *pythonanalyzer.ResolvedAST
	err = kitectx.Background().WithTimeout(10*time.Second, func(ctx kitectx.Context) error {
		var err error
		rast, err = pythonanalyzer.NewResolver(rm, pythonanalyzer.Options{Path: "/src.py"}).ResolveContext(ctx, ast2, false)
		return err
	})
	if err != nil {
		return State{}, fmt.Errorf("resolve error: %v", err)
	}

	state := State{
		Buffer:        src,
		Words:         words2,
		RAST:          rast,
		AttributeExpr: expr2,
		UserTyped:     userTyped,
		Cursor:        int64(cursor),
	}
	return state, nil
}

func getSample(s State, hash string, cr compResources) (pythonmixing.TrainSample, error) {
	// get mixed input
	mips, err := getMixInputs(s, cr)
	if err != nil {
		return pythonmixing.TrainSample{}, fmt.Errorf("unable to get completions %v", err)
	}

	// get list of completion features
	compFeatures := getCompFeatures(s, cr, mips)

	// check to make sure that there is more than one completion type
	compTypeMap := make(map[pythonmixing.CompType]int)
	for _, cf := range compFeatures {
		compTypeMap[cf.Model]++
	}

	if len(compTypeMap) <= 1 {
		return pythonmixing.TrainSample{}, errors.New("bias error: only one completion type")
	}

	// get label
	label, err := getLabel(s.UserTyped, mips)
	if err != nil {
		return pythonmixing.TrainSample{}, fmt.Errorf("unable to fetch label %v", err)
	}

	// get contextual features
	n := mips[0].Completion().MixData.NumVarsInScope
	parentNode := s.RAST.Parent[s.AttributeExpr]
	parentType := pythonkeyword.NodeToCat(parentNode)
	contextFeature := pythonmixing.ContextualFeatures{NumVars: n, ParentType: parentType}
	compIds := make([]string, 0, len(mips))
	for _, mip := range mips {
		compIds = append(compIds, mip.Completion().Identifier)
	}

	return pythonmixing.TrainSample{
		Contextual:   contextFeature,
		Label:        label,
		CompFeatures: compFeatures,
		Meta: pythonmixing.TrainSampleMeta{
			Hash:            hash,
			Cursor:          s.Cursor,
			CompIdentifiers: compIds,
		},
	}, nil
}

func getLabel(ut []byte, mips []pythoncompletions.MixInput) (int, error) {
	scannedWords, err := pythonscanner.Scan(ut)
	if err != nil {
		return 0, err
	}

	var label int
	var maxLength int

	for i, mip := range mips {
		comp := mip.Completion()

		scannedComp, err := pythonscanner.Scan([]byte(comp.Identifier))
		if err != nil {
			return 0, err
		}

		newMax := isPrefix(scannedWords, scannedComp)
		if newMax > maxLength {
			label = i
			maxLength = newMax
		}
	}

	if maxLength > 0 {
		return label, nil
	}

	return 0, errors.New("no match completions")
}

// isPrefix checks if prefix is a prefix of tokens, if yes then it returns the length of the prefix otherwise returns 0.
func isPrefix(tokens []pythonscanner.Word, prefix []pythonscanner.Word) int {
	min := len(prefix)

	var length int
	for i := 0; i < min; i++ {
		t := tokens[i]
		p := prefix[i]
		if p.Token == pythonscanner.EOF {
			break
		}
		if t.Literal == p.Literal && t.Token == p.Token {
			length++
		} else {
			return 0
		}
	}
	return length
}

func getCompFeatures(in State, cr compResources, mixIP []pythoncompletions.MixInput) []pythonmixing.CompFeatures {
	var compF []pythonmixing.CompFeatures
	for _, mip := range mixIP {
		comp := mip.Completion()

		length := len(comp.Identifier)

		var model pythonmixing.CompType
		var numArgs int
		var popularityScore float64
		var softmaxScore float64
		var attrScore float64
		switch mip.(type) {
		case pythoncompletions.PopularitySingleAttribute:
			model = pythonmixing.SinglePopularityComp
			numArgs = 0
			popularityScore = comp.Score / 1000
			softmaxScore = 0.0
			attrScore = 0.0
		case pythoncompletions.MultiTokenCall:
			model = pythonmixing.CallGGNNComp
			numArgs = strings.Count(comp.Identifier, ",") + 1
			popularityScore = 0.0
			softmaxScore = comp.Score
			attrScore = 0.0
			inputs := python.CompletionsInputs{
				Buffer:   in.Buffer,
				Resolved: in.RAST,
				Importer: pythonstatic.Importer{Global: cr.rm},
				Models:   cr.models,
			}

			cb := python.NewCompletionsEngine(inputs).Callbacks
			score, _ := cb.CallProbs(kitectx.Background(), []pythoncompletions.Completion{comp})
			if len(score) == 0 {
				softmaxScore = comp.Score
			} else {
				softmaxScore = float64(score[0])
			}
		case pythoncompletions.GGNNAttribute:
			model = pythonmixing.SingleGGNNAttrComp
			numArgs = 0
			popularityScore = 0.0
			softmaxScore = 0.0
			attrScore = comp.Score
		}
		compF = append(compF, pythonmixing.CompFeatures{
			PopularityScore:  popularityScore,
			SoftmaxScore:     softmaxScore,
			CompletionLength: length,
			Model:            model,
			NumArgs:          numArgs,
			AttrScore:        attrScore,
		})
	}
	if len(compF) == 0 {
		return nil
	}
	return compF
}

func main() {
	args := struct {
		Out      string
		MaxFiles int
	}{
		Out:      "./train_data.json",
		MaxFiles: 10000,
	}

	arg.MustParse(&args)
	outFile := args.Out
	maxFiles := args.MaxFiles

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		log.Fatal(err)
	}
	serviceOpts := python.DefaultServiceOptions

	models, err := pythonmodels.New(serviceOpts.ModelOptions)
	if err != nil {
		log.Fatalln(err)
	}
	cr := compResources{rm: rm, models: models}

	start := time.Now()
	bucket, keys := bucketAndKeys(datasetPath)

	var skippedErr int
	var successful int
	var total int
	var done bool

	randSource := rand.NewSource(1)
	random := rand.New(randSource)

	outf, err := os.Create(outFile)
	if err != nil {
		log.Fatal(err)
	}
	defer outf.Close()

	encoder := json.NewEncoder(outf)
	for _, key := range keys {
		uri := fmt.Sprintf("s3://%s/%s", bucket, key)

		log.Printf("reading %s", uri)

		r, err := awsutil.NewCachedS3Reader(uri)
		if err != nil {
			log.Fatalln(err)
		}

		in := awsutil.NewEMRIterator(r)

		for in.Next() {
			total++
			if total > maxFiles {
				done = true
				break
			}

			state, err := getState(in.Value(), rm, random)
			if err != nil {
				skippedErr++
				log.Printf("Error getting state: %v", err)
				continue
			}

			sample, err := getSample(state, in.Key(), cr)
			if err != nil {
				skippedErr++
				log.Printf("Error getting sample: %v", err)
				continue
			}

			if err := encoder.Encode(sample); err != nil {
				log.Fatal(err)
			}
			successful++

		}

		if done {
			break
		}

	}

	log.Printf("Done! took %v, successful: %d, skipped (err): %d\n",
		time.Since(start), successful, skippedErr)

}
