package pythonindex

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/stretchr/testify/assert"
)

func TestSearchSuggestionResponse(t *testing.T) {
	query := "csv.reader"
	act := SearchSuggestionResponse(&QueryCompletionResult{query, query})

	assert.Equal(t, response.PythonSearchSuggestionType, act.Type)
	assert.Equal(t, query, act.Identifier)
	assert.Equal(t, query, act.RawQuery)
}
