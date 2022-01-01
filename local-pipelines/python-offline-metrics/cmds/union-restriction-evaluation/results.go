package main

import (
	"fmt"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
)

func getDeltaCount(data pairCompAggregation) (map[int]int, []int) {
	result := make(map[int]int)
	for k, m := range data {
		for k2, c := range m {
			if k2 != -1 && k != -1 {
				result[k2-k] += c
			}
		}
	}
	keys := make([]int, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return result, keys
}

func buildDeltaTable(data pairCompAggregation, drawer *rundb.HistogramDrawer, name string) string {
	deltas, order := getDeltaCount(data)
	var result string
	if drawer != nil {
		histData, min, max := rundb.GetHistogramDataFromFrequencyMap(deltas)
		histStr, err := drawer.GetHistogramString(histData, 400, 400, name, float32(min), float32(max))
		if err != nil {
			fmt.Println("Error while generating an histrogram : ", err)
		} else {
			result += histStr + "</br>"
		}
	}

	result += `<table style="width:100%"> <tr> <th> Delta </th> <th> Count </th> </tr>`
	for _, k := range order {
		result += fmt.Sprintf("<tr> <td> %d </td> <td> %d </td> </tr>", k, deltas[k])
	}
	result += "</table>"

	return result
}

func buildHTMLTable(data pairCompAggregation) string {
	result := `<table style="width:100%"> <tr> <th> Before </th> <th> After </th> <th> Count </th> </tr>`
	keys := make([]int, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		values := data[k]
		keys2 := make([]int, 0, len(values))
		for k2 := range values {
			keys2 = append(keys2, k2)
		}
		sort.Ints(keys2)
		for _, k2 := range keys2 {
			result += fmt.Sprintf("<tr> <td> %d </td> <td> %d </td> <td> %d </td> </tr>", k, k2, values[k2])
		}
	}
	result += "</table>"

	return result
}

func compileResults(repoName string) func(results map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
	return func(results map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
		drawer, err := rundb.NewHistogramDrawer()
		if err != nil {
			fmt.Println("Error while building the histrogramDrawer : ", err)
		}

		var result []rundb.Result
		for agg, res := range results {
			result = append(result, rundb.Result{
				Name:       fmt.Sprintf("Delta for %s", agg.Name()),
				Value:      buildDeltaTable(res.(pairCompAggregation), drawer, agg.Name()),
				Aggregator: agg.Name(),
			})
		}

		for agg, res := range results {
			result = append(result, rundb.Result{
				Name:       agg.Name(),
				Value:      buildHTMLTable(res.(pairCompAggregation)),
				Aggregator: agg.Name(),
			})
		}
		return result
	}
}
