package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"

	"github.com/kiteco/kiteco/kite-go/curation"
)

// googleSuggestion is one search query returned by the Google Suggest API.
type googleSuggestion struct {
	Data string `xml:"data,attr"`
}

// googleResults is an entire response returned by the Google Suggest API.
type googleResults struct {
	Suggestions []googleSuggestion `xml:"CompleteSuggestion>suggestion"`
}

// asStrings returns the suggestions in a GoogleResults object as plain strings for ease of use.
func (g *googleResults) asStrings() []string {
	var strings []string
	for _, s := range g.Suggestions {
		strings = append(strings, s.Data)
	}
	return strings
}

func parseGoogle(name, lang, source string, data []byte) (*curation.Suggestions, error) {
	// remove any invalid characters that are not UTF encoded
	data = curation.ValidUTF(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("error parsing google data, length 0")
	}

	var r googleResults
	err := xml.Unmarshal(data, &r)
	if err != nil {
		return nil, fmt.Errorf("error while parsing google xml: %v", err)
	}

	return &curation.Suggestions{
		Ident:       name,
		Language:    lang,
		Source:      source,
		Suggestions: r.asStrings(),
	}, nil
}

func parseBing(name, lang, source string, data []byte) (*curation.Suggestions, error) {
	// remove any invalid characters that are not UTF encoded
	data = curation.ValidUTF(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("error parsing bing data, length 0")
	}

	// transform data into json format
	data = bytes.Replace(data, []byte(",["), []byte(","), -1)
	data = bytes.Replace(data, []byte("]]"), []byte("]"), -1)
	data = bytes.Replace(data, []byte(",]"), []byte("]"), -1)

	var suggestions []string
	err := json.Unmarshal(data, &suggestions)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling Bing response: %v", err)
	}

	if len(suggestions) == 0 {
		return nil, fmt.Errorf("bing returned array of 0 length")
	}

	return &curation.Suggestions{
		Ident:       name,
		Language:    lang,
		Source:      source,
		Suggestions: suggestions[1:],
	}, nil
}

func parseSuggestions(name, lang, source string, data []byte) (*curation.Suggestions, error) {
	switch source {
	case "google":
		return parseGoogle(name, lang, source, data)
	case "bing":
		return parseBing(name, lang, source, data)
	default:
		return nil, fmt.Errorf("unknown source: %s", source)
	}
}

// --

// Query is a convenience method to construct a url-encoded query string to query the suggestions APIs.
func constructQuery(name string) string {
	return url.QueryEscape(strings.Replace(name, ".", " ", -1))
}
