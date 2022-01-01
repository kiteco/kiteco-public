package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"

	arg "github.com/alexflint/go-arg"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels/callprob"

	"github.com/kiteco/kiteco/kite-go/lang/python"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/local-pipelines/python-call-filtering/internal/utils"
)

const (
	maxLines = 250
)

func inSlice(s []int, i int) bool {
	for _, ii := range s {
		if i == ii {
			return true
		}
	}
	return false
}

func auxComputeTopNRecall(n int, res utils.Resources, mips []metricInputs, partialCall bool) float32 {
	var countTopNMatches int
	for _, mip := range mips {
		var callprobPred []float32
		var err error
		if partialCall {
			callprobPred, err = res.Models.PartialCallProb.Infer(kitectx.Background(), mip.TruncatedInput)
		} else {
			callprobPred, err = res.Models.FullCallProb.Infer(kitectx.Background(), mip.FullInput)
		}
		if err != nil {
			log.Printf("callProb cannot do inference: %v", err)
			continue
		}
		idSlice := make([]int, len(callprobPred))
		for i := 0; i < len(callprobPred); i++ {
			idSlice[i] = i
		}
		slice := revFloat32Slice{
			Floats: callprobPred,
			Idxs:   idSlice,
		}
		sort.Sort(slice)
		for i, val := range slice.Idxs {
			matches := mip.matchesFull
			if partialCall {
				matches = mip.matchesTruncated
			}
			if inSlice(matches, val) {
				countTopNMatches++
			}
			if i >= n {
				break
			}
		}
	}
	return float32(countTopNMatches) / float32(len(mips))
}

// compute top n recall for each segment of the data
func computeTopNRecall(n int, res utils.Resources, mips *segmentedData) {
	for i := 0; i < utils.ArgCategories; i++ {
		recallFull := auxComputeTopNRecall(n, res, mips.inputs[i], false)
		recallTruncated := auxComputeTopNRecall(n, res, mips.inputs[i], true)
		mips.recalls[i][n] = recallInfo{
			fullCalls:      recallFull,
			truncatedCalls: recallTruncated,
		}
	}
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

type metricInputs struct {
	FullInput        callprob.Inputs
	TruncatedInput   callprob.Inputs
	matchesFull      []int
	matchesTruncated []int
}

type recallInfo struct {
	fullCalls, truncatedCalls float32
}

type segmentedData struct {
	inputs  [utils.ArgCategories][]metricInputs
	recalls [utils.ArgCategories]map[int]recallInfo
}

func (sd segmentedData) String() string {
	var result []string
	for i := range sd.recalls {
		result = append(result, fmt.Sprintf("Recalls for calls with %d arguments: ", i))
		for j, r := range sd.recalls[i] {
			result = append(result, fmt.Sprintf("Top%d recall : %v / %v (truncated calls/full calls)", j, r.truncatedCalls, r.fullCalls))
		}
	}
	return strings.Join(result, "\n")
}

func newSegmentedData() *segmentedData {
	result := segmentedData{
		inputs:  [utils.ArgCategories][]metricInputs{},
		recalls: [utils.ArgCategories]map[int]recallInfo{},
	}
	for i := range result.recalls {
		result.recalls[i] = make(map[int]recallInfo)
	}
	return &result
}

func main() {
	datadeps.Enable()
	args := struct {
		Input string
	}{
		Input: "../data/gt_data.json",
	}

	arg.MustParse(&args)

	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		log.Fatal(err)
	}
	serviceOpts := python.DefaultServiceOptions

	models, err := pythonmodels.New(serviceOpts.ModelOptions)
	if err != nil {
		log.Fatalln(err)
	}

	res := utils.Resources{RM: rm, Models: models}

	dataFile, err := os.Open(args.Input)
	if err != nil {
		log.Fatalf("cannot access data file: %v", err)
	}
	defer dataFile.Close()
	fmt.Println("Succesfully opened gt_data.json")

	dec := json.NewDecoder(dataFile)
	var total, lines, totalCalls int
	sd := newSegmentedData()

	for {
		lines++
		var s utils.Sample
		err := dec.Decode(&s)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("cannot decode json: %v", err)
		}
		// pass in and first traverse to get the corresponding call expressions.
		calls, err := utils.FindCalls(s)
		if err != nil {
			log.Printf("error finding corresponding call expressions: %v", err)
			continue
		}

		//for each call evaluate the expr model to get results and then look if there is a match
		for _, call := range calls {
			totalCalls++
			// do all the munging then return expr input and pass to expr model to get the prediction
			inp, err := utils.TryCall(call, s.Source, res)
			if err != nil {
				fmt.Printf("problem getting input: %v", err)
				continue
			}

			preds, sym, scopeSize, err := res.PredictCall(inp.Src, inp.Words, inp.RAST, inp.Call)
			if err != nil {
				log.Printf("expr model can't predict this: %v", err)
				continue
			}
			argLength := len(call.Args)
			truncated := utils.TruncateCalls(preds)
			fullCalls := utils.KeepOnlyCompleteCalls(preds)

			matchedFull, err := utils.IsPredicted(fullCalls, call, false)
			if err != nil {
				continue
			}
			matchedTruncated, err := utils.IsPredicted(truncated, call, true)
			if err != nil {
				continue
			}
			// tally up the calls that was able to be predicted and there is a match
			total++

			cpiTruncated := callprob.Inputs{
				RM:        res.RM,
				RAST:      inp.RAST,
				CallComps: truncated,
				Sym:       sym,
				Cursor:    inp.Cursor,
				ScopeSize: scopeSize,
			}
			cpiFull := callprob.Inputs{
				RM:        res.RM,
				RAST:      inp.RAST,
				CallComps: fullCalls,
				Sym:       sym,
				Cursor:    inp.Cursor,
				ScopeSize: scopeSize,
			}

			// Add segmentation to data

			if argLength < utils.ArgCategories {
				sd.inputs[argLength] = append(sd.inputs[argLength], metricInputs{
					FullInput:        cpiFull,
					TruncatedInput:   cpiTruncated,
					matchesFull:      matchedFull,
					matchesTruncated: matchedTruncated,
				})
			}
		}
		if lines > maxLines {
			break
		}
	}
	computeTopNRecall(1, res, sd)
	computeTopNRecall(5, res, sd)

	fmt.Println(sd.String())
	fmt.Printf("total data is %v (for %d lines in the file, and %d calls) \n", total, lines, totalCalls)
}
