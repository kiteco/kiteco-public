package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
)

const (
	bingAPIKey      = "XXXXXXX"
	bingEndpointFmt = "https://api.datamarket.azure.com/Bing/Search/Web?$format=json&$top=%d&Query='site:%s+%s'"
)

// bingSearcher is a searcher that uses the bing api.
type bingSearcher struct {
	httpClient *http.Client
}

func newBingSearcher() bingSearcher {
	return bingSearcher{
		httpClient: &http.Client{},
	}
}

// Search satisfies the searcher interface.
func (bs bingSearcher) Search(query string, st stackoverflow.SearchType, maxResults int) ([]int64, error) {
	req, err := makeAPIRequest(query, maxResults)
	if err != nil {
		return nil, err
	}

	resp, err := bs.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %s", err)
	}

	var d bingResultsDocument
	err = json.Unmarshal(buf, &d)
	if err != nil {
		log.Println("error unmarshaling json:", err)
		return nil, fmt.Errorf("error unmarshaling json: %s", err)
	}

	var ids []int64
	for _, result := range d.D.Results {
		ids = append(ids, idFromURL(result.URL))
	}

	return ids, nil
}

// --

// bingResult contains each result returned by the API
type bingResult struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// bingResults contains all the actual page results
type bingResults struct {
	Results []*bingResult `json:"results"`
}

// bingResultsDocument is the top level of the json returned by the API
type bingResultsDocument struct {
	D bingResults `json:"d"`
}

// --

// makeAPIURL takes a query and builds a API compliant URL
func makeAPIURL(query string, maxResults int) (*url.URL, error) {
	q := url.QueryEscape(query)
	u := fmt.Sprintf(bingEndpointFmt, maxResults, "stackoverflow.com", q)
	parsedURL, err := url.Parse(u)
	if err != nil {
		log.Println("cannot parse url:", err)
		return nil, err
	}
	return parsedURL, nil
}

// makeAPIRequest makes the http.Request object for the API request
func makeAPIRequest(query string, maxResults int) (*http.Request, error) {
	apiURL, err := makeAPIURL(query, maxResults)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", apiURL.String(), nil)
	if err != nil {
		log.Println("error making request:", err)
		return nil, err
	}

	req.SetBasicAuth(bingAPIKey, bingAPIKey)
	return req, nil
}
