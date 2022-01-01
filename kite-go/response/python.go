package response

// Python-related response types
const (
	PythonDocumentationType        = "python_documentation"
	PythonSignaturePatternType     = "python_signatures"
	PythonSignatureCompletionsType = "python_signature_completions"
	PythonCompletionsType          = "python_completions"
	PythonSuggestionsType          = "python_suggestions"
	PythonSuggestionType           = "python_suggestion"
	PythonSearchSuggestionType     = "python_search_suggestion"
	PythonDefinitionType           = "python_definition"
	PythonTopMembersType           = "python_top_members"
)

// PythonExample represents one snippet of code.
type PythonExample struct {
	Code   string `json:"code"`
	Source string `json:"source"`
}

// PythonSignaturePatterns is a collection of patterns.
type PythonSignaturePatterns struct {
	Type       string                    `json:"type"`
	Signatures []*PythonSignaturePattern `json:"signatures"`
}

// PythonSignaturePattern is an example of a piece of code with a particular
// invocation pattern.
type PythonSignaturePattern struct {
	Frequency float64          `json:"frequency"`
	Signature string           `json:"signature"`
	Examples  []*PythonExample `json:"examples"`
}

// PythonTypeInfo contains information about a argument type including it's
// canonical name, friendly (human-readable) name, and documentation
type PythonTypeInfo struct {
	Name         string `json:"name"`
	FriendlyName string `json:"friendly_name"`
}

// PythonSignatureToken is a component of a signature completion result.
type PythonSignatureToken struct {
	Token     string           `json:"token"`
	TokenType string           `json:"tokenType"`
	Types     []PythonTypeInfo `json:"types,omitempty"`
	DocString string           `json:"docsString,omitempty"`
	Examples  []string         `json:"examples,omitempty"`
}

// PythonSignature wraps an array of PythonSignatureToken's
type PythonSignature struct {
	Frequency float64                 `json:"frequency"`
	Signature []*PythonSignatureToken `json:"signature"`
}

// PythonSignatureCompletions are a set of argument-level signature completions
type PythonSignatureCompletions struct {
	Type        string             `json:"type"`
	Prefix      string             `json:"prefix"`
	ReturnTypes []PythonTypeInfo   `json:"returnTypes"`
	ArgIndex    int                `json:"tokenCursorIndex"`
	Completions []*PythonSignature `json:"completions"`
}

// PythonSearchSuggestion is a response containing search query suggestions.
// This struct contains PythonDocumentation and other
// relevant python responses; therefore, while the suggested search query
// are not python specific (it is only a string), we still define the struct
// here.
type PythonSearchSuggestion struct {
	Type          string               `json:"type"`
	RawQuery      string               `json:"raw_query"`
	Identifier    string               `json:"identifier"`
	Documentation *PythonDocumentation `json:"documentation"`
}

// PythonIdentifier is a response containing different string identifiers for a
// Python language entity
type PythonIdentifier struct {
	ID         string `json:"id"`
	RelativeID string `json:"relative_id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
}

// PythonDocumentation is a response containing documentation for a Python language
// entity (such as a method).
type PythonDocumentation struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Kind        string `json:"kind"`
	Signature   string `json:"signature"`
	Description string `json:"description"`
	LocalCode   bool   `json:"local_code"`

	SeeAlso  []PythonSeeAlso `json:"see_also"`
	Warnings []PythonWarning `json:"warnings"`
	Hints    []string        `json:"hints"`

	StructuredDoc *PythonStructuredDoc `json:"structured_doc"`

	Ancestors  []PythonIdentifier `json:"ancestors"`
	Children   []PythonIdentifier `json:"children"`
	References []PythonIdentifier `json:"references"`

	CuratedExamplePreviews []*CuratedExamplePreview `json:"curated_example_previews"`
}

// PythonSeeAlso contains information about a related identifier.
type PythonSeeAlso struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

// PythonWarning contains a warning for the identifier.
type PythonWarning struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

// PythonStructuredDoc is a response containing structured information for a Python language entity.
type PythonStructuredDoc struct {
	Ident       string             `json:"ident"`
	Parameters  []*PythonParameter `json:"parameters"`
	Description string             `json:"description"`
	ReturnType  string             `json:"return_type"`
}

// PythonParameter is a response containing structured information for a parameter of a Python language class, function or method.
type PythonParameter struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Default string `json:"default"`

	Description string `json:"description"`
}

// PythonCompletions is a collection of completions results.
type PythonCompletions struct {
	Type        string              `json:"type"`
	Prefix      string              `json:"prefix"`
	PrefixType  PythonTypeInfo      `json:"prefixType"`
	Completions []*PythonCompletion `json:"completions"`
}

// PythonCompletion is a response containing a single completion result.
type PythonCompletion struct {
	// Attr contains the suggested attribute with no periods
	Attr string `json:"attr"`
	// TopLevel is true if this is a root-level package name, in which case no
	// period will be shown before the attribute in the completions view.
	TopLevel       bool   `json:"toplevel"`
	Identifier     string `json:"identifier"`
	Type           string `json:"type"`
	FullIdentifier string `json:"full_identifier"`
}

// PythonSuggestions contains diff suggestions that are for the same
// error.
type PythonSuggestions struct {
	Type        string              `json:"type"`
	File        string              `json:"file"`
	Suggestions []*PythonSuggestion `json:"suggestions"`
	ID          uint64              `json:"hash_id"`
	Line        int                 `json:"line"`
	Begin       int                 `json:"begin"`
	End         int                 `json:"end"`
}

// PythonSuggestion wraps a set of diffs for fixing a potential error.
type PythonSuggestion struct {
	Type       string        `json:"type"`
	Score      float64       `json:"score"`
	PluginID   string        `json:"plugin_id"`
	FileMD5    string        `json:"file_md5"`
	FileBase64 []byte        `json:"file_base64"`
	Filename   string        `json:"filename"`
	Diffs      []*PythonDiff `json:"diffs"`
}

// PythonDiff contains two strings for making a diff, the line number
// and the token positions of the diff.
type PythonDiff struct {
	Type        string `json:"type"`
	LineNum     int    `json:"linenum"`
	Begin       int    `json:"begin"`
	End         int    `json:"end"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	LineSrc     string `json:"line_src"`
	LineDest    string `json:"line_dest"`
	ID          string `json:"hash_id"`
}

// PythonReference represents a reference to a fully-qualified name within a
// program
type PythonReference struct {
	Begin              int    `json:"begin"`
	End                int    `json:"end"`
	Original           string `json:"expression"`
	FullyQualifiedName string `json:"fully_qualified"`
	Instance           bool   `json:"instance"`
	NodeType           string `json:"node_type"` // "import", "name", "attribute", or "call"
}

// PythonTopMembers represents the top members of a python package.
type PythonTopMembers struct {
	Type    string          `json:"type"`
	Root    string          `json:"root"`
	Members []*PythonMember `json:"members"`
}

// PythonTopMembersUnified represents the top members of a python package.
type PythonTopMembersUnified struct {
	Type         string          `json:"type"`
	Symbol       string          `json:"symbol"`
	SymbolType   string          `json:"symbol_type"`
	Members      []*PythonMember `json:"members"`
	TotalMembers int             `json:"total_members"`
}

// PythonMember represents a member of a module.
type PythonMember struct {
	Attr       string `json:"attr"`
	Identifier string `json:"identifier"`
	Type       string `json:"type"`
	ID         string `json:"id"`
}

// PythonKwargs represents the possible `**kwargs` for a function.
type PythonKwargs struct {
	// Name is the name of the `**kwargs` as specified in the arg spec.
	Name string `json:"name"`
	// Kwargs is the possible `**kwargs` for a function.
	Kwargs []*PythonKwarg `json:"kwargs"`
}

// PythonKwarg reporesent a possible **kwarg for a function.
type PythonKwarg struct {
	Name  string   `json:"name"`
	Types []string `json:"types"`
}
