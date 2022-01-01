package response

// JSDocumentation is a response containing documentation for a JavaScript
// language entity (such as a method).
type JSDocumentation struct {
	Type        string `json:"type"`
	Signature   string `json:"signature"`
	Description string `json:"description"`
}

// JSCompletions is a response containing possible completions to an input
// string, each completion corresponding to a JavaScript language entity.
type JSCompletions struct {
	Type        string   `json:"type"`
	Prefix      string   `json:"prefix"`
	Completions []string `json:"completions"`
}

// JSExample is a response containing an example use of a JavaScript
// language entity.
type JSExample struct {
	Code      string `json:"code"`
	From      string `json:"from"`
	ExampleOf string `json:"exampleof"`
}

// JSExamples is a response containing multiple `JSExample`s.
type JSExamples struct {
	Type     string       `json:"type"`
	Examples []*JSExample `json:"examples"`
}
