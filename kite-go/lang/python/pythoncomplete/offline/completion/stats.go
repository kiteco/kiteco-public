package completion

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/example"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/sbwhitecap/tqdm"
	"github.com/sbwhitecap/tqdm/iterators"
)

// Stats is a collection of Stat
type Stats struct {
	Stats []Stat `json:"stats"`
}

// Stat contains information about and example and all the completions generated for it
// and some basics stats on it (count of completion per provider, total number of completions)
type Stat struct {
	Example         example.Example `json:"-"`
	Description     string          `json:"description"`
	ProviderCount   map[string]int  `json:"provider_count"`
	CompletionCount uint            `json:"completion_count"`
}

var specificIndex = -1

// ComputeStats generate completions for the collection of examples and generate Stats for each of them
func ComputeStats(examples example.Collection, provider *Provider) Stats {
	selectedSlice := examples.Examples
	if specificIndex != -1 {
		selectedSlice = selectedSlice[specificIndex : specificIndex+1]
	}
	stats := make([]Stat, 0, len(examples.Examples))

	err := tqdm.With(iterators.Interval(0, len(selectedSlice)), "Computing completions", func(c interface{}) (brk bool) {
		cnt := c.(int)
		ex := selectedSlice[cnt]
		stat := computeStat(ex, provider)
		stats = append(stats, stat)
		return
	})
	maybeQuit(err)
	return Stats{stats}
}

func computeStat(ex example.Example, provider *Provider) Stat {
	completions := provider.GetNRCompletions(ex, false, false)
	providerCounts := countProviders(completions)
	description := ex.Symbol
	result := Stat{
		Example:         ex,
		ProviderCount:   providerCounts,
		Description:     description,
		CompletionCount: countCompletions(completions),
	}
	return result
}

func countCompletions(completions []data.NRCompletion) uint {
	var counter uint
	completionMap(completions, func(completion *data.RCompletion) {
		counter++
	})
	return counter
}

func completionMap(completions []data.NRCompletion, functor func(completion *data.RCompletion)) {
	for ii := range completions {
		c := &completions[ii]
		functor(&c.RCompletion)
	}
}

func countProviders(completions []data.NRCompletion) map[string]int {
	result := make(map[string]int)
	completionMap(completions, func(completion *data.RCompletion) {
		result[string(completion.Source)]++
	})
	return result
}

func maybeQuit(err error) {
	if err != nil {
		panic(err)
	}
}
