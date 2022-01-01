package stackoverflow

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// SearchResults is a collection of StackOverflow search results
// from a particular source.
type SearchResults struct {
	Query   string
	Source  string
	Results []SearchResult
}

// SearchResult is an individual search result item
type SearchResult struct {
	ID      int64
	Title   string
	URL     string
	Snippet string
}

// SearchGoogle queries Google for StackOverflow results for the given query.
func SearchGoogle(query string) (*SearchResults, error) {
	q := strings.Join(strings.Split(strings.Trim(query, " \n"), " "), "+")
	url := fmt.Sprintf("http://www.google.com/search?q=site:stackoverflow.com+" + q)

	results := &SearchResults{
		Query:  query,
		Source: "google",
	}

	doc, err := goquery.NewDocument(url)
	if err != nil {
		return nil, err
	}

	doc.Find("li.g").Each(func(i int, s *goquery.Selection) {
		title := s.Find("h3.r").Find("a").Not(".l").First().Text()
		url, _ := s.Find("h3.r").Find("a").Not(".l").First().Attr("href")
		snippet := s.Find("span.st").First().Text()
		if title == "" || url == "" || snippet == "" {
			log.Println("for query:", query, "one of title, url, snippet is empty. DOM changed?")
		}
		url = sanitizeURL(url)
		id, err := idFromURL(url)
		if err != nil {
			id = 0
		}
		results.Results = append(results.Results, SearchResult{
			ID:      id,
			Title:   title,
			URL:     sanitizeURL(url),
			Snippet: snippet,
		})
	})

	return results, nil
}

func idFromURL(u string) (int64, error) {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return 0, err
	}
	var id int64
	var discard string
	_, err = fmt.Sscanf(parsedURL.Path, "/questions/%d/%s", &id, &discard)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func sanitizeURL(u string) string {
	if strings.HasPrefix(u, "/url?") {
		u = strings.TrimPrefix(u, "/url?")
		v, _ := url.ParseQuery(u)
		return v.Get("q")
	}
	return u
}
