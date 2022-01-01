package completions

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/lexicalproviders"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/predict"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Basics(t *testing.T) {
	maxStats = 2
	m := NewMetrics().Get(lang.Python)
	assert.EqualValues(t, 0, m.read(false).Triggered)
	assert.EqualValues(t, 0, m.read(false).Shown)
	assert.EqualValues(t, 0, m.read(false).AtLeastOneShown)
	assert.EqualValues(t, 0, m.read(false).AtLeastOneNewShown)
	assert.EqualValues(t, 0, m.read(false).NumSelected)
	assert.EqualValues(t, RequestCounts{}, m.read(false).Requested)
	assert.EqualValues(t, RequestCounts{}, m.read(false).RequestedRaw)
	assert.EqualValues(t, SourceBreakdown{}, m.read(false).ShownBySource)
	assert.EqualValues(t, SourceBreakdown{}, m.read(false).AtLeastOneShownBySource)
	assert.EqualValues(t, SourceBreakdown{}, m.read(false).AtLeastOneNewShownBySource)

	stats, err := m.readCompletionStats()
	require.NoError(t, err)
	assert.Len(t, stats, 0)

	m.Requested()
	m.Expected(true)
	m.ReturnedCompat("abc", 3, []response.EditorCompletion{
		{Insert: "abcd", Source: response.TraditionalCompletionSource},
		{Insert: "abcde", Source: response.TraditionalCompletionSource},
		{Insert: "abcdef", Source: response.AttributeModelCompletionSource}},
		time.Now().Add(-51*time.Millisecond))

	assert.Len(t, m.s.lastSeen, 3)
	assert.EqualValues(t, 1, m.read(false).Triggered)
	assert.EqualValues(t, 3, m.read(false).Shown)
	assert.EqualValues(t, 1, m.read(false).AtLeastOneShown)
	assert.EqualValues(t, 1, m.read(false).AtLeastOneNewShown)
	assert.EqualValues(t, 0, m.read(false).NumSelected)
	assert.EqualValues(t, 1, m.read(false).DurationHistogram[50])
	assert.EqualValues(t, RequestCounts{Total: 1, Expected: 1}, m.read(false).Requested)
	assert.EqualValues(t, RequestCounts{Total: 1, Expected: 1}, m.read(false).RequestedRaw)
	assert.EqualValues(t,
		SourceBreakdown{response.TraditionalCompletionSource: 2, response.AttributeModelCompletionSource: 1},
		m.read(false).ShownBySource)
	assert.EqualValues(t,
		SourceBreakdown{response.TraditionalCompletionSource: 1, response.AttributeModelCompletionSource: 1},
		m.read(false).AtLeastOneShownBySource)
	assert.EqualValues(t,
		SourceBreakdown{response.TraditionalCompletionSource: 1, response.AttributeModelCompletionSource: 1},
		m.read(false).AtLeastOneNewShownBySource)
	assert.EqualValues(t, 0, m.read(false).ReportedCounts.TotalSelected)
	assert.EqualValues(t, 0, m.read(false).ReportedCounts.TotalWordsInserted)
	assert.EqualValues(t, 0, m.read(false).ReportedCounts.SelectedPerEditor[data.VSCodeEditor])

	stats, err = m.readCompletionStats()
	require.NoError(t, err)
	assert.Len(t, stats, 0)

	cs := CompletionSelectedEvent{
		Editor:   data.VSCodeEditor,
		Language: lang.Python.Name(),
		Completion: data.RCompletion{
			Completion: data.Completion{Snippet: data.Snippet{Text: "abcd"}},
		},
	}
	m.CompletionSelected(cs)

	m.BufferEdited("abcd", 4)
	m.Requested()
	m.Expected(true)
	m.ReturnedCompat("abcd", 4, []response.EditorCompletion{
		{Insert: "abcde", Source: response.TraditionalCompletionSource},
		{Insert: "abcdef", Source: response.AttributeModelCompletionSource},
		{Insert: "abcdefg", Source: response.TraditionalCompletionSource}},
		time.Now().Add(-61*time.Millisecond))

	assert.Len(t, m.s.lastSeen, 3)
	assert.EqualValues(t, 2, m.read(false).Triggered)
	assert.EqualValues(t, 4, m.read(false).Shown)
	assert.EqualValues(t, 2, m.read(false).AtLeastOneShown)
	assert.EqualValues(t, 2, m.read(false).AtLeastOneNewShown)
	assert.EqualValues(t, 1, m.read(false).NumSelected)
	assert.EqualValues(t, 1, m.read(false).DurationHistogram[60])
	assert.EqualValues(t, RequestCounts{Total: 2, Expected: 2}, m.read(false).Requested)
	assert.EqualValues(t, RequestCounts{Total: 2, Expected: 2}, m.read(false).RequestedRaw)
	assert.EqualValues(t,
		SourceBreakdown{response.TraditionalCompletionSource: 3, response.AttributeModelCompletionSource: 1},
		m.read(false).ShownBySource)
	assert.EqualValues(t,
		SourceBreakdown{response.TraditionalCompletionSource: 2, response.AttributeModelCompletionSource: 2},
		m.read(false).AtLeastOneShownBySource)
	assert.EqualValues(t,
		SourceBreakdown{response.TraditionalCompletionSource: 2, response.AttributeModelCompletionSource: 1},
		m.read(false).AtLeastOneNewShownBySource)
	assert.EqualValues(t, 1, m.read(false).ReportedCounts.TotalSelected)
	assert.EqualValues(t, 1, m.read(false).ReportedCounts.TotalWordsInserted)
	assert.EqualValues(t, 1, m.read(false).ReportedCounts.SelectedPerEditor[data.VSCodeEditor])

	stats, err = m.readCompletionStats()
	require.NoError(t, err)
	assert.Len(t, stats, 1)

	m.BufferEdited("abcde", 5)
	m.Requested()
	m.Expected(false)
	m.ReturnedCompat("abcde", 5, []response.EditorCompletion{
		{Insert: "abcdef", Source: response.AttributeModelCompletionSource},
		{Insert: "abcdefg", Source: response.TraditionalCompletionSource}},
		time.Now().Add(-200*time.Millisecond))

	assert.Len(t, m.s.lastSeen, 2)
	assert.EqualValues(t, 3, m.read(false).Triggered)
	assert.EqualValues(t, 4, m.read(false).Shown)
	assert.EqualValues(t, 3, m.read(false).AtLeastOneShown)
	assert.EqualValues(t, 2, m.read(false).AtLeastOneNewShown)
	assert.EqualValues(t, 2, m.read(false).NumSelected)
	assert.EqualValues(t, 1, m.read(false).DurationHistogram[150])
	assert.EqualValues(t, 0, m.read(false).DurationHistogram[200])
	assert.EqualValues(t, RequestCounts{Total: 3, Expected: 2, Unexpected: 1}, m.read(false).Requested)
	assert.EqualValues(t, RequestCounts{Total: 3, Expected: 2, Unexpected: 1}, m.read(false).RequestedRaw)
	assert.EqualValues(t,
		SourceBreakdown{response.TraditionalCompletionSource: 3, response.AttributeModelCompletionSource: 1},
		m.read(false).ShownBySource)
	assert.EqualValues(t,
		SourceBreakdown{response.TraditionalCompletionSource: 3, response.AttributeModelCompletionSource: 3},
		m.read(false).AtLeastOneShownBySource)
	assert.EqualValues(t,
		SourceBreakdown{response.TraditionalCompletionSource: 2, response.AttributeModelCompletionSource: 1},
		m.read(false).AtLeastOneNewShownBySource)
	assert.EqualValues(t, 1, m.read(false).ReportedCounts.TotalSelected)
	assert.EqualValues(t, 1, m.read(false).ReportedCounts.TotalWordsInserted)
	assert.EqualValues(t, 1, m.read(false).ReportedCounts.SelectedPerEditor[data.VSCodeEditor])

	stats, err = m.readCompletionStats()
	require.NoError(t, err)
	assert.Len(t, stats, 2)

	m.read(true)
	assert.EqualValues(t, 0, m.read(false).Triggered)
	assert.EqualValues(t, 0, m.read(false).Shown)
	assert.EqualValues(t, 0, m.read(false).AtLeastOneShown)
	assert.EqualValues(t, 0, m.read(false).AtLeastOneNewShown)
	assert.EqualValues(t, 0, m.read(false).NumSelected)
	assert.EqualValues(t, 0, len(m.read(false).DurationHistogram))
	assert.EqualValues(t, RequestCounts{}, m.read(false).Requested)
	assert.EqualValues(t, RequestCounts{}, m.read(false).RequestedRaw)
	assert.EqualValues(t, SourceBreakdown{}, m.read(false).ShownBySource)
	assert.EqualValues(t, SourceBreakdown{}, m.read(false).AtLeastOneShownBySource)
	assert.EqualValues(t, SourceBreakdown{}, m.read(false).AtLeastOneNewShownBySource)
	assert.Empty(t, m.s.shown)
	assert.EqualValues(t, 0, m.read(false).ReportedCounts.TotalSelected)
	assert.EqualValues(t, 0, m.read(false).ReportedCounts.TotalWordsInserted)
	assert.EqualValues(t, 0, m.read(false).ReportedCounts.SelectedPerEditor[data.VSCodeEditor])

	stats, err = m.readCompletionStats()
	require.NoError(t, err)
	assert.Len(t, stats, 2) // not reset by read

	m.BufferEdited("abcdef", 6)
	m.Requested()
	m.Expected(false)
	m.ReturnedCompat("abcdef", 6, []response.EditorCompletion{
		{Insert: "abcdefg", Source: response.TraditionalCompletionSource}},
		time.Now())

	stats, err = m.readCompletionStats()
	require.NoError(t, err)
	assert.Len(t, stats, 0) // reset by hitting maxStats == 2
}

func TestMetrics_GetEngineMetricsCallback(t *testing.T) {
	m := NewMetrics().Get(lang.Python)

	acceptedProviders := make(map[data.ProviderName]struct{})
	acceptedProviders[data.PythonLexicalProvider] = struct{}{}
	acceptedProviders[data.PythonImportsProvider] = struct{}{}

	fulfilledProviders := make(map[data.ProviderName]struct{})
	fulfilledProviders[data.PythonLexicalProvider] = struct{}{}

	oldTokenCallback := m.GetEngineMetricsCallback(false)
	oldTokenCallback(acceptedProviders, fulfilledProviders)

	assert.EqualValues(t, 1, m.read(false).EngineMetrics.RequestsAccepted)
	assert.EqualValues(t, 0, m.read(false).EngineMetrics.RequestsAcceptedNewToken)
	assert.EqualValues(t, 1, m.read(false).EngineMetrics.RequestsFulfilled)
	assert.EqualValues(t, 0, m.read(false).EngineMetrics.RequestsFulfilledNewToken)

	assert.EqualValues(t, 1, m.read(false).EngineMetrics.AcceptedPerProvider[data.PythonLexicalProvider])
	assert.EqualValues(t, 1, m.read(false).EngineMetrics.AcceptedPerProvider[data.PythonImportsProvider])
	assert.EqualValues(t, 1, m.read(false).EngineMetrics.FulfilledPerProvider[data.PythonLexicalProvider])

	newTokenCallback := m.GetEngineMetricsCallback(true)
	newTokenCallback(acceptedProviders, fulfilledProviders)

	assert.EqualValues(t, 2, m.read(false).EngineMetrics.RequestsAccepted)
	assert.EqualValues(t, 1, m.read(false).EngineMetrics.RequestsAcceptedNewToken)
	assert.EqualValues(t, 2, m.read(false).EngineMetrics.RequestsFulfilled)
	assert.EqualValues(t, 1, m.read(false).EngineMetrics.RequestsFulfilledNewToken)

	assert.EqualValues(t, 2, m.read(false).EngineMetrics.AcceptedPerProvider[data.PythonLexicalProvider])
	assert.EqualValues(t, 2, m.read(false).EngineMetrics.AcceptedPerProvider[data.PythonImportsProvider])
	assert.EqualValues(t, 1, m.read(false).EngineMetrics.AcceptedPerProviderNewToken[data.PythonLexicalProvider])
	assert.EqualValues(t, 1, m.read(false).EngineMetrics.AcceptedPerProviderNewToken[data.PythonImportsProvider])
	assert.EqualValues(t, 2, m.read(false).EngineMetrics.FulfilledPerProvider[data.PythonLexicalProvider])
	assert.EqualValues(t, 1, m.read(false).EngineMetrics.FulfilledPerProviderNewToken[data.PythonLexicalProvider])
}

func TestUsedMetric_Simple(t *testing.T) {
	m := NewMetrics().Get(lang.Python)

	m.BufferEdited("json.", 5)
	assert.Empty(t, m.read(false).SelectedBySource)

	m.ReturnedCompat("json.", 5, []response.EditorCompletion{{Insert: "foo"}, {Insert: "dump"}, {Insert: "bar"}}, time.Now())
	assert.Equal(t, 1, m.read(false).Triggered)

	m.BufferEdited("json.d", 6)
	assert.Empty(t, m.read(false).SelectedBySource)

	m.ReturnedCompat("json.d", 6, []response.EditorCompletion{{Insert: "foo"}, {Insert: "dump"}, {Insert: "bar"}}, time.Now())
	assert.Equal(t, 2, m.read(false).Triggered)

	m.BufferEdited("json.dump", 9)
	assert.NotEmpty(t, m.read(false).SelectedBySource)
}

func TestUsedMetric_Nested(t *testing.T) {
	m := NewMetrics().Get(lang.Python)

	m.BufferEdited("f(django.)", 9)
	assert.Empty(t, m.read(false).SelectedBySource)

	m.ReturnedCompat("f(django.)", 9, []response.EditorCompletion{{Insert: "db"}, {Insert: "models"}, {Insert: "forms"}}, time.Now())

	m.BufferEdited("f(django.db)", 11)
	assert.NotEmpty(t, m.read(false).SelectedBySource)
}

func TestUsedMetric_Placeholders(t *testing.T) {
	ms := NewMetrics()
	m := ms.Get(lang.Python)

	m.BufferEdited("json.", 5)
	assert.Empty(t, m.read(false).SelectedBySource)

	compls := []data.NRCompletion{
		{
			RCompletion: data.RCompletion{
				Completion: data.Completion{
					Snippet: data.BuildSnippet(fmt.Sprintf("foo(%s)", data.HolePH("x"))),
					Replace: data.Selection{Begin: 5, End: 5},
				},
			},
		},
		{
			RCompletion: data.RCompletion{
				Completion: data.Completion{
					Snippet: data.BuildSnippet(fmt.Sprintf("dump(%s, %s)", data.HolePH("obj"), data.HolePH("fp"))),
					Replace: data.Selection{Begin: 5, End: 5},
				},
			},
		},
		{
			RCompletion: data.RCompletion{
				Completion: data.Completion{
					Snippet: data.Snippet{Text: "dumps()"},
					Replace: data.Selection{Begin: 5, End: 5},
				},
			},
		},
	}

	resp := data.NewAPIResponse(data.APIRequest{
		SelectedBuffer: data.NewBuffer("json.").Select(data.Cursor(5)),
	})
	resp.Completions = compls
	m.Returned(resp, time.Now())
	assert.EqualValues(t, 1, m.read(false).AtLeastOneShown)
	assert.EqualValues(t, 1, m.read(false).AtLeastOneNewShown)
	assert.EqualValues(t, 3, m.read(false).Shown)
	assert.EqualValues(t, 1, m.read(false).Triggered)

	m.BufferEdited("json.d", 6)
	assert.Empty(t, m.read(false).SelectedBySource)

	var newCompls []data.NRCompletion
	for _, c := range compls {
		c.RCompletion.Replace = data.Selection{Begin: 5, End: 6}
		newCompls = append(newCompls, c)
	}
	resp = data.NewAPIResponse(data.APIRequest{
		SelectedBuffer: data.NewBuffer("json.d").Select(data.Cursor(6)),
	})
	resp.Completions = newCompls
	m.Returned(resp, time.Now())
	assert.EqualValues(t, 2, m.read(false).Triggered)
	assert.EqualValues(t, 3, m.read(false).Shown)

	m.BufferEdited("json.dump(obj, fp)", 13)
	assert.NotEmpty(t, m.read(false).SelectedBySource)
	assert.EqualValues(t, 1, m.read(false).NumSelected)

	stats, err := m.readCompletionStats()
	require.NoError(t, err)
	assert.NotEmpty(t, stats)
	stat := stats[0]
	// selected json.dump(obj, fp) after typing json.d
	assert.EqualValues(t, lang.Python, stat.Language)
	assert.EqualValues(t, completionSelected, stat.CompletionType)
	assert.EqualValues(t, 1, stat.Rank)
	assert.EqualValues(t, 12, stat.NumCharsInserted)
	assert.EqualValues(t, 7, stat.NumTokensInserted)
	assert.EqualValues(t, 1, stat.NumCharsReplaced)
	assert.EqualValues(t, 2, stat.NumTokensReplaced)
	assert.EqualValues(t, 5, stat.NumPlaceholderChars)
	assert.EqualValues(t, false, stat.Lexical)
	// flush metrics
	ms.Flush()
	stats, err = m.readCompletionStats()
	require.NoError(t, err)
	assert.Empty(t, stats)

	m.read(true)
	resp = data.NewAPIResponse(data.APIRequest{
		SelectedBuffer: data.NewBuffer("json.").Select(data.Cursor(5)),
	})
	resp.Completions = compls
	m.Returned(resp, time.Now())
	m.BufferEdited("json.dumps(obj)", 14)
	assert.Empty(t, m.read(false).SelectedBySource)
	assert.EqualValues(t, 3, m.read(false).Shown)

	stats, err = m.readCompletionStats()
	require.NoError(t, err)
	assert.Empty(t, stats)
}

func TestRequestMetric_Dedup(t *testing.T) {
	m := NewMetrics().Get(lang.Python)

	m.BufferEdited("json.", 5)
	m.Requested()
	m.Requested()
	m.Expected(true)
	m.Expected(true)
	m.ReturnedCompat("json.", 5, []response.EditorCompletion{
		{Insert: "json.dump", Source: response.TraditionalCompletionSource},
		{Insert: "json.dumps", Source: response.AttributeModelCompletionSource}}, time.Now())
	m.ReturnedCompat("json.", 5, []response.EditorCompletion{
		{Insert: "json.dump", Source: response.TraditionalCompletionSource},
		{Insert: "json.dumps", Source: response.AttributeModelCompletionSource}}, time.Now())
	assert.EqualValues(t, RequestCounts{Total: 1, Expected: 1}, m.read(false).Requested)
	assert.EqualValues(t, RequestCounts{Total: 2, Expected: 2}, m.read(false).RequestedRaw)
	assert.EqualValues(t, 1, m.read(false).AtLeastOneShown)
	assert.EqualValues(t, 1, m.read(false).AtLeastOneNewShown)
	assert.EqualValues(t, 2, m.read(false).Shown)
	assert.EqualValues(t, 1, m.read(false).Triggered)
	assert.EqualValues(t,
		SourceBreakdown{response.TraditionalCompletionSource: 1, response.AttributeModelCompletionSource: 1},
		m.read(false).ShownBySource)
	assert.EqualValues(t,
		SourceBreakdown{response.TraditionalCompletionSource: 1, response.AttributeModelCompletionSource: 1},
		m.read(false).AtLeastOneShownBySource)
	assert.EqualValues(t,
		SourceBreakdown{response.TraditionalCompletionSource: 1, response.AttributeModelCompletionSource: 1},
		m.read(false).AtLeastOneNewShownBySource)

	m.read(true)
	m.BufferEdited("json.d", 6)
	m.Requested()
	m.Requested()
	m.Expected(false)
	m.Expected(false)
	assert.EqualValues(t, RequestCounts{Total: 1, Unexpected: 1}, m.read(false).Requested)
	assert.EqualValues(t, RequestCounts{Total: 2, Unexpected: 2}, m.read(false).RequestedRaw)

	m.read(true)
	m.BufferEdited("json.du", 7)
	m.Requested()
	m.Requested()
	m.Errored()
	m.Errored()
	assert.EqualValues(t, RequestCounts{Total: 1, Error: 1}, m.read(false).Requested)
	assert.EqualValues(t, RequestCounts{Total: 2, Error: 2}, m.read(false).RequestedRaw)
}

func TestIdentifierBegin(t *testing.T) {
	assert.Equal(t, 0, legacyComplStart(" abc12 ", 0, "abc12"))
	assert.Equal(t, 1, legacyComplStart(" abc12 ", 1, "abc12"))
	assert.Equal(t, 1, legacyComplStart(" abc12 ", 2, "abc12"))
	assert.Equal(t, 1, legacyComplStart(" abc12 ", 3, "abc12"))
	assert.Equal(t, 1, legacyComplStart(" abc12 ", 4, "abc12"))
	assert.Equal(t, 1, legacyComplStart(" abc12 ", 5, "abc12"))
	assert.Equal(t, 1, legacyComplStart(" abc12 ", 6, "abc12"))
	assert.Equal(t, 7, legacyComplStart(" abc12 ", 7, "abc12"))

	assert.Equal(t, 0, legacyComplStart("++xy", 0, "xy"))
	assert.Equal(t, 1, legacyComplStart("++xy", 1, "xy"))
	assert.Equal(t, 2, legacyComplStart("++xy", 2, "xy"))
	assert.Equal(t, 2, legacyComplStart("++xy", 3, "xy"))
	assert.Equal(t, 2, legacyComplStart("++xy", 4, "xy"))

	assert.Equal(t, 0, legacyComplStart("", 0, ""))

	assert.Equal(t, 0, legacyComplStart("x", 0, "x"))
	assert.Equal(t, 0, legacyComplStart("x", 1, "x"))

	assert.Equal(t, 1, legacyComplStart(" x", 1, "x"))
	assert.Equal(t, 1, legacyComplStart(" x", 2, "x"))

	assert.Equal(t, 0, legacyComplStart("x ", 0, "x"))
	assert.Equal(t, 0, legacyComplStart("x ", 1, "x"))

	assert.Equal(t, 0, legacyComplStart("baz = foo.bar(x, y, z)", 0, "foo.bar(x, y, z)"))
	assert.Equal(t, 1, legacyComplStart("baz = foo.bar(x, y, z)", 1, "foo.bar(x, y, z)"))
	assert.Equal(t, 2, legacyComplStart("baz = foo.bar(x, y, z)", 2, "foo.bar(x, y, z)"))
	assert.Equal(t, 3, legacyComplStart("baz = foo.bar(x, y, z)", 3, "foo.bar(x, y, z)"))
	assert.Equal(t, 4, legacyComplStart("baz = foo.bar(x, y, z)", 4, "foo.bar(x, y, z)"))
	assert.Equal(t, 5, legacyComplStart("baz = foo.bar(x, y, z)", 5, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 6, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 7, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 8, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 9, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 10, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 11, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 12, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 13, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 14, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 15, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 16, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 17, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 18, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 19, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 20, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 21, "foo.bar(x, y, z)"))
	assert.Equal(t, 6, legacyComplStart("baz = foo.bar(x, y, z)", 22, "foo.bar(x, y, z)"))
}

func TestUsedMetric_Lexical_Golang(t *testing.T) {
	ms := NewMetrics()
	m := ms.Get(lang.Golang)

	m.BufferEdited("json.", 5)
	assert.Empty(t, m.read(false).SelectedBySource)

	compls := []data.NRCompletion{
		{
			RCompletion: data.RCompletion{
				Completion: data.Completion{
					Snippet: data.Snippet{
						Text: "NewDecoder",
					},
					Replace: data.Selection{Begin: 5, End: 5},
				},
				Metrics: &lexicalproviders.LexicalMetrics{
					Probability:     1.0,
					Score:           1.0,
					NumVocabTokens:  1,
					ModelDurationMS: 1,
					PredictedMetrics: predict.PredictedMetrics{
						CuratedContextExists: true,
						CuratedContextUsed:   true,
					},
				},
			},
		},
		{
			RCompletion: data.RCompletion{
				Completion: data.Completion{
					Snippet: data.Snippet{
						Text: "MarshalIndent",
					},
					Replace: data.Selection{Begin: 5, End: 5},
				},
				Metrics: &lexicalproviders.LexicalMetrics{
					Probability:     2.0,
					Score:           2.0,
					NumVocabTokens:  2,
					ModelDurationMS: 2,
					PredictedMetrics: predict.PredictedMetrics{
						CuratedContextExists: true,
						CuratedContextUsed:   false,
					},
				},
			},
		},
	}

	resp := data.NewAPIResponse(data.APIRequest{
		SelectedBuffer: data.NewBuffer("json.").Select(data.Cursor(5)),
	})
	resp.Completions = compls
	m.Returned(resp, time.Now())
	assert.EqualValues(t, 1, m.read(false).AtLeastOneShown)
	assert.EqualValues(t, 1, m.read(false).AtLeastOneNewShown)
	assert.EqualValues(t, 2, m.read(false).Shown)
	assert.EqualValues(t, 1, m.read(false).Triggered)

	m.BufferEdited("json.M", 6)
	assert.Empty(t, m.read(false).SelectedBySource)

	var newCompls []data.NRCompletion
	for _, c := range compls {
		c.RCompletion.Replace = data.Selection{Begin: 5, End: 6}
		newCompls = append(newCompls, c)
	}
	resp = data.NewAPIResponse(data.APIRequest{
		SelectedBuffer: data.NewBuffer("json.M").Select(data.Cursor(6)),
	})
	resp.Completions = newCompls
	m.Returned(resp, time.Now())
	assert.EqualValues(t, 2, m.read(false).Triggered)
	assert.EqualValues(t, 2, m.read(false).Shown)

	m.BufferEdited("json.MarshalIndent", 18)
	assert.NotEmpty(t, m.read(false).SelectedBySource)
	assert.EqualValues(t, 1, m.read(false).NumSelected)

	stats, err := m.readCompletionStats()
	require.NoError(t, err)
	assert.NotEmpty(t, stats)
	stat := stats[0]

	// selected json.MarshalIndent after typing json.M
	assert.EqualValues(t, lang.Golang, stat.Language)
	assert.EqualValues(t, completionSelected, stat.CompletionType)
	assert.EqualValues(t, 1, stat.Rank)
	assert.EqualValues(t, 12, stat.NumCharsInserted)
	assert.EqualValues(t, 1, stat.NumTokensInserted)
	assert.EqualValues(t, 1, stat.NumCharsReplaced)
	assert.EqualValues(t, 1, stat.NumTokensReplaced)
	assert.EqualValues(t, 0, stat.NumPlaceholderChars)

	// check lexical metrics
	assert.EqualValues(t, true, stat.Lexical)
	assert.EqualValues(t, 2.0, stat.LexicalMetrics.Probability)
	assert.EqualValues(t, 2.0, stat.LexicalMetrics.Score)
	assert.EqualValues(t, 2, stat.LexicalMetrics.NumVocabTokens)
	assert.EqualValues(t, 2, stat.LexicalMetrics.ModelDurationMS)
	assert.EqualValues(t, true, stat.LexicalMetrics.CuratedContextExists)
	assert.EqualValues(t, false, stat.LexicalMetrics.CuratedContextUsed)

	// flush metrics
	ms.Flush()
	stats, err = m.readCompletionStats()
	require.NoError(t, err)
	assert.Empty(t, stats)
}

func (m *Metrics) readCompletionStats() ([]CompletionStat, error) {
	m.cLock.Lock()
	defer m.cLock.Unlock()

	data := m.encodeStatsLocked()

	buf, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	reader, err := gzip.NewReader(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}

	var stats []CompletionStat
	if err := json.NewDecoder(reader).Decode(&stats); err != nil {
		return nil, err
	}
	return stats, nil
}
