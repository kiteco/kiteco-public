package main

import "sort"

type providerBreakdown struct {
	Provider string

	Total   int
	Best    int
	Correct int

	BestPercent    float64
	CorrectPercent float64
	IsBest         bool
}

func computePrecentages(breakdowns []providerBreakdown) []providerBreakdown {
	var maxBestPercent float64
	maxBPIdx := -1

	ret := make([]providerBreakdown, 0, len(breakdowns))

	for i, pb := range breakdowns {
		if pb.Total == 0 {
			continue
		}

		pb.BestPercent = float64(pb.Best*100) / float64(pb.Total)
		if pb.BestPercent > maxBestPercent {
			maxBestPercent = pb.BestPercent
			maxBPIdx = i
		}
		pb.CorrectPercent = float64(pb.Correct*100) / float64(pb.Total)

		ret = append(ret, pb)
	}

	for i := range breakdowns {
		if i == maxBPIdx {
			ret[i].IsBest = true
		}
	}

	return ret
}

func (l listingSet) ProviderBreakdowns() []providerBreakdown {
	breakdowns := make([]providerBreakdown, 0, len(l.Providers))
	for _, p := range l.Providers {
		breakdowns = append(breakdowns, providerBreakdown{Provider: p})
	}

	for _, listing := range l.Listings {
		for i, res := range listing.Results {
			if res.Present {
				breakdowns[i].Total++
			}
			if res.Best {
				breakdowns[i].Best++
			}
			if res.Correct {
				breakdowns[i].Correct++
			}
		}
	}

	return computePrecentages(breakdowns)
}

type symbolBreakdown struct {
	Symbol    string
	Count     int
	Providers []providerBreakdown
}

func (l listingSet) SymbolBreakdowns(sortByProvider string) []symbolBreakdown {
	bySymbol := make(map[string]symbolBreakdown)

	for _, listing := range l.Listings {
		bd, found := bySymbol[listing.Symbol]
		if !found {
			bd = symbolBreakdown{
				Symbol:    listing.Symbol,
				Providers: make([]providerBreakdown, 0, len(l.Providers)),
			}
			for _, p := range l.Providers {
				bd.Providers = append(bd.Providers, providerBreakdown{Provider: p})
			}
		}
		bd.Count++

		for i, res := range listing.Results {
			if res.Present {
				bd.Providers[i].Total++
			}
			if res.Best {
				bd.Providers[i].Best++
			}
			if res.Correct {
				bd.Providers[i].Correct++
			}
		}

		bySymbol[listing.Symbol] = bd
	}

	all := make([]symbolBreakdown, 0, len(bySymbol))
	for _, bd := range bySymbol {
		bd.Providers = computePrecentages(bd.Providers)
		all = append(all, bd)
	}

	provSortIdx := -1
	for i, p := range l.Providers {
		if p == sortByProvider {
			provSortIdx = i
			break
		}
	}

	sort.Slice(all, func(i, j int) bool {
		if sortByProvider != "" && provSortIdx >= 0 {
			p1 := all[i].Providers[provSortIdx]
			p2 := all[j].Providers[provSortIdx]

			if p1.IsBest && !p2.IsBest {
				return true
			}
			if p2.IsBest && !p1.IsBest {
				return false
			}
		}

		if all[i].Count == all[j].Count {
			return all[i].Symbol < all[j].Symbol
		}
		return all[i].Count > all[j].Count
	})

	return all
}
