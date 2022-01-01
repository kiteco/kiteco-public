package search

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/stackoverflow"
)

// CloudSearchIndex is an index that wraps AWS cloudsearch.
type CloudSearchIndex struct {
	endpointURL *url.URL
}

// NewCloudSearchIndex returns a new CloudSearchIndex or err if endpoint url is invalid.
func NewCloudSearchIndex(endpoint string) (*CloudSearchIndex, error) {
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	return &CloudSearchIndex{
		endpointURL: endpointURL,
	}, nil
}

// Search implements the Index interface.
func (csi *CloudSearchIndex) Search(query string, st stackoverflow.SearchType, numResults int) ([]int64, error) {
	queryURL, err := csi.endpointURL.Parse("/2013-01-01/search")
	if err != nil {
		return nil, err
	}

	values := make(url.Values)
	parts := strings.Split(cleanString(query), " ")
	switch st {
	case stackoverflow.Disjunction:
		values.Add("q", strings.Join(parts, "|"))
	case stackoverflow.Conjunction:
		values.Add("q", strings.Join(parts, "+"))
	}
	values.Add("size", strconv.Itoa(numResults))
	values.Add("return", "_no_fields")

	queryURL.RawQuery = values.Encode()

	resp, err := http.Get(queryURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response csResponse
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&response)
	if err != nil {
		return nil, err
	}

	var ids []int64
	for _, hit := range response.Hits.Hit {
		id, err := strconv.ParseInt(hit.ID, 10, 64)
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

type csHit struct {
	ID     string            `json:"id"`
	Fields map[string]string `json:"fields"`
}

type csHits struct {
	Hit []csHit `json:"hit"`
}

type csResponse struct {
	Hits csHits `json:"hits"`
}

// TODO(juan) faster way to do this?
// TODO(juan) what to replace with?
// TODO(juan) how handle variations e.g ' and ’
func cleanString(s string) string {
	s = strings.Replace(s, "/", "", -1)
	s = strings.Replace(s, "'", "", -1)
	s = strings.Replace(s, "`", "", -1)
	s = strings.Replace(s, "’", "", -1)
	s = strings.Replace(s, ">", "", -1)
	s = strings.Replace(s, "<", "", -1)
	s = strings.Replace(s, "&", "", -1)
	s = strings.Replace(s, "?", "", -1)
	return s
}
