package main

import (
	"encoding/json"
	"log"
	"math"
	"os"
)

type synonymChart map[string][]string

// loadSynonyms loads a map of tag synonyms
func loadSynonyms(path string) synonymChart {
	in, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	decoder := json.NewDecoder(in)
	var synonyms synonymChart

	err = decoder.Decode(&synonyms)
	if err != nil {
		log.Fatal(err)
	}
	return synonyms
}

// buildSynonymChart maps the synonyms of a pakcage to the package, so that it's
// easier to look up what package a post refers to.
func buildSynonymChart(synonyms synonymChart, packages []string) synonymChart {
	packageTags := make(synonymChart)
	for _, p := range packages {
		packageTags[p] = append(packageTags[p], p)
		for _, syn := range synonyms[p] {
			packageTags[syn] = append(packageTags[syn], p)
		}
	}
	for _, syn := range synonyms["python"] {
		packageTags[syn] = append(packageTags[syn], "python")
	}
	packageTags["python"] = append(packageTags["python"], "python")

	// we hard code some synonyms that are not included in SO's list of tag synonyms
	packageTags["zip"] = append(packageTags["zip"], "zipfile")

	return packageTags
}

// match returns true if the target string is found in the list of candiates.
func match(target string, candidates []string) bool {
	for _, c := range candidates {
		if c == target {
			return true
		}
	}
	return false
}

func entropy(scores []float64) float64 {
	min := findMin(scores) - 0.0001
	var total float64
	for i, s := range scores {
		scores[i] = s - min
		total += scores[i]
	}
	var h float64
	for _, s := range scores {
		h += math.Log2(s/total) * (s / total)
	}
	return -h
}

// findMin returns the min value of an array.
func findMin(array []float64) float64 {
	min := math.Inf(1)
	for _, a := range array {
		if a < min {
			min = a
		}
	}
	return min
}
