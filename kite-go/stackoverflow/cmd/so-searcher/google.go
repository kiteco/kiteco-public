package main

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/kiteco/kiteco/kite-go/stackoverflow"
)

// googleSearcher is a searcher that uses the google api.
type googleSearcher struct{}

func newGoogleSearcher() googleSearcher {
	return googleSearcher{}
}

// Search satisfies the searcher interface.
func (gs googleSearcher) Search(query string, st stackoverflow.SearchType, maxResults int) ([]int64, error) {
	return searchGoogle(query), nil
}

func idFromURL(u string) int64 {
	parsedURL, _ := url.Parse(u)
	var id int64
	var discard string
	_, err := fmt.Sscanf(parsedURL.Path, "/questions/%d/%s", &id, &discard)
	if err != nil {
		log.Println("cannot extract id from url:", parsedURL.String())
		return 0
	}
	return id
}

func sanitizeURL(u string) string {
	if strings.HasPrefix(u, "/url?") {
		u = strings.TrimPrefix(u, "/url?")
		v, _ := url.ParseQuery(u)
		return v.Get("q")
	}
	return u
}

func searchGoogle(query string) []int64 {
	q := strings.Join(strings.Split(strings.Trim(query, " \n"), " "), "+")
	url := fmt.Sprintf("http://www.google.com/search?q=site:stackoverflow.com+" + q)
	doc, _ := goquery.NewDocument(url)

	var ids []int64
	doc.Find("li.g").Each(func(i int, s *goquery.Selection) {
		url, _ = s.Find("h3.r").Find("a").Not(".l").First().Attr("href")
		url = sanitizeURL(url)
		ids = append(ids, idFromURL(url))
	})

	return ids
}
