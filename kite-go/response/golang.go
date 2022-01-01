package response

// GoDocumentation is a response containing documentation for a Go language
// entity (such as a method).
type GoDocumentation struct {
	Type        string `json:"type"`
	Signature   string `json:"signature"`
	Description string `json:"description"`
}

// GoCompletions is a response containing possible completions to an input
// string, each completion corresponding to a Go language entity.
type GoCompletions struct {
	Type        string   `json:"type"`
	Prefix      string   `json:"prefix"`
	Completions []string `json:"completions"`
}

// GoExample is a response containing an example use of a Go language entity.
type GoExample struct {
	Code      string `json:"code"`
	From      string `json:"from"`
	ExampleOf string `json:"exampleof"`
	DebugInfo string `json:"debuginfo"` // Information about why this example was selected
}

// GoExamples is a response containing multiple `GoExample`s.
type GoExamples struct {
	Type     string       `json:"type"`
	Examples []*GoExample `json:"examples"`
}
