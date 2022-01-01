package pythonproviders

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
)

func checkNoCompletionWithSuffix(t *testing.T, completions provisionResult, forbiddenSuffix string) {
	comp := getCompletionWithSuffix(t, completions, forbiddenSuffix)
	var compText string
	if len(comp) > 0 {
		compText = comp[0]
	}
	assert.Empty(t, comp, "The completions %s ends with %s", compText, forbiddenSuffix)
}

func getCompletionWithSuffix(t *testing.T, completions provisionResult, suffix string) []string {
	var result []string
	for _, cList := range completions.out {
		for _, c := range cList {
			if strings.HasSuffix(c.Snippet.Text, suffix) {
				result = append(result, c.Snippet.Text)
			}
		}
	}
	return result
}

func checkNoCompletionWithPrefix(t *testing.T, completions provisionResult, forbiddenPrefix string) {
	comp := getCompletionWithPrefix(t, completions, forbiddenPrefix)
	var compText string
	if len(comp) > 0 {
		compText = comp[0]
	}
	assert.Empty(t, comp, "The completions ", compText, " starts with ", forbiddenPrefix)
}

func getCompletionWithPrefix(t *testing.T, completions provisionResult, prefix string) []string {
	var result []string
	for _, cList := range completions.out {
		for _, c := range cList {
			if strings.HasPrefix(c.Snippet.Text, prefix) {
				result = append(result, c.Snippet.Text)
			}
		}
	}
	return result
}

func getCompletionContaining(completions provisionResult, substr string) []MetaCompletion {
	var result []MetaCompletion
	for _, cList := range completions.out {
		for _, c := range cList {
			if strings.Contains(c.Snippet.Text, substr) {
				result = append(result, c)
			}
		}
	}
	return result
}

func mustContainsCompletion(t *testing.T, snippet string, completions provisionResult) {
	comps := getCompletionContaining(completions, snippet)
	var found bool
	for _, c := range comps {
		if c.Snippet.Text == snippet {
			found = true
			break
		}
	}
	assert.True(t, found, "The completion %s is missing from the completion result", snippet)
}

func mustContainsCompletionWithPlaceholders(t *testing.T, snippet string, placeholders []data.Selection, completions provisionResult) {
	comps := getCompletionContaining(completions, snippet)
	var found bool
	for _, c := range comps {
		if c.Snippet.Text == snippet {
			if containsPlaceholders(placeholders, c.Snippet.Placeholders()) {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "The completion %s is missing from the completion result", snippet)
}

func mustContainsCompletionWithoutPlaceholders(t *testing.T, snippet string, placeholders []data.Selection, completions provisionResult) {
	comps := getCompletionContaining(completions, snippet)
	var found bool
	for _, c := range comps {
		if c.Snippet.Text == snippet {
			if !containsPlaceholders(placeholders, c.Snippet.Placeholders()) {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "The completion %s is missing from the completion result", snippet)
}

func containsPlaceholders(targets, list []data.Selection) bool {
	for _, p := range targets {
		if !containsPlaceholder(p, list) {
			return false
		}
	}
	return true
}

func containsPlaceholder(p data.Selection, ps []data.Selection) bool {
	for _, pp := range ps {
		if pp.Begin == p.Begin && pp.End == p.End {
			return true
		}
	}

	return false
}

func runProviderWithPartialDecoder(t *testing.T, p Provider, template string) (provisionResult, error) {
	return runProviderWithOpts(t, p, template, true)
}

func TestGGNNPartial(t *testing.T) {
	SetUseGGNNCompletions(true)

	src := `
import requests
my_url = "www.google.fr"
resp = requests.get($
`
	completions, err := runProviderWithPartialDecoder(t, GGNNModelAccumulator{}, src)
	assert.NotNil(t, completions)
	require.NoError(t, err)
	checkNoCompletionWithSuffix(t, completions, ",)")

}

func TestGGNNPartialWithClosingParenthesis(t *testing.T) {
	SetUseGGNNCompletions(true)

	src := `
import requests
my_url = "www.google.fr"
resp = requests.get($)
`
	completions, err := runProviderWithPartialDecoder(t, GGNNModelAccumulator{}, src)
	require.NoError(t, err)
	checkNoCompletionWithSuffix(t, completions, ")")
}

func TestGGNNPartialWithOneArgAndComma(t *testing.T) {
	SetUseGGNNCompletions(true)

	src := `
import requests
my_url = "www.google.fr"
data = {}
resp = requests.get(my_url,$)
`
	completions, err := runProviderWithPartialDecoder(t, GGNNModelAccumulator{}, src)
	require.NoError(t, err)
	checkNoCompletionWithSuffix(t, completions, ")")
	checkNoCompletionWithPrefix(t, completions, ",")
}

func TestGGNNPartialAttributeCompletion(t *testing.T) {
	SetUseGGNNCompletions(true)

	src := `
import requests
my_url = "www.google.fr"
data = {}
resp = requests.ge$
`
	completions, err := runProviderWithPartialDecoder(t, GGNNModelAccumulator{}, src)
	require.NoError(t, err)
	assert.NotEmpty(t, completions.out, "There should be attribute completions from the GGNN model")
}

func TestGGNNFullAndPartialCompletions(t *testing.T) {
	SetUseGGNNCompletions(true)

	src := `
import requests
my_url = "www.google.fr"
data = {}
resp = requests.get($
`
	completions, err := runProviderWithPartialDecoder(t, GGNNModelAccumulator{}, src)
	require.NoError(t, err)
	mustContainsCompletionWithPlaceholders(t, "my_url)", []data.Selection{{Begin: 6, End: 6}}, completions)
	// Full call completions are no more generated as their buffer looks like a duplicate of the corresponding partial call
	//mustContainsCompletionWithoutPlaceholders(t, "my_url)", []data.Selection{{6, 6}}, completions)

}

func TestGGNNSecondArgAfterComma(t *testing.T) {
	SetUseGGNNCompletions(true)

	src := `
import requests
my_url = "www.google.fr"
data = {}
resp = requests.post(my_url,$
`
	completions, err := runProviderWithPartialDecoder(t, GGNNModelAccumulator{}, src)
	require.NoError(t, err)
	require.NotEmptyf(t, completions.out, "Completion should be generated for the state `requests.get(my_url,$")
}

func TestGGNNNotApplicableOnNameExpr(t *testing.T) {
	SetUseGGNNCompletions(true)

	src := `
import requests
my_url = "www.google.fr"
data = {}
resp$
`
	_, err := runProviderWithPartialDecoder(t, GGNNModelAccumulator{}, src)
	require.Error(t, err, data.ProviderNotApplicableError{})
}
