package corpustest

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalcomplete/api"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-golib/complete/corpustests"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	corpusDir   = "./corpus"
	buildPrefix = "+build"
)

var commentsByLang = map[lang.Language]string{
	lang.Golang:     "//",
	lang.JavaScript: "//",
	lang.Vue:        "//",
	lang.Python:     "#",
}

var cursorByLang = map[lang.Language]string{
	lang.Golang:     "$",
	lang.JavaScript: "^",
	lang.Vue:        "^",
	lang.Python:     "$",
}

var models *lexicalmodels.Models

func init() {
	datadeps.Enable()

	var err error
	models, err = lexicalmodels.NewModels(lexicalmodels.DefaultModelOptions)
	if err != nil {
		panic(err)
	}
}

func runFromCorpus(t *testing.T, timeout time.Duration, matchedStates ...string) {
	dirs, err := ioutil.ReadDir(corpusDir)
	require.NoError(t, err)

	for _, dir := range dirs {
		dpath := filepath.Join(corpusDir, dir.Name())
		langDir, err := ioutil.ReadDir(dpath)
		require.NoError(t, err)

		for _, file := range langDir {
			l := lang.FromExtension(strings.TrimPrefix(filepath.Ext(file.Name()), "."))

			commentPrefix, exists := commentsByLang[l]
			if !exists {
				continue
			}

			cursor, exists := cursorByLang[l]
			require.True(t, exists, "no cursor specified for %s", l)

			fpath := filepath.Join(dpath, file.Name())
			if !strings.HasSuffix(fpath, lang.LanguageTags[l].Ext) {
				continue
			}
			runTestCases(t, fpath, commentPrefix, cursor, timeout, api.DefaultLexicalOptions, matchedStates...)
		}
	}
}

func requireTestCase(t *testing.T, file, commentPrefix, cursor string) corpustests.TestCase {
	f, err := os.Open(file)
	require.NoError(t, err)
	defer f.Close()

	var before, after, description []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.Contains(text, buildPrefix) {
			// ignore build constraints
			continue
		}
		if strings.Contains(text, commentPrefix) {
			description = append(description, strings.TrimPrefix(strings.TrimSpace(text), commentPrefix))
		} else {
			if len(description) > 0 {
				after = append(after, text)
			} else {
				before = append(before, text)
			}
		}
	}

	require.NoError(t, scanner.Err())
	assert.True(t, len(description) > 0)
	assert.True(t, strings.HasPrefix(strings.TrimSpace(description[0]), "TEST"))

	require := func(cond bool, errStr string, args ...interface{}) {
		if !cond {
			t.Errorf("error parsing test case %s", file)
			t.Errorf(errStr, args...)
			t.FailNow()
		}
	}
	testCase := corpustests.CreateTestCase(t, "", file, cursor, description[1:], require)

	all := append(before, testCase.Insert) // TODO: indent somehow?
	all = append(all, after...)
	buf := strings.Join(all, "\n")

	// cursor is still in the text so lets remove this
	parts := strings.Split(buf, cursor)
	require(len(parts) == 2 || len(parts) == 3, "expected exactly 2 or 3 (got %d) parts in the fn buf:\n%s\n", len(parts), string(buf))

	buf = strings.Join(parts, "")
	buffer := data.NewBuffer(buf)
	var sb data.SelectedBuffer
	if len(parts) == 2 {
		sb = buffer.Select(data.Cursor(len(parts[0])))
	} else {
		start := len(parts[0])
		end := start + len(parts[1])
		sb = buffer.Select(data.Selection{Begin: start, End: end})
	}
	testCase.SB = sb

	return testCase
}

func runTestCases(t *testing.T, file, commentPrefix, cursor string, timeout time.Duration, opts api.CompleteOptions, matchedStates ...string) {
	c := requireTestCase(t, file, commentPrefix, cursor)
	t.Run(c.Name, func(t *testing.T) {
		for _, s := range matchedStates {
			if c.Status != s {
				t.Skipf("skipping %s test %s", c.Status, c.Name)
				return
			}
		}

		require.NoError(t, kitectx.Background().WithTimeout(timeout, func(ctx kitectx.Context) error {
			runTestCase(ctx, t, c, opts)
			return nil
		}))
	})
}

func runTestCase(ctx kitectx.Context, t *testing.T, tc corpustests.TestCase, options api.CompleteOptions) {
	compls := requireCompletions(ctx, t, tc, options)
	corpustests.RunTestCase(ctx, t, tc, compls)
}

func requireCompletions(ctx kitectx.Context, t *testing.T, tc corpustests.TestCase, options api.CompleteOptions) []data.RCompletion {
	completionAPI := api.New(ctx.Context(), api.Options{
		Models: models,
	}, licensing.Pro)
	req := data.APIRequest{
		SelectedBuffer: tc.SB,
		UMF:            data.UMF{Filename: tc.Filename},
	}
	options.BlockDebug = true

	resp := completionAPI.Complete(ctx, options, req, nil, nil)
	defer completionAPI.Reset()

	require.NoError(t, resp.ToError(), "got error getting completions for case %s/%s: %v", tc.Filename, tc.Name, resp.Error)

	var compls []data.RCompletion
	for _, nrc := range resp.Completions {
		compls = append(compls, nrc.RCompletion)
	}
	return compls
}
