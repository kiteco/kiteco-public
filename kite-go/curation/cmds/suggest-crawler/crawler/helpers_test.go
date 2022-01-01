package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	sampleGoogleResp = `<?xml version="1.0"?>
							<toplevel>
								<CompleteSuggestion><suggestion data="simplejson"/></CompleteSuggestion>
								<CompleteSuggestion><suggestion data="simplejson vs json"/></CompleteSuggestion>
								<CompleteSuggestion><suggestion data="simplejson dumps"/></CompleteSuggestion>
								<CompleteSuggestion><suggestion data="simplejson unity"/></CompleteSuggestion>
								<CompleteSuggestion><suggestion data="simplejson java"/></CompleteSuggestion>
								<CompleteSuggestion><suggestion data="simplejson dump to file"/></CompleteSuggestion>
								<CompleteSuggestion><suggestion data="simplejson vs ujson"/></CompleteSuggestion>
								<CompleteSuggestion><suggestion data="simplejson load from file"/></CompleteSuggestion>
								<CompleteSuggestion><suggestion data="simplejson documentation"/></CompleteSuggestion>
								<CompleteSuggestion><suggestion data="simplejson django"/></CompleteSuggestion>
							</toplevel>`
	sampleGoogleParsed = []string{"simplejson", "simplejson vs json", "simplejson dumps", "simplejson unity",
		"simplejson java", "simplejson dump to file", "simplejson vs ujson",
		"simplejson load from file", "simplejson documentation", "simplejson django"}

	sampleBingResp = `["simplejson",["simplejson","simplejson python","simplejson load encoding","simplejson c\u002b\u002b",
							"simplejson c\u0023 example","simple json parse without quotes",
							"simplejson property name without quotes","simplejson datetime","simple json example",
							"simplejson unity","simplejson golang","simplejson.load","simple json java",
							"simplejson python tutorial","simplejson vs json","simplejson example .net c\u0023",
							"simplejson dumps","simplejson download","simplejson documentation","simplejson.deserializeobject",
							"simple json decode","simplejson django","simplejson library","simplejson python windows",
							"simplejson windows"]]`
	sampleBingParsed = []string{"simplejson", "simplejson python", "simplejson load encoding", "simplejson c\u002b\u002b",
		"simplejson c\u0023 example", "simple json parse without quotes", "simplejson property name without quotes",
		"simplejson datetime", "simple json example", "simplejson unity", "simplejson golang", "simplejson.load",
		"simple json java", "simplejson python tutorial", "simplejson vs json", "simplejson example .net c\u0023",
		"simplejson dumps", "simplejson download", "simplejson documentation", "simplejson.deserializeobject",
		"simple json decode", "simplejson django", "simplejson library", "simplejson python windows", "simplejson windows"}
)

func TestParseGoogle(t *testing.T) {
	suggestions, err := parseGoogle("testname", "python", "google", []byte(sampleGoogleResp))
	if err != nil {
		t.Fatalf("Error in parsing google data: %v\n", err)
	}
	fmt.Println(suggestions.Suggestions)
	for i, suggestion := range suggestions.Suggestions {
		assert.Equal(t, sampleGoogleParsed[i], suggestion, fmt.Sprintf("%s did not match %s\n", sampleGoogleParsed[i], suggestion))
	}
}

func TestParseBing(t *testing.T) {
	suggestions, err := parseBing("testname", "python", "bing", []byte(sampleBingResp))
	if err != nil {
		t.Fatalf("Error in parsing bing data: %v\n", err)
	}
	for i, suggestion := range suggestions.Suggestions {
		assert.Equal(t, sampleBingParsed[i], suggestion, fmt.Sprintf("%s did not match %s\n", sampleBingParsed[i], suggestion))
	}
}

func TestConstructQuery(t *testing.T) {
	name := "Products.CMFDynamicViewFTI.fti.DynamicViewTypeInformation.__p4a_z2utils_orig_getAvailableViewMethods"
	query := constructQuery(name)
	expected := "Products+CMFDynamicViewFTI+fti+DynamicViewTypeInformation+__p4a_z2utils_orig_getAvailableViewMethods"
	assert.Equal(t, expected, query, fmt.Sprintf("Expected %s but got %s\n", expected, query))
}
