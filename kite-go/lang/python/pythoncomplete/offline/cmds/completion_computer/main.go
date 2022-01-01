package main

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/sbwhitecap/tqdm/iterators"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/scoring"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/sbwhitecap/tqdm"
	"github.com/syndtr/goleveldb/leveldb/util"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/completion"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/example"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

// CompletionSituation caracterize the situation in which the completion is made
type CompletionSituation string

const (
	// NoArg represent a call completion when no arg have written for the function yet
	NoArg CompletionSituation = "NO_ARG"
)

func maybeQuit(err error) {
	if err != nil {
		panic(err)
	}
}

type completionCase struct {
	// Hash of the buffer and expected result of the example
	ExampleID uint32 `json:"example_id"`
	// Scoring data of the completion (see CompletionScore struct)
	Score scoring.CompletionScore `json:"score"`
	// Kind of completion (no arg yet, already one arg, named arg, etc.)
	Situation CompletionSituation `json:"situation"`
	// Provider for this completion
	Source string `json:"source"`
	// Rank of this completion in this provider
	Rank int `json:"rank"`
	// Number of completion from this provider
	CompletionCount int `json:"completion_count"`

	CompletionSnippet string `json:"completion_snippet"`

	Expected string `json:"expected"`
}

func main() {
	args := struct {
		Dir string `arg:"positional,required"`
	}{}
	arg.MustParse(&args)

	maybeQuit(datadeps.Enable())
	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	maybeQuit(<-errc)

	completionProvider := completion.NewProvider(rm)

	collection, err := example.NewCollection(args.Dir)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("found %d examples in %s", len(collection.Examples), args.Dir)
	if len(collection.Examples) == 0 {
		log.Fatal("no examples found")
	}

	computeAndWriteScores("/data/kite/mixing/scores.json", collection, completionProvider)

}

func scoreExample(ex example.Example, provider *completion.Provider) ([]completionCase, error) {
	completions := provider.GetCompletions(ex)

	exID := util.Hash([]byte(ex.Buffer), 0)
	result := make([]completionCase, 0, len(completions))

	var source string
	var providerRank int
	countPerSource := make(map[string]int)
	for _, comp := range completions {
		if actual := string(comp.MixCompletion.MetaCompletion().Source); source != actual {
			source = actual
			if _, ok := countPerSource[source]; ok {
				return nil, errors.New("Error, some completions share the same source but are not contiguous in the result slice (source = %s)", source)
			}
			providerRank = 1
		}
		score, err := scoring.ScoreCompletion(comp, ex.Expected)
		if err != nil {
			return nil, err
		}
		result = append(result, completionCase{
			ExampleID:         exID,
			Source:            source,
			Score:             score,
			Rank:              providerRank,
			Situation:         "ArgBeginInCall",
			Expected:          ex.Expected,
			CompletionSnippet: comp.Identifier,
		})
		providerRank++
		countPerSource[source]++
	}
	for i := range result {
		result[i].CompletionCount = countPerSource[result[i].Source]
	}
	return result, nil
}

func scoreCompletions(examples example.Collection, provider *completion.Provider) ([]completionCase, error) {
	result := make([]completionCase, 0)
	err := tqdm.With(iterators.Interval(0, len(examples.Examples)), "Scoring examples ", func(c interface{}) (brk bool) {
		idx := c.(int)
		ex := examples.Examples[idx]
		cases, err := scoreExample(ex, provider)
		if err == nil {
			result = append(result, cases...)
		}
		return
	})
	maybeQuit(err)
	return result, nil
}

func computeAndWriteScores(targetFile string, examples example.Collection, provider *completion.Provider) {
	scores, err := scoreCompletions(examples, provider)
	maybeQuit(err)
	jsonContent, err := json.MarshalIndent(scores, "", "  ")
	maybeQuit(err)
	err = ioutil.WriteFile(targetFile, jsonContent, 0644)
	maybeQuit(err)

}
