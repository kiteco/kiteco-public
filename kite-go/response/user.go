package response

import "github.com/kiteco/kiteco/kite-go/localcode"

// Root is the top level structure of responses that are sent back to the client.
type Root struct {
	ID          int64         `json:"id,omitempty"`
	Hash        string        `json:"hash,omitempty"`
	Cursor      int64         `json:"cursor,omitempty"`
	Latency     int64         `json:"latency,omitempty"`
	Type        string        `json:"type"`
	Query       string        `json:"query,omitempty"`
	Description string        `json:"description,omitempty"`
	Errors      []ErrorInfo   `json:"errors,omitempty"`
	Results     []interface{} `json:"results"`
	Filename    string        `json:"filename"`
	Editor      string        `json:"editor"`
	ResendText  bool          `json:"resendtext"`
	Diagnostics string        `json:"diagnostics"`
	State       string        `json:"state"`

	LocalIndexPresent bool                      `json:"local_index_present"`
	LocalIndexStatus  *localcode.StatusResponse `json:"local_index_status"`

	ExpectCompletions bool `json:"expect_completions"`

	// EditorCompletions contains the completions that should be shown in the editor
	// context menu. This is separate from the language-specific completions that may
	// be stored within results, since those completions can contain language-specific
	// information (e.g. as identifiers for interlinking).
	EditorCompletions         *EditorCompletions   `json:"editor_completions"`
	PrefetchedCompletionsList []*EditorCompletions `json:"prefetched_completions_list"`
}

// RouterRoot is the subset of Root that is accessible from kited. Most of the struct
// is simply forwarded on to the js client, but some fields are inspected en route.
type RouterRoot struct {
	ID                        int64                `json:"id,omitempty"`
	Hash                      string               `json:"hash,omitempty"`
	Cursor                    int64                `json:"cursor,omitempty"`
	Latency                   int64                `json:"latency,omitempty"`
	Type                      string               `json:"type"`
	State                     string               `json:"state"`
	EditorCompletions         *EditorCompletions   `json:"editor_completions"`
	PrefetchedCompletionsList []*EditorCompletions `json:"prefetched_completions_list"`
}

// ManDocumentation is a response containing a man document.
type ManDocumentation struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// JournalReference is a response containing the current journal user ID and time.
type JournalReference struct {
	Type      string `json:"type"`
	UserID    int64  `json:"userid"`
	Timestamp int64  `json:"timestamp"`
}

// ErrorInfo represents one error message that was extracted from the terminal
// output. For example:
//
// {
//   DetectedError: `mysource.go:50: undefined "foobar"`,
//   Line: 50,
//   Filename: "mysource.go",
//   Template: `undefined "%s"`,
//   Wildcards: []string{"foobar"},
//   CommunityError: CommunityError{ ... }
// }
//
type ErrorInfo struct {
	DetectedError string   `json:"detectedError"`
	Line          int      `json:"line"`
	Filename      string   `json:"filename"`
	Template      string   `json:"template"`
	TemplateID    int      `json:"templateID"`
	TemplateLang  string   `json:"templateLang"`
	Wildcards     []string `json:"wildcards"`

	Clusters []EpisodeCluster `json:"episodeClusters"`
}

// EpisodeCluster represents an episode cluster for an error message.
type EpisodeCluster struct {
	Frequency float64   `json:"frequency"`
	Episodes  []Episode `json:"episodes"`
}

// Episode is a response for a diff of code as an error fix/a semantic
// programming block.
type Episode struct {
	Type   string     `json:"type"`
	Source string     `json:"source"`
	Code   []FileDiff `json:"code"`
}

// FileDiff represents the changes made to a file, as part of an Episode.
type FileDiff struct {
	Filename string `json:"filename"`
	Code1    string `json:"code1"`
	Code2    string `json:"code2"`
}

// Lockout is the result type returned when Kite has not been enabled on
// the current file.
type Lockout struct {
	Type       string `json:"type"`
	Filename   string `json:"filename"`
	ProjectDir string `json:"projectDir"`
}

// LocalIndexPresent contains whether a local index was present during processing
type LocalIndexPresent struct {
	Present bool `json:"present"`
}

// ExpectCompletions stores whether or not we expect completions
type ExpectCompletions struct {
	ExpectCompletions bool `json:"expect_completions"`
}

// EditorCompletions contains completions to be shown in editors (e.g. the completions
// dropdown in sublime)
type EditorCompletions struct {
	Hash        string             `json:"hash"`   // non-cryptographic hash of file contents
	Cursor      int64              `json:"cursor"` // cursor offset in bytes
	Completions []EditorCompletion `json:"completions"`
	Attr        string             `json:"attr"`
}

// EditorCompletionSource represents the "source" of a completion (i.e. how the completion was generated)
type EditorCompletionSource string

const (
	// UnknownCompletionSource is the zero EditorCompletionSource value
	UnknownCompletionSource EditorCompletionSource = ""
	// TraditionalCompletionSource is for completions from traditional program analysis, potentially ordered with basic statistical data
	TraditionalCompletionSource EditorCompletionSource = "traditional"
	// KeywordModelCompletionSource is for completions from the keyword completions model
	KeywordModelCompletionSource EditorCompletionSource = "keyword_model"
	// CallModelCompletionSource is for completions from the call completions model
	// we use this for both the old call completion model and the new call completion
	// model that is powered by the infer_expr model
	CallModelCompletionSource EditorCompletionSource = "call_model"
	// AttributeModelCompletionSource is for completions from the attribute completions model
	// we use this for both the old attr completion model and the new attr completion
	// model that is powered by the infer_expr model
	AttributeModelCompletionSource EditorCompletionSource = "attribute_model"
	// ExprModelCompletionsSource is for completions from the infer_expr model
	ExprModelCompletionsSource EditorCompletionSource = "expr_model"
	// GlobalPopularPatternCompletionSource is for completion from popular patterns for global functions
	GlobalPopularPatternCompletionSource EditorCompletionSource = "global_popular_pattern"
	// LocalPopularPatternCompletionSource is for completions from popular patterns for local functions
	LocalPopularPatternCompletionSource EditorCompletionSource = "local_popular_pattern"
	// ArgSpecCompletionSource is for completion from argspec based completions
	ArgSpecCompletionSource EditorCompletionSource = "argspec"
	// DictCompletionSource if for completion related to dictionary
	DictCompletionSource EditorCompletionSource = "dict"
	// EmptyCallCompletionSource is for completion from the empty call provider (adding parenthesis after a function name)
	EmptyCallCompletionSource EditorCompletionSource = "empty_call"
	// EmptyAttrCompletionSource is for completion from empty_attr provider
	EmptyAttrCompletionSource EditorCompletionSource = "empty_attr"
	// GGNNModelPartialCallSource is for partial call completion from GGNNModel provider
	GGNNModelPartialCallSource EditorCompletionSource = "ggnn_model_partial_call"
	// GGNNModelFullCallSource is for full call completion from GGNNModel provider
	GGNNModelFullCallSource EditorCompletionSource = "ggnn_model_full_call"
	// GGNNModelAttributeSource is for attribute completion from GGNNModel provider
	GGNNModelAttributeSource EditorCompletionSource = "ggnn_model_attribute"
	// LexicalPythonSource is for completions from the lexical python provider
	LexicalPythonSource EditorCompletionSource = "lexical_python"
)

// EditorCompletion represents one possible completion to be shown in an editor.
// The Display field is what to display to the user and the Insert field is what
// to actually insert when the user selects that option. The hint is further
// information about the completion, such as its type.
type EditorCompletion struct {
	Display           string `json:"display"`
	Insert            string `json:"insert"`
	Hint              string `json:"hint"`
	DocumentationText string `json:"documentation_text"`
	DocumentationHTML string `json:"documentation_html"`
	// Symbol contains an import graph representation of the completion. Within
	// Python, it's set to the pythonresponse.Symbol type.
	Symbol interface{} `json:"symbol"`
	// Source encapsulates where the completion was generated; it is not serialized to disallow use by the editor (or kited if using Kite Cloud)
	Source EditorCompletionSource `json:"-"`
	// Smart indicates whether the completion should be considered "smart" for the purposes of editor functionality
	Smart bool `json:"smart"`
}

// SearchResults contains the results of an active search request. The contents
// of this response may include search results for different languages, though
// currently only search results from Python are stored.
type SearchResults struct {
	PythonResults interface{} `json:"python_results"`
}
