package main

import (
	"sort"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/example"
)

type listingResult struct {
	Rank    int
	Present bool
	Found   bool
	Best    bool
	Correct bool
}

type listing struct {
	ID       int
	Symbol   string
	Expected string
	Results  []listingResult
}

type listingSet struct {
	Providers []string
	Listings  []listing
}

func newListingSet(collection example.Collection, filterSymbol string) listingSet {
	providers := providerNames(collection)

	listings := make([]listing, 0, len(collection.Examples))

	for i, ex := range collection.Examples {
		if filterSymbol != "" && ex.Symbol != filterSymbol {
			continue
		}

		results := resultsForExample(ex, providers)

		listings = append(listings, listing{
			ID:       i,
			Symbol:   ex.Symbol,
			Expected: ex.Expected,
			Results:  results,
		})
	}

	return listingSet{
		Providers: providers,
		Listings:  listings,
	}
}

func providerNames(collection example.Collection) []string {
	provNames := make(map[string]struct{})
	for _, ex := range collection.Examples {
		for c := range ex.Provided {
			provNames[c] = struct{}{}
		}
	}

	providers := make([]string, 0, len(provNames))
	for p := range provNames {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

func resultsForExample(ex example.Example, providers []string) []listingResult {
	results := make([]listingResult, 0, len(providers))

	for _, prov := range providers {
		res := listingResult{}
		if _, found := ex.Provided[prov]; found {
			res.Present = true
		}
		res.Rank = ex.Rank(prov)
		if res.Rank >= 0 {
			res.Found = true
		}
		if res.Rank == 0 {
			res.Correct = true
		}
		results = append(results, res)
	}

	bestRank := -1
	for _, res := range results {
		if res.Found && (bestRank < 0 || res.Rank < bestRank) {
			bestRank = res.Rank
		}
	}

	if bestRank >= 0 {
		for i, res := range results {
			if res.Found && res.Rank == bestRank {
				results[i].Best = true
			}
		}
	}

	return results
}
