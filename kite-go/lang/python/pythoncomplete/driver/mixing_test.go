package driver

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *scheduler) generateCompletions(completionMap map[pythonproviders.Provider][]pythonproviders.MetaCompletion, sb data.SelectedBuffer) {
	for p, comps := range completionMap {
		for _, c := range comps {
			s.GotCompletion(workItem{Provider: p}, sb, c)
		}
	}
}

func TestDedup(t *testing.T) {
	testScheduler := newScheduler(func() {})
	testBuffer := data.NewBuffer("").Select(data.Cursor(0))

	specs := &pythonimports.ArgSpec{Args: []pythonimports.Arg{pythonimports.Arg{Name: "filename"}, pythonimports.Arg{Name: "mode"}}}

	comp := data.Completion{
		Snippet: data.BuildSnippet(fmt.Sprintf(
			"open(%s,%s)",
			data.HoleWithPlaceholderMarks("filename"),
			data.HoleWithPlaceholderMarks("mode"),
		)),
		Replace: data.Selection{},
	}
	completionMap := map[pythonproviders.Provider][]pythonproviders.MetaCompletion{
		pythonproviders.CallPatterns{}: {
			{Source: response.GlobalPopularPatternCompletionSource,
				CallModelMeta: &pythonproviders.CallModelMeta{ArgSpec: specs},
				Completion:    comp,
			},
			{Source: response.GlobalPopularPatternCompletionSource,
				CallModelMeta: &pythonproviders.CallModelMeta{ArgSpec: specs},
				Completion:    comp,
			},
		}}

	testScheduler.generateCompletions(completionMap, testBuffer)
	result := testScheduler.Mix(kitectx.Context{}, MixOptions{}, nil, pythonproviders.Global{}, testBuffer)
	fmt.Println(result)
	require.Len(t, result, 1, "Completion should be dedupped")
}

// - Fixture tests

func loadFixture(t *testing.T, s scheduler, filename string) {
	f, err := os.Open(filename)
	require.NoError(t, err)

	dec := json.NewDecoder(f)
	var fixture Fixture
	require.NoError(t, dec.Decode(&fixture))
	_, err = dec.Token()
	require.Equal(t, io.EOF, err)

	s.FromFixture(fixture)
}

func TestBasic(t *testing.T) {
	testScheduler := newScheduler(func() {})
	loadFixture(t, testScheduler, "mixing_tests/cache-fixtures/basic_test.json")

	b, err := ioutil.ReadFile("mixing_tests/inputs/basic_test.py")
	require.NoError(t, err)
	text := string(b)

	var sb data.SelectedBuffer
	switch parts := strings.Split(text, "$"); len(parts) {
	case 1:
		sb = data.NewBuffer(text).Select(data.Cursor(len(text)))
	case 2:
		sb = data.NewBuffer(strings.Join(parts, "")).Select(data.Cursor(len(parts[0])))
	default:
	}

	result := testScheduler.Mix(kitectx.Context{}, MixOptions{}, nil, pythonproviders.Global{}, sb)
	fmt.Println(result)
	require.NotEmpty(t, result, "Mixing should return completions")
}

func TestMaxReturnedCompletions(t *testing.T) {
	testScheduler := newScheduler(func() {})
	loadFixture(t, testScheduler, "mixing_tests/cache-fixtures/requests_test.json")

	b, err := ioutil.ReadFile("mixing_tests/inputs/requests_test.py")
	require.NoError(t, err)
	text := string(b)

	var sb data.SelectedBuffer
	switch parts := strings.Split(text, "$"); len(parts) {
	case 1:
		sb = data.NewBuffer(text).Select(data.Cursor(len(text)))
	case 2:
		sb = data.NewBuffer(strings.Join(parts, "")).Select(data.Cursor(len(parts[0])))
	default:
	}

	for _, maxReturnedCompletions := range []int{20, 10, 5} {
		opts := MixOptions{
			MaxReturnedCompletions: maxReturnedCompletions,
		}
		result := testScheduler.Mix(kitectx.Context{}, opts, nil, pythonproviders.Global{}, sb)
		assert.Equal(t, maxReturnedCompletions, len(result), "Wrong number of completions")
	}
}

func TestSorting(t *testing.T) {
	testScheduler := newScheduler(func() {})
	loadFixture(t, testScheduler, "mixing_tests/cache-fixtures/example1.json")

	b, err := ioutil.ReadFile("mixing_tests/inputs/example1.py")
	require.NoError(t, err)
	text := string(b)

	var sb data.SelectedBuffer
	switch parts := strings.Split(text, "$"); len(parts) {
	case 1:
		sb = data.NewBuffer(text).Select(data.Cursor(len(text)))
	case 2:
		sb = data.NewBuffer(strings.Join(parts, "")).Select(data.Cursor(len(parts[0])))
	default:
	}

	result := testScheduler.Mix(kitectx.Context{}, MixOptions{}, nil, pythonproviders.Global{}, sb)
	fmt.Println(result)
	require.Len(t, result, 3)

	// results are sorted by normalized score, which equals score in this test.
	expected := []string{"my_func()", "my_obj", "my_pkg"}
	sort.Strings(expected)
	for _, compl := range result {
		fmt.Println(compl.Snippet.Text, compl.Source)
	}
}

func TestCallPromotion(t *testing.T) {
	testScheduler := newScheduler(func() {})
	testScheduler.opts.GGNNSubtokenEnabled = true
	loadFixture(t, testScheduler, "mixing_tests/cache-fixtures/call_promotion_test.json")

	b, err := ioutil.ReadFile("mixing_tests/inputs/call_promotion_test.py")
	require.NoError(t, err)
	text := string(b)

	var sb data.SelectedBuffer
	switch parts := strings.Split(text, "$"); len(parts) {
	case 1:
		sb = data.NewBuffer(text).Select(data.Cursor(len(text)))
	case 2:
		sb = data.NewBuffer(strings.Join(parts, "")).Select(data.Cursor(len(parts[0])))
	default:
	}
	// set the flag to true and set MaxreturnedCompletions the same as in kited.
	result := testScheduler.Mix(kitectx.Context{}, MixOptions{GGNNSubtokenEnabled: true, MaxReturnedCompletions: 0}, nil, pythonproviders.Global{}, sb)

	// expect these concrete calls at some position
	expected := []string{"get()", "get(my_url)", "get(my_url, my_data)"}
	texts := make(map[string]bool)
	for _, compl := range result {
		texts[compl.Snippet.Text] = true
	}
	for _, text := range expected {
		assert.True(t, texts[text], text)
	}
}

func TestMergeChildren(t *testing.T) {
	testScheduler := newScheduler(func() {})
	loadFixture(t, testScheduler, "mixing_tests/cache-fixtures/merge_children_test.json")

	b, err := ioutil.ReadFile("mixing_tests/inputs/merge_children_test.py")
	require.NoError(t, err)
	text := string(b)

	var sb data.SelectedBuffer
	switch parts := strings.Split(text, "$"); len(parts) {
	case 1:
		sb = data.NewBuffer(text).Select(data.Cursor(len(text)))
	case 2:
		sb = data.NewBuffer(strings.Join(parts, "")).Select(data.Cursor(len(parts[0])))
	default:
	}

	result := testScheduler.Mix(kitectx.Context{}, MixOptions{}, nil, pythonproviders.Global{}, sb)
	fmt.Println(result)
	require.Len(t, result, 10)

	emptyCallCompletions := map[string]map[string]struct{}{
		"loads()": {},
		"load()":  {},
	}
	for _, compl := range result {
		_, isEmptyCallCompletion := emptyCallCompletions[compl.Snippet.Text]
		if isEmptyCallCompletion {
			require.Equal(t, response.EmptyCallCompletionSource, compl.Source)
			continue
		}
		if compl.RCompletion.Source == response.GlobalPopularPatternCompletionSource || compl.RCompletion.Source == response.LocalPopularPatternCompletionSource {
			assert.Equal(t, "snippet", compl.Hint, "Should have hint of snippet")
		} else if compl.RCompletion.Source == response.CallModelCompletionSource {
			assert.Equal(t, "call", compl.Hint, "Should have hint of call")
		}
		assert.NotEqual(t, 0, len(compl.Snippet.Placeholders()), "Should have placeholders")
	}
}

func TestRenderingNoUnicode(t *testing.T) {
	testScheduler := newScheduler(func() {})
	loadFixture(t, testScheduler, "mixing_tests/cache-fixtures/basic_get.json")

	b, err := ioutil.ReadFile("mixing_tests/inputs/basic_get.py")
	require.NoError(t, err)
	text := string(b)

	var sb data.SelectedBuffer
	switch parts := strings.Split(text, "$"); len(parts) {
	case 1:
		sb = data.NewBuffer(text).Select(data.Cursor(len(text)))
	case 2:
		sb = data.NewBuffer(strings.Join(parts, "")).Select(data.Cursor(len(parts[0])))
	default:
	}

	var opts MixOptions
	opts.NoUnicode = true
	result := testScheduler.Mix(kitectx.Context{}, opts, nil, pythonproviders.Global{}, sb)
	fmt.Println(result)
	errUnicode := checkForUnicode(result)
	require.NoError(t, errUnicode, "Completions returned shouldn't contains any unicode char when NoUnicode is set to true : %v", err)
}

func TestRenderingStar(t *testing.T) {
	testScheduler := newScheduler(func() {})
	loadFixture(t, testScheduler, "mixing_tests/cache-fixtures/basic_get.json")

	b, err := ioutil.ReadFile("mixing_tests/inputs/basic_get.py")
	require.NoError(t, err)
	text := string(b)

	var sb data.SelectedBuffer
	switch parts := strings.Split(text, "$"); len(parts) {
	case 1:
		sb = data.NewBuffer(text).Select(data.Cursor(len(text)))
	case 2:
		sb = data.NewBuffer(strings.Join(parts, "")).Select(data.Cursor(len(parts[0])))
	default:
	}

	result := testScheduler.Mix(kitectx.Context{}, MixOptions{}, nil, pythonproviders.Global{}, sb)
	var hasSmart bool
	for _, c := range result {
		if c.Smart {
			require.True(t, strings.HasPrefix(c.Hint, "★"))
			hasSmart = true
		}
	}
	require.True(t, hasSmart, "no smart completions found")
}

func checkForUnicode(completions []data.NRCompletion) error {
	for _, c := range completions {
		if err := checkForUnicodeInCompletion(c.RCompletion); err != nil {
			return err
		}
	}
	return nil
}

func checkForUnicodeInCompletion(comp data.RCompletion) error {
	if checkForUnicodeInStr(comp.Snippet.Text) {
		return errors.Errorf("Error, unicode character detected in the snippet %s", comp.Snippet.Text)
	}
	if checkForUnicodeInStr(comp.Display) {
		return errors.Errorf("Error, unicode character detected in the display text %s", comp.Snippet.Text)
	}
	return nil
}

func checkForUnicodeInStr(str string) bool {
	for _, c := range str {
		if c > unicode.MaxASCII {
			return true
		}
	}
	return false
}
