package main

import (
	"log"
	"strconv"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/internal/search"
)

// Searcher encapsulates a stackoverflow searcher.
type searcher struct {
	index        search.Index
	pageFinder   search.PageFinder
	ranker       *search.Ranker
	langDetector *search.LanguageDetector
	resultFilter *search.ResultFilter
	// missed stores ids of pages that were returned by the index
	// but were not found by the PageFinder.
	missed map[int64]struct{}
}

// Search returns the results for a given query.
func (s searcher) Search(query string, st stackoverflow.SearchType, numResults int) ([]stackoverflow.SearchResult, error) {
	lang, found := s.langDetector.Detect(query)
	if !found {
		query = lang + " " + query
	}
	ids, err := s.index.Search(query, st, numResults)
	if err != nil {
		return nil, err
	}
	var pages []*stackoverflow.StackOverflowPage
	for _, id := range ids {
		page, err := s.pageFinder.Find(id)
		if err != nil {
			log.Println("error for id:" + strconv.FormatInt(id, 10) + ", error msg: " + err.Error())
			s.missed[id] = struct{}{}
			continue
		}
		pages = append(pages, page)
	}
	pages = s.resultFilter.Filter(query, lang, pages)
	s.ranker.Rank(query, pages)
	results := make([]stackoverflow.SearchResult, len(pages))
	for i, page := range pages {
		results[i] = stackoverflow.SearchResult{
			ID: page.GetQuestion().GetPost().GetId(),
		}
	}
	return results, nil
}
