package response

// Response types
const (
	BashCommandType     = "bash_command"
	BashExamplesType    = "bash_examples"    // Type for BashExamples
	BashCompletionsType = "bash_completions" // Type for BashCompletions
)

// BashCommand is the passive search response for terminal interactions.
type BashCommand struct {
	// Type must be BashCommandType
	Type string `json:"type"`

	// These fields are for the interactive/typeahead portion of the sidebar.
	Input       TerminalInput `json:"input"`   // Echo what the user is typing
	CurrentArgs []BashOption  `json:"options"` // List of option completions
	// List of synopses, like "mv [options] source target"
	Synopses []string `json:"synopses"`

	// Following fields will be used soon

	// Docs
	DocHeader string `json:"doc_header"` // like "cp - copy files"
	DocText   string `json:"doc_text"`   // one-paragraph summary of the command

	// Examples
	Examples []BashExample `json:"examples"`

	// Bash history completions
	History []string `json:"history"`
}

// TerminalInput is used to echo what the user is typing in the sidebar,
// optionally highlighting a region (the current token, for instance).
type TerminalInput struct {
	Input          string `json:"input"` // text the user has typed at prompt
	HighlightBegin int    `json:"highlightBegin"`
	HighlightEnd   int    `json:"highlightEnd"`
}

// BashOption represents the parsed manpage information about a command-line option.
type BashOption struct {
	// The flag as labelled in the manpage, like "-s" or "--insecure"
	// TODO this field should be renamed, and this entire struct should be
	// revised after we do more man-page-parsing work.
	Label string `json:"label"`
	// A one-line description of what the option does.
	ShortDescription string `json:"shortDescription"`
	// A paragraph about what the option does.
	Description string `json:"description"`
}

// BashExample is a single curated bash example
type BashExample struct {
	Command  string `json:"command"`
	Title    string `json:"title"`
	Example  string `json:"example"`
	ImageSrc string `json:"imageSrc"`
}

// BashExamples is a collection of related curated bash examples
type BashExamples struct {
	Type     string        `json:"type"`
	Key      string        `json:"key"`
	Prefix   string        `json:"prefix"`
	Examples []BashExample `json:"examples"`
}

// BashCompletions is a collection of completions results.
type BashCompletions struct {
	Type        string            `json:"type"`
	Prefix      string            `json:"prefix"`
	Completions []*BashCompletion `json:"completions"`
}

// BashCompletion is a response containing a single completion result.
type BashCompletion struct {
	Command string `json:"command"`
}
