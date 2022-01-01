package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	arg "github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonexpr"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/mtacconf"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/threshold"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/local-pipelines/python-mtac-filtering/internal/utils"
)

var (
	// parseOpts should match the options in the driver (or whatever is running inference with the model)
	parseOpts = pythonparser.Options{
		ScanOptions: pythonscanner.Options{
			ScanComments: false,
			ScanNewLines: false,
		},
		ErrorMode: pythonparser.Recover,
	}
)

type sample struct {
	Source  []byte
	PosList []int64
}

// get sorted score and original ID slice for each MTAC.
func getSortedScoreAndID(res utils.Resources, mip metricInputs) (revFloat32Slice, []threshold.MTACScenario, error) {
	mtacconfPred, err := res.Models.MTACConf.Infer(kitectx.Background(), mip.Input)
	if err != nil {
		return revFloat32Slice{}, nil, fmt.Errorf("MTACConf cannot do inference: %v", err)
	}

	scenarios := make([]threshold.MTACScenario, len(mtacconfPred))
	for i := 0; i < len(mtacconfPred); i++ {
		scenarios[i] = mip.Input.Comps[i].MixData.Scenario
	}
	idSlice := make([]int, len(mtacconfPred))
	for i := 0; i < len(mtacconfPred); i++ {
		idSlice[i] = i
	}
	slice := revFloat32Slice{
		Floats: mtacconfPred,
		Idxs:   idSlice,
	}
	sort.Sort(slice)

	return slice, scenarios, nil
}

// revFloat32Slice wraps sort to get it to work with float32 and in reverse order
type revFloat32Slice struct {
	Idxs   []int
	Floats []float32
}

func (s revFloat32Slice) Swap(i, j int) {
	s.Floats[i], s.Floats[j] = s.Floats[j], s.Floats[i]
	s.Idxs[i], s.Idxs[j] = s.Idxs[j], s.Idxs[i]
}

func (s revFloat32Slice) Less(i, j int) bool {
	return s.Floats[j] < s.Floats[i]
}

func (s revFloat32Slice) Len() int {
	return len(s.Floats)
}

func findNames(s sample, res utils.Resources) ([]*pythonast.NameExpr, *pythonanalyzer.ResolvedAST, error) {
	ast, err := pythonparser.Parse(kitectx.Background(), s.Source, parseOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse file: %v", err)
	}

	var names []*pythonast.NameExpr
	pythonast.Inspect(ast, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) {
			return false
		}

		if n, ok := n.(*pythonast.NameExpr); ok {
			for _, p := range s.PosList {
				if int64(n.Begin()) == p {
					names = append(names, n)
					break
				}
			}
		}
		return true
	})

	if len(names) == 0 {
		return nil, nil, fmt.Errorf("no name expressions found")
	}
	rast, err := utils.Resolve(ast, res.RM)
	if err != nil {
		return nil, nil, fmt.Errorf("can't resolve")
	}

	return names, rast, nil
}

type metricInputs struct {
	Input   mtacconf.Inputs
	MatchID int
}

func main() {
	datadeps.Enable()
	args := struct {
		Input     string
		OutParams string
		ModelPath string
	}{
		Input: "../data/gt_data.json",
	}

	arg.MustParse(&args)

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		log.Fatal(err)
	}
	serviceOpts := python.DefaultServiceOptions

	mtacconfModel, err := mtacconf.NewBaseModel(args.ModelPath)
	if err != nil {
		log.Fatalln(err)
	}

	exprModel, err := pythonexpr.NewModel(serviceOpts.ModelOptions.ExprModelShards[0].ModelPath, pythonexpr.DefaultOptions)
	if err != nil {
		log.Fatalln(err)
	}

	res := utils.Resources{RM: rm, Models: &pythonmodels.Models{MTACConf: mtacconfModel, Expr: exprModel}}

	dataFile, err := fileutil.NewCachedReader(args.Input)
	if err != nil {
		log.Fatalf("cannot access data file: %v", err)
	}
	defer dataFile.Close()
	fmt.Println("Succesfully opened gt_data.json")

	scanner := bufio.NewScanner(dataFile)

	var total int
	nameCompsByScenario := make(map[threshold.MTACScenario][]threshold.Comp)
	for scanner.Scan() {
		var s sample
		if err := json.Unmarshal(scanner.Bytes(), &s); err != nil {
			log.Fatalf("cannot decode json: %v", err)
		}
		// pass in and first traverse to get the corresponding call expressions.
		names, rast, err := findNames(s, res)
		if err != nil {
			log.Printf("error finding corresponding call expressions: %v", err)
			continue
		}

		//for each call evaluate the expr model to get results and then look if there is a match
		for _, name := range names {
			// do all the munging then return expr input and pass to expr model to get the prediction
			global := pythonproviders.Global{
				ResourceManager: res.RM,
				Models:          res.Models,
				Product:         licensing.Pro,
			}
			inputs, err := utils.TryName(s.Source, name, rast, global)
			if err != nil {
				fmt.Printf("problem getting input: %v", err)
				continue
			}

			exprInput := pythonexpr.Input{
				RM:          global.ResourceManager,
				RAST:        inputs.ResolvedAST(),
				Words:       inputs.Words(),
				Expr:        inputs.Name,
				MaxPatterns: 10,
			}
			exprPred, err := res.Models.Expr.Predict(kitectx.Background(), exprInput)
			if err != nil {
				log.Printf("expr model can't predict this: %v", err)
				continue
			}

			var completions []mtacconf.Completion
			var idents []string
			probs, prefixes := []float64{1.}, []string{""}
			pythongraph.Inspect(exprPred.OldPredictorResult, func(n *pythongraph.PredictionTreeNode) bool {
				lastProb := len(probs) - 1
				lastPrefix := len(prefixes) - 1
				if n == nil {
					probs = probs[:lastProb]
					prefixes = prefixes[:lastPrefix]
					return false
				}

				prob := probs[lastProb] * float64(n.Prob)
				probs = append(probs, prob)

				parentPrefix := prefixes[lastPrefix]

				switch {
				case n.AttrBase != "":
					// NOTE: no need to use existing prefix since this is always the root if it exists
					prefixes = append(prefixes, n.AttrBase)
					var c mtacconf.Completion
					if val, err := inputs.TypeValueForName(kitectx.Background(), global.ResourceManager, n.AttrBase); err == nil {
						c.Referent = val
					}
					c.MixData = mtacconf.GetMixData(kitectx.Background(), global.ResourceManager, inputs.Selection, inputs.Words(), inputs.ResolvedAST(), inputs.Name)
					c.Score = prob
					c.Source = response.ExprModelCompletionsSource
					completions = append(completions, c)
					idents = append(idents, n.AttrBase)
				case !n.Attr.Nil():
					prefix := fmt.Sprintf("%s.%s", parentPrefix, n.Attr.Path().Last())
					prefixes = append(prefixes, prefix)

					var c mtacconf.Completion
					c.Referent = pythontype.NewExternal(n.Attr, global.ResourceManager)
					c.MixData = mtacconf.GetMixData(kitectx.Background(), global.ResourceManager, inputs.Selection, inputs.Words(), inputs.ResolvedAST(), inputs.Name)
					c.Score = prob
					c.Source = response.AttributeModelCompletionSource // TODO: add separate sources for expr generated attributes?
					completions = append(completions, c)
					idents = append(idents, prefix)
				default:
					prefixes = append(prefixes, "")
				}
				return true
			})

			matched, err := utils.GetLabel(inputs.UserTyped, idents)
			if err != nil {
				log.Printf("error making a prediction: %v", err)
				continue
			}
			// tally up the calls that was able to be predicted and there is a match
			total++
			mtacconfInput := mtacconf.Inputs{
				RM:    res.RM,
				Words: inputs.Words(),
				RAST:  inputs.ResolvedAST(),
				Comps: completions,
			}

			// Add data to a map for different scenarios of MTAC.
			mip := metricInputs{Input: mtacconfInput, MatchID: matched}
			ssID, scenarios, err := getSortedScoreAndID(res, mip)
			if err != nil {
				log.Println(err)
				continue
			}
			for i, comID := range ssID.Idxs {
				matching := comID == mip.MatchID
				nameCompsByScenario[scenarios[i]] = append(nameCompsByScenario[scenarios[i]], threshold.Comp{IsMatched: matching, Score: ssID.Floats[i]})
			}
		}
	}

	mtacThreshold := make(threshold.MTACThreshold)
	for i, nameComps := range nameCompsByScenario {
		mtacThreshold[i] = threshold.GetOptimalThreshold(nameComps)
	}

	// writing threshold to a file
	outf2, err := os.Create(args.OutParams)
	if err != nil {
		log.Fatal(err)
	}
	defer outf2.Close()

	encoder2 := json.NewEncoder(outf2)
	if err := encoder2.Encode(mtacThreshold); err != nil {
		log.Fatal(err)
	}
	for i, t := range mtacThreshold {
		fmt.Printf("threshold for scenario %v is %v", i, t)
	}
	fmt.Printf("total data is %v \n", total)
}
