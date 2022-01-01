package response

// SandboxCompletions contains completions to be shown in the web sandbox
type SandboxCompletions struct {
	Hash        string              `json:"hash"`   // non-cryptographic hash of file contents
	Cursor      int64               `json:"cursor"` // cursor offset in bytes
	Completions []SandboxCompletion `json:"completions"`
	Attr        string              `json:"attr"`
}

// SandboxCompletion represents one possible completion to be shown in an editor.
// The Display field is what to display to the user and the Insert field is what
// to actually insert when the user selects that option. The hint is further
// information about the completion, such as its type.
// It takes out the DocumentationText and DocumentationHTML fields to reduce payload
// size over the wire
type SandboxCompletion struct {
	Display string `json:"display"`
	Insert  string `json:"insert"`
	Hint    string `json:"hint"`
	// Symbol contains an import graph representation of the completion. Within
	// Python, it's set to the pythonresponse.Symbol type.
	Symbol interface{} `json:"symbol"`
}

// XXXXXXXCompletions contains completions for XXXXXXX
type XXXXXXXCompletions struct {
	Completions []XXXXXXXCompletion `json:"completions"`
	StartCol    int                  `json:"startCol"`
}

// XXXXXXXCompletion ...
type XXXXXXXCompletion struct {
	// Text is a unified display/insert field for now
	Text string `json:"text"`
	Info string `json:"info"`

	// Display is equal to Text for now, but may become distinct
	Display     string `json:"-"`
	Hint        string `json:"-"`
	WebDocsLink string `json:"-"`
	Synopsis    string `json:"-"`
	Signature   string `json:"-"`

	/*
		Display     string `json:"kite_display"`
		Hint        string `json:"kite_hint"`
		WebDocsLink string `json:"kite_web_docs_link"`
		Synopsis    string `json:"kite_synopsis"`
		Signature   string `json:"kite_signature"`
	*/
}
