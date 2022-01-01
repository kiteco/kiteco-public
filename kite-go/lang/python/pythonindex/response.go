package pythonindex

import (
	"strings"

	"github.com/kiteco/kiteco/kite-go/response"
)

const (
	builtin = "builtins."
)

// --

// SearchSuggestionResponse converts a QueryCompletionResult object to a response type
// that is sent to the front end.
func SearchSuggestionResponse(result *QueryCompletionResult) *response.PythonSearchSuggestion {
	return &response.PythonSearchSuggestion{
		Type:       response.PythonSearchSuggestionType,
		Identifier: strings.TrimPrefix(result.Display, builtin),
		RawQuery:   result.Ident,
	}
}
