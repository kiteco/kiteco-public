package recalltest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonpipeline"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/tensorflow"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

type record struct {
	Source []byte
}

// RecallSet contains recall results for one recall definition (exact match or placeholder match)
// Thresholds fields are used for baseline definition when comparing new recall to expected ones
type RecallSet struct {
	Recalls           []float32 `json:"recalls"`
	AbsoluteThreshold float32   `json:"absolute_threshold"`
	RelativeThreshold float32   `json:"relative_threshold"`
}

// RecallsInfo groups all results of recall computation (and is also used to store baseline results)
type RecallsInfo struct {
	ExactMatch       RecallSet     `json:"exact_match"`
	PlaceholderMatch RecallSet     `json:"placeholder_match"`
	Timestamp        time.Time     `json:"timestamp"`
	Duration         time.Duration `json:"duration"`
}

func (info RecallsInfo) String() string {
	var result string
	result += "Exact Match recall:\n"
	for i, r := range info.ExactMatch.Recalls {
		result += fmt.Sprintf("- Recall %d: %.2f\n", i, r)
	}
	result += "Placeholder Match recall:\n"
	for i, r := range info.PlaceholderMatch.Recalls {
		result += fmt.Sprintf("- Recall %d: %.2f\n", i, r)
	}
	result += fmt.Sprintf("Duration : %v\n", info.Duration)
	return result
}

type matchingCompletion struct {
	Rank                     int    `json:"rank"`                       // Rank of this completion in the result returned by the API
	ConcretePlaceholderMatch int    `json:"concrete_placeholder_match"` // Number of match against a concrete placeholder (wrong name, but accepted for signature match)
	Completion               string `json:"completion"`                 // Text of the completion
	ExactMatch               bool   `json:"exact_match"`                // Exact match means exact same args and name
	PlaceholderCount         int    `json:"placeholder_count"`          // Number of placeholder in the completion
}

// CompletionResult contains all information from the completion matching for one buffer (initial buffer, completions generated and list of matches).
type CompletionResult struct {
	Buffer              data.SelectedBuffer   `json:"buffer"`
	ExpectedCompletion  string                `json:"expected_completion"`
	CompletionReturned  []ReturnedComp        `json:"completion_returned"`
	MatchingCompletions []*matchingCompletion `json:"matching_completions"`
	HasOnlyNames        bool                  `json:"has_only_names"` // Tells if all the expected arguments are NameExpr only
}

// ReturnedComp represents a completion returned by the completion API (string + rank only)
type ReturnedComp struct {
	Completion string `json:"completion"`
	Rank       int    `json:"rank"`
}

// ComputeMatchingCompletions takes a path to a file containing a list of records, compute completions for each
// of them and then match these completions against the arguments types by the user.
// completionAPI can be nil and will be automatically instanciated (param is there to avoid loading multiple times the model in the tests)
func ComputeMatchingCompletions(filename string, outputFile string, computeRecalls bool, useGGNN bool, completionAPI *api.API, maxSamples int) []CompletionResult {
	if completionAPI == nil {
		fail(datadeps.Enable())
		tensorflow.SetTensorflowThreadpoolSize(4)
		rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
		fail(<-errc)

		models, err := pythonmodels.New(pythonmodels.DefaultOptions)
		fail(err)

		completionAPITemp := api.New(context.Background(), api.Options{
			ResourceManager: rm,
			Models:          models,
		}, licensing.Pro)
		completionAPI = &completionAPITemp
	} else {
		completionAPI.Reset()
	}

	opts := api.IDCCCompleteOptions
	opts.MixOptions.GGNNSubtokenEnabled = useGGNN
	opts.ScheduleOptions.GGNNSubtokenEnabled = useGGNN
	opts.DepthLimit = 3
	opts.BlockDebug = true
	buffers := readPerLineJSON(filename)
	res, count := processBuffers(buffers, *completionAPI, opts, maxSamples)
	fmt.Println("number of completions returned are:", count)
	if outputFile != "" {
		jsonRes, err := json.MarshalIndent(res, "", " ")
		fail(err)
		err = ioutil.WriteFile(outputFile, jsonRes, 0644)
		fail(err)
	}
	if computeRecalls {
		exactMatch, placeholderMatch := ComputeBothRecalls(res, 5)
		recalls := RecallsInfo{
			ExactMatch:       RecallSet{Recalls: exactMatch},
			PlaceholderMatch: RecallSet{Recalls: placeholderMatch},
			Timestamp:        time.Now(),
		}
		fmt.Println(recalls)
	}
	return res
}

func readPerLineJSON(filename string) []string {
	var results []string
	in, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Error opening %s: %v", filename, err)
	}
	defer in.Close()

	decoder := json.NewDecoder(in)
	for {
		var rec record
		err := decoder.Decode(&rec)
		if err == io.EOF {
			break
		}
		fail(err)
		results = append(results, string(rec.Source))
	}
	return results
}

func parseCompletion(comp string) []*pythonast.Argument {
	comp = fmt.Sprintf("theFunk(%s)", comp)
	parseOpts := pythonparser.Options{
		ErrorMode:   pythonparser.Recover,
		Approximate: true,
	}

	ast, _, err := pythonpipeline.Parse(parseOpts, time.Second, sample.ByteSlice(comp))
	fail(err)

	expr, ok := ast.Body[0].(*pythonast.ExprStmt)
	if !ok {
		fmt.Printf("Invalid expected completion : %v\n", comp)
		return nil
	}

	return expr.Value.(*pythonast.CallExpr).Args
}

func processBuffers(buffers []string, compAPI api.API, opts api.CompleteOptions, maxSamples int) ([]CompletionResult, int) {
	var result []CompletionResult
	var count int
	for i, s := range buffers {
		r := processBuffer(s, compAPI, opts)
		result = append(result, r)
		count += len(r.CompletionReturned)
		if maxSamples > 0 && i >= maxSamples {
			break
		}
		if i > 0 && i%50 == 0 {
			fmt.Printf("%d / %d buffers processed\n", i, maxSamples)
		}
	}
	return result, count
}

func processBuffer(s string, compAPI api.API, opts api.CompleteOptions) CompletionResult {
	buffer, expComp, err := getSelectedBuffer(s)
	fail(err)
	compsS := getCompletions(compAPI, buffer, opts)
	var comps []ReturnedComp
	for i, comp := range compsS {

		if comp.Source != response.CallModelCompletionSource && comp.Source != response.EmptyCallCompletionSource &&
			comp.Source != response.GGNNModelFullCallSource && comp.Source != response.GGNNModelPartialCallSource &&
			comp.Source != response.GGNNModelAttributeSource {
			continue
		}
		comps = append(comps, ReturnedComp{
			Completion: comp.Snippet.Text,
			Rank:       i,
		})
	}
	expArgs := parseCompletion(expComp)
	matches := matchCompletions(expComp, comps)

	return CompletionResult{
		Buffer:              buffer,
		ExpectedCompletion:  expComp,
		CompletionReturned:  comps,
		MatchingCompletions: matches,
		HasOnlyNames:        isOnlyNames(expArgs),
	}
}

func isOnlyNames(arguments []*pythonast.Argument) bool {
	for _, a := range arguments {
		if _, ok := a.Value.(*pythonast.NameExpr); !ok {
			return false
		}
	}
	return true
}

func getSelectedBuffer(s string) (data.SelectedBuffer, string, error) {
	var sb data.SelectedBuffer
	var expectedComp string
	switch parts := strings.Split(s, "$"); len(parts) {
	case 3:
		sb = data.NewBuffer(parts[0]).Select(data.Cursor(len(parts[0])))
		expectedComp = parts[1]
	default:
		return sb, "", errors.Errorf("Not enough (or too many) $ in the buffer, please put exactly 2 around the expected completion")
	}
	return sb, expectedComp, nil
}

func matchCompletions(expComp string, completions []ReturnedComp) []*matchingCompletion {
	expArgs := parseCompletion(expComp)
	var result []*matchingCompletion
	for _, comp := range completions {
		completion := comp.Completion
		if strings.HasSuffix(completion, ")") {
			completion = completion[:len(completion)-1]
		}
		compArgs := parseCompletion(completion)
		match := matchCompletion(compArgs, expArgs, comp.Rank, completion)
		if match != nil {
			result = append(result, match)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ExactMatch != result[j].ExactMatch {
			return result[i].ExactMatch
		}
		if result[i].ConcretePlaceholderMatch != result[j].ConcretePlaceholderMatch {
			return result[i].ConcretePlaceholderMatch < result[j].ConcretePlaceholderMatch
		}
		return result[i].Rank < result[j].Rank
	})
	return result
}

func matchCompletion(argComp, argExp []*pythonast.Argument, rank int, comp string) *matchingCompletion {
	if len(argComp) > len(argExp) {
		return nil
	}
	exactMatch := true
	posExp := countPositional(argExp)
	posComp := countPositional(argComp)
	if posExp != posComp {
		return nil
	}

	var concretePlaceholderMatch int
	var placeholderCount int
	for i, ac := range argComp {
		if pythonast.IsNil(ac.Name) != pythonast.IsNil(argExp[i].Name) {
			return nil
		}
		if !pythonast.IsNil(ac.Name) {
			if ac.Name.(*pythonast.NameExpr).Ident.Literal != argExp[i].Name.(*pythonast.NameExpr).Ident.Literal {
				return nil
			}
		}
		pac := getArgValue(ac)
		if pac == "KITE_PLACEHOLDER" {
			placeholderCount++
		}
		if pac != getArgValue(argExp[i]) {
			if pac == "KITE_PLACEHOLDER" {
				concretePlaceholderMatch++
				exactMatch = false
			} else {
				return nil
			}
		} else if pac == "KITE_PLACEHOLDER" {
			exactMatch = false
			placeholderCount++
		}

	}
	return &matchingCompletion{
		Rank:                     rank,
		ConcretePlaceholderMatch: concretePlaceholderMatch,
		Completion:               comp,
		ExactMatch:               exactMatch,
		PlaceholderCount:         placeholderCount,
	}
}

func countPositional(args []*pythonast.Argument) interface{} {
	var count int
	for _, a := range args {
		if pythonast.IsNil(a.Name) {
			count++
		} else {
			break
		}
	}
	return count
}

func getArgValue(arg *pythonast.Argument) string {
	if name, ok := arg.Value.(*pythonast.NameExpr); ok {
		return name.Ident.Literal
	}
	return "KITE_PLACEHOLDER"
}

func getCompletions(compAPI api.API, buffer data.SelectedBuffer, options api.CompleteOptions) []data.RCompletion {
	req := data.APIRequest{
		SelectedBuffer: buffer,
		UMF:            data.UMF{Filename: "/test.py"},
	}
	var resp data.APIResponse
	err := kitectx.Background().WithTimeout(5*time.Second, func(ctx kitectx.Context) error {
		resp = compAPI.Complete(ctx, options, req, nil, nil)
		return resp.ToError()
	})
	if err != nil {
		fmt.Printf("Error while computing completions : %v\n", err)
	}
	defer compAPI.Reset()
	var compls []data.RCompletion
	for _, nrc := range resp.Completions {
		compls = append(compls, nrc.RCompletion)
	}
	return compls
}

// ComputeBothRecalls computes recalls for exact match and placeholder matchs and return results as a 2 arrays of float
func ComputeBothRecalls(results []CompletionResult, maxRecall int) ([]float32, []float32) {
	recExact := computeRecalls(results, maxRecall, false, true, false)
	recMatch := computeRecalls(results, maxRecall, false, false, false)
	return recExact, recMatch
}

// computeRecalls compute recalls until maxRecall based on the results provided
// signatureMatch will consider it's a match by only checking what parameters are present (just need the same amount of positional arguments and same set of keyword arguments)
// Exact match only consider expected completion that only have name arguments in them and do an exact match, so text have to be exactly the same
// When both boolean are at false (standard match), names arguments have to be the same, and if the expected arg is not a name (functio call, constant or attribute expression), the completion has to be a placeholder for a match.
func computeRecalls(results []CompletionResult, maxRecall int, signatureMatch, onlyExactMatch bool, printRecalls bool) []float32 {
	var recalls []float32
	for i := 1; i <= maxRecall; i++ {
		r := computeRecall(i, results, signatureMatch, onlyExactMatch)
		recalls = append(recalls, r)
		if printRecalls {
			fmt.Printf("%v\n", r)
		}
	}
	return recalls
}

func computeRecall(maxRank int, results []CompletionResult, signatureMatch, onlyExactMatch bool) float32 {
	var count float32
	var size float32
	for _, r := range results {
		var found bool
		for _, comp := range r.MatchingCompletions {
			if (comp.Rank < maxRank) &&
				(signatureMatch || comp.ConcretePlaceholderMatch == 0) &&
				(!onlyExactMatch || comp.ExactMatch) {
				count += 1.0
				size++
				found = true
				break
			}
		}
		if !found && (!onlyExactMatch || r.HasOnlyNames) {
			size++
		}
	}
	if size > 0 {
		return count / size
	}
	fmt.Println("No sample with only name arguments, recall not computed")
	return -1
}

func computeAndCompareRecalls(datasetPath string, baseline RecallsInfo, useGGNN bool, api *api.API, maxSamples int) []string {
	fmt.Printf("Computing %d recalls\n", len(baseline.ExactMatch.Recalls))
	completionMatches := ComputeMatchingCompletions(datasetPath, "", false, useGGNN, api, maxSamples)
	exactRecalls, placeholderRecalls := ComputeBothRecalls(completionMatches, len(baseline.ExactMatch.Recalls))
	return compareRecalls(baseline, exactRecalls, placeholderRecalls)
}

func compareRecalls(baseline RecallsInfo, exactMatch, placeholdersMatch []float32) []string {
	var result []string
	result = append(result, compareRecall(baseline.ExactMatch, exactMatch, "Exact match")...)
	result = append(result, compareRecall(baseline.PlaceholderMatch, placeholdersMatch, "Match allowing placeholder")...)
	return result
}

func compareRecall(baseline RecallSet, values []float32, matchType string) []string {
	var result []string
	for i, rv := range values {
		rb := baseline.Recalls[i]
		if rv == 0 {
			result = append(result, fmt.Sprintf("%s recall %d is 0", matchType, i+1))
			continue
		}
		if delta := (rv - rb) / rb; rv > rb && delta > baseline.RelativeThreshold {
			result = append(result, fmt.Sprintf("%s recall %d is too high (%.2f, baseline being %.2f, relative diff of %.1f%%, relative threshold set to %.1f%%), please update the expected recall in the test file", matchType, i+1, rv, rb, delta*100, baseline.RelativeThreshold*100))
			baseline.Recalls[i] = rv
		}
		if delta := (rb - rv) / rv; rb > rv && delta > baseline.RelativeThreshold {
			result = append(result, fmt.Sprintf("%s recall %d is too low (%.2f, baseline being %.2f, relative diff of %.1f%%, relative threshold set to %.1f%%), please fix model or update the expected recall in the test file", matchType, i+1, rv, rb, delta*100, baseline.RelativeThreshold*100))
			baseline.Recalls[i] = rv
		}
		if delta := rv - rb; rv > rb && delta > baseline.AbsoluteThreshold {
			result = append(result, fmt.Sprintf("%s recall %d is too high (%.2f, baseline being %.2f, absolute diff of %.1f%%, absolute threshold set to %.1f%%), please update the expected recall in the test file", matchType, i+1, rv, rb, delta*100, baseline.AbsoluteThreshold*100))
			baseline.Recalls[i] = rv
		}
		if delta := rb - rv; rb > rv && delta > baseline.AbsoluteThreshold {
			result = append(result, fmt.Sprintf("%s recall %d is too low (%.2f, baseline being %.2f, absolute diff of %.1f%%, absolute threshold set to %.1f%%), please fix model or update the expected recall in the test file", matchType, i+1, rv, rb, delta*100, baseline.AbsoluteThreshold*100))
			baseline.Recalls[i] = rv
		}
	}
	return result
}
