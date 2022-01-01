package editorapi

// Kind of the source value, all values have a `Kind`.
type Kind string

// Value represents an "object" that could be assigned to a source code variable.
// A `Value` is a function, module, class, instance, or anything else representable as an “object.”
// Some but not all values have `ID`s, but the `ID`` of a value is different to the name of the symbol that holds it.
type Value struct {
	ID     ID     `json:"id"`
	Kind   Kind   `json:"kind"`
	Repr   string `json:"repr"`
	Type   string `json:"type"`
	TypeID ID     `json:"type_id"`
}

// ValueExt represents a value with extra information (this extra info is too big for some contexts).
type ValueExt struct {
	Value
	Synopsis string `json:"synopsis"`

	// Details contains the added details about the `Value`.
	Details Details `json:"details"`

	// Ancestors of the value.
	Ancestors []Ancestor `json:"ancestors"`
}

// IDName encapsulates a symbol identifier along with a display name for that symbol.
type IDName struct {
	ID   ID     `json:"id"`
	Name string `json:"name"`
}

// SymbolBase encapsulates a symbol identifier/dispay name and those of its parent
type SymbolBase struct {
	IDName
	Parent *IDName `json:"parent"` // nil for top-level modules/packages
}

// Symbol represents a named source code variable or attribute.
// A symbol is an attribute of an object, or a local variable.
// All symbols have names, live in a namespace, and can hold values.
type Symbol struct {
	SymbolBase
	// Namespace will be nil for local variables or global modules
	Namespace *Value `json:"namespace"`
	Value     Union  `json:"value"`
}

// SymbolExt represents a symbol with extra information.
type SymbolExt struct {
	SymbolBase
	// Namespace will be nil for local variables or global modules
	Namespace *Value   `json:"namespace"`
	Value     UnionExt `json:"value"`
	Synopsis  string   `json:"synopsis"`
}

// Union represents the value of a variable that could have one of
// several possible values.
type Union []*Value

// UnionExt represents a union with extra information.
type UnionExt []*ValueExt

// Ancestor for a value.
type Ancestor struct {
	ID   ID     `json:"id"`
	Name string `json:"name"`
}

//
// Details
//

// Details wraps the possible `Details` responses for a ValueExt.
type Details struct {
	Function *FunctionDetails `json:"function"`
	Type     *TypeDetails     `json:"type"`
	Instance *InstanceDetails `json:"instance"`
	Module   *ModuleDetails   `json:"module"`
}

// FunctionDetails is attached to ValueExt.Detail when the value's kind is "function".
type FunctionDetails struct {
	Parameters      []*Parameter            `json:"parameters"`
	ReturnValue     Union                   `json:"return_value"`
	Signatures      []*Signature            `json:"signatures"`
	LanguageDetails LanguageFunctionDetails `json:"language_details"`
}

// LanguageFunctionDetails wraps the possilbe `LanugageDetails` for `FunctionDetails`.
type LanguageFunctionDetails struct {
	Python *PythonFunctionDetails `json:"python"`
}

// Parameter represents a parameter in a function definition.
type Parameter struct {
	Name            string                   `json:"name"`
	InferredValue   Union                    `json:"inferred_value"`
	Synopsis        string                   `json:"synopsis"` // currently never set
	LanguageDetails LanguageParameterDetails `json:"language_details"`
}

// LanguageParameterDetails wraps the possible `LanguageDetails` for `Parameter`.
type LanguageParameterDetails struct {
	Python *PythonParameterDetails `json:"python"`
}

// TypeDetails is attached to ValueExt.Detail when the value's kind is "type".
type TypeDetails struct {
	Components      []*Value            `json:"components"`
	Members         []*Symbol           `json:"members"`       // only includes the first N members
	TotalMembers    int                 `json:"total_members"` // may be greater than len(Members)
	LanguageDetails LanguageTypeDetails `json:"language_details"`
}

// LanguageTypeDetails wraps the possible `LanguageDetails` for `TypeDetails`.
type LanguageTypeDetails struct {
	Python *PythonTypeDetails `json:"python"`
}

// InstanceDetails is attached to ValueExt.Detail when the value's kind is "instance".
type InstanceDetails struct {
	Type Union `json:"type"`
}

// ModuleDetails is attached to ValueExt.Detail when the value is a module.
type ModuleDetails struct {
	Members      []*Symbol `json:"members"`       // only includes the first N members
	TotalMembers int       `json:"total_members"` // may be greater than len(Members)
}

//
// Signatures
//

// Signature represents a possible signature completion for a function call.
type Signature struct {
	Args            []*ParameterExample      `json:"args"`
	LanguageDetails LanguageSignatureDetails `json:"language_details"`
	Frequency       float64                  `json:"frequency"` // freq of pattern
}

// LanguageSignatureDetails wraps the possible `LanguageDetails` for `Signatures`.
type LanguageSignatureDetails struct {
	Python *PythonSignatureDetails `json:"python"`
}

// ParameterExample represents a single argument and the value(s) it could take.
type ParameterExample struct {
	// Name of the argument used in a function call.
	Name string `json:"name"`

	// Types of the argument used in a function call.
	Types []*ParameterTypeExample `json:"types"`
}

// ParameterTypeExample represents the type (value) an argument could take as well as examples of the type (value).
type ParameterTypeExample struct {
	// ID for the value of the type.
	ID ID `json:"id"`

	// Name is a human readable name for the type (value).
	Name string `json:"name"`

	// Examples are plain string examples from codebases on Github.
	Examples  []string `json:"examples"`
	Frequency float64  `json:"frequency"`
}

//
// Report
//

// Report contains all of the information that Kite knows about a value or symbol.
type Report struct {
	// Definition of the value or symbol.
	Definition *Definition `json:"definition"`

	// DescriptionText is the full documentation for the value/symbol.
	DescriptionText string `json:"description_text"`

	// DescriptionHTML is the full documenation for the value/symbol in HTML.
	DescriptionHTML string `json:"description_html"`

	// Examples of the value/symbol.
	Examples []*Example `json:"examples"`

	// Usages of the value/symbol, contains the first N.
	// TODO(naman) deprecated; rm
	Usages []struct{} `json:"usages"`

	// TotalUsages is the total number of usages of the value/symbol.
	//   See: /usages endpoint.
	// TODO(naman) deprecated; rm
	TotalUsages int `json:"total_usages"`

	// Links to webpages relevant to the value/symbol, contains the first N.
	Links []*Link `json:"links"`

	// TotalLinks is the total number of links of the value/symbol.
	//   See: /links endpoint.
	TotalLinks int `json:"total_links"`
}

// Definition contains information about where a symbol/value is defined.
type Definition struct {
	// Filename is the full native path to the file.
	Filename string `json:"filename"`
	// Line number that the definition starts on, 1 indexed.
	Line int `json:"line"`
}

// Example is a reference to a curated code example.
// See: /examples endpoint.
type Example struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// Link is a reference to an external page.
type Link struct {
	// Title is the title for the link
	Title string `json:"title"`
	// URL is the location of the page
	URL string `json:"url"`
	// Stackoverflow is non-nil only if this is a link to a stackoverflow post
	Stackoverflow *StackoverflowPost `json:"stackoverflow"`
}

// StackoverflowPost contains additional link data for stackoverflow links
type StackoverflowPost struct {
	// Score is the number of upvotes minus the number of downvotes
	Score int64 `json:"score"`
}

//
// Endpoint responses
//

// HoverResponse is the response struct for the /hover endpoint.
type HoverResponse struct {
	Language     string       `json:"language"`
	PartOfSyntax string       `json:"part_of_syntax"`
	Symbol       []*SymbolExt `json:"symbol"`
	Report       *Report      `json:"report"`
}

// AnswersLink encapsulates a link to a Kite Answers page
type AnswersLink struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

// ReportResponse is the response struct for the /report endpoint.
type ReportResponse struct {
	// LinkCanonical indicates the ID of the symbol to use for <link rel="canonical">
	// If it omitted, the client should use <meta name="robots" value="noindex, nofollow">
	LinkCanonical string `json:"link_canonical,omitempty"`

	AnswersLinks []AnswersLink `json:"answers_links"`

	Language string     `json:"language"`
	Symbol   *SymbolExt `json:"symbol,omitempty"`
	Value    *ValueExt  `json:"value,omitempty"`
	Report   *Report    `json:"report"`
}

// MembersExtResponse is the response struct for the /members endpoint when symbolexts are requested
type MembersExtResponse struct {
	Language string       `json:"language"`
	Total    int          `json:"total"`
	Start    int          `json:"start"`
	End      int          `json:"end"`
	Members  []*SymbolExt `json:"members"`
}

// MembersResponse is the response struct for the /members endpoint when symbols are requested
type MembersResponse struct {
	Language string    `json:"language"`
	Total    int       `json:"total"`
	Start    int       `json:"start"`
	End      int       `json:"end"`
	Members  []*Symbol `json:"members"`
}

// CalleeResponse is the response struct for the /callee endpoint.
type CalleeResponse struct {
	FuncName   string       `json:"func_name"`
	Language   string       `json:"language"`
	Callee     *ValueExt    `json:"callee"`
	Report     *Report      `json:"report"`
	Signatures []*Signature `json:"signatures"`
}

// SearchResult is a single search result.
type SearchResult struct {
	// Type is either "value" or "symbol"
	// Result is either a Value or Symbol object, respectively
	Type   string      `json:"type"`
	Result interface{} `json:"result"`
}

// SearchResults is the response struct for the /search endpoint.
type SearchResults struct {
	Language        string         `json:"language"`
	Total           int            `json:"total"`
	Start           int            `json:"start"`
	End             int            `json:"end"`
	Results         []SearchResult `json:"results"`
	DataUnavailable bool           `json:"data_unavailable"`
}

// LinksResponse is the response struct for the /links enpoint.
type LinksResponse struct {
	Language string `json:"language"`
	// Total number of links
	Total int `json:"total"`
	// Start index for links in the response
	Start int `json:"start"`
	// End index for links in the response
	End int `json:"end"`
	// Links in the response
	Links []*Link `json:"links"`
}

// SignatureResponse is the response struct for
// the /clientapi/editor/signatures endpoint.
// Contains all possible calls for the given buffer and cursor position.
// There may be more than one call in the case of a nested call expression,
// in this case the array of calls is sorted with the innermost call expression at the front of the array.
type SignatureResponse struct {
	Language string  `json:"language"`
	Calls    []*Call `json:"calls"`
}

// Call groups together all possible signature completions for a given function.
type Call struct {
	// Callee is the value of the function being called.
	Callee     *ValueExt    `json:"callee"`
	FuncName   string       `json:"func_name"`
	Signatures []*Signature `json:"signatures"`
	// ArgIndex is the index of the argument currently being assigned.
	ArgIndex        int                 `json:"arg_index"`
	LanguageDetails LanguageCallDetails `json:"language_details"`
}

// LanguageCallDetails wraps the possible `LanguageDetails` for a `Call`.
type LanguageCallDetails struct {
	Python *PythonCallDetails `json:"python"`
}
