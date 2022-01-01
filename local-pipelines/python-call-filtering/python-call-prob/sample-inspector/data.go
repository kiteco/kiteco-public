package main

import "github.com/kiteco/kiteco/kite-go/lang/python/pythongraph"

// CompletionInfo ...
type CompletionInfo struct {
	Completion string
	Features   []pythongraph.NameAndWeight
	Label      int
}

// SampleInfo ...
type SampleInfo struct {
	Source      string
	UserTyped   string
	Completions []CompletionInfo
	Truncated   bool
}

// RootData -> all the struct used to render the template, need to be exported to be read by the template engine
type RootData struct {
	Samples []SampleInfo
}
