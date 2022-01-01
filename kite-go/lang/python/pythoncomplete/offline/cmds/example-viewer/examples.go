package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
)

func (a *app) handleExamples(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	sortParam := params.Get("sort")
	symbolParam := params.Get("symbol")

	ls := newListingSet(a.collection, symbolParam)
	if sortParam != "" {
		ls = ls.SortedByProvider(sortParam)
	}

	byProvider := ls.ProviderBreakdowns()

	urlForParam := func(param string) string {
		u := "/examples?"
		for _, p := range []string{"sort", "symbol"} {
			if p == param {
				continue
			}
			val := params.Get(p)
			if val == "" {
				continue
			}
			u += fmt.Sprintf("%s=%s&", p, url.QueryEscape(val))
		}
		u += param + "="
		return u
	}

	err := a.templates.Render(w, "examples.html", map[string]interface{}{
		"ByProvider": byProvider,
		"Listings":   ls,
		"Path":       a.collection.Path,
		"Count":      len(ls.Listings),
		"TotalCount": len(a.collection.Examples),
		"ProvURL":    urlForParam("sort"),
		"SymbolURL":  urlForParam("symbol"),
		"Symbol":     symbolParam,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (l listingSet) SortedByProvider(provider string) listingSet {
	type withRank struct {
		listing listing
		order   int
	}

	wr := make([]withRank, 0, len(l.Listings))
	for _, listing := range l.Listings {
		wr = append(wr, withRank{
			listing: listing,
			order:   bestSortOrder(listing, l.Providers, provider),
		})
	}

	sort.Slice(wr, func(i, j int) bool {
		if wr[i].order == wr[j].order {
			return wr[i].listing.ID < wr[j].listing.ID
		}
		return wr[i].order < wr[j].order
	})

	sorted := make([]listing, 0, len(wr))
	for _, w := range wr {
		sorted = append(sorted, w.listing)
	}

	return listingSet{
		Providers: l.Providers,
		Listings:  sorted,
	}
}

func bestSortOrder(listing listing, providers []string, provider string) int {
	resIdx := -1
	var res listingResult

	for i, p := range providers {
		if p == provider {
			resIdx = i
			res = listing.Results[i]
			break
		}
	}

	if resIdx < 0 {
		return 6
	}

	if !res.Found {
		return 5
	}

	if !res.Best {
		return 4
	}

	var otherBest bool
	for i, r := range listing.Results {
		if i == resIdx {
			continue
		}
		if r.Found && r.Rank <= res.Rank {
			otherBest = true
			break
		}
	}

	if otherBest {
		if res.Correct {
			return 2
		}
		return 3
	}

	if res.Correct {
		return 0
	}
	return 1
}
