package test

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/lexical/lexicalmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/driver"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/pythonproviders"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmodels"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-golib/complete/corpustests"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/stringindex"
	"github.com/stretchr/testify/require"
)

const corpusDir = "./corpus"

var (
	rm            pythonresource.Manager
	models        *pythonmodels.Models
	lexicalModels *lexicalmodels.Models
	scanOpts      = pythonscanner.Options{
		ScanComments: true,
		ScanNewLines: true,
	}
	customOptions = map[string]api.CompleteOptions{}
	customProduct = map[string]licensing.ProductGetter{}
)

func init() {
	driver.UseTestProviders()
	datadeps.Enable()

	pythonproviders.SetUseGGNNCompletions(true)

	var errc <-chan error
	rm, errc = pythonresource.NewManager(pythonresource.SmallOptions)
	if err := <-errc; err != nil {
		panic(err)
	}

	var err error
	models, err = pythonmodels.New(pythonmodels.DefaultOptions)
	if err != nil {
		panic(err)
	}

	lexicalModels, err = lexicalmodels.NewModels(lexicalmodels.DefaultModelOptions)
	if err != nil {
		panic(err)
	}

	noElision := api.IDCCCompleteOptions
	noElision.NoElision = true
	customOptions["no_elision.py"] = noElision
	noSnippets := api.IDCCCompleteOptions
	noSnippets.NoSnippets = true
	customOptions["no_snippets.py"] = noSnippets
	noEmptyCalls := api.IDCCCompleteOptions
	noEmptyCalls.NoEmptyCalls = true
	customOptions["no_empty_calls.py"] = noEmptyCalls
	noSubtokenDecoder := api.IDCCCompleteOptions
	noSubtokenDecoder.MixOptions.GGNNSubtokenEnabled = false
	noSubtokenDecoder.ScheduleOptions.GGNNSubtokenEnabled = false
	customOptions["arg_placeholder_with_call_model.py"] = noSubtokenDecoder
	customOptions["call_with_call_model.py"] = noSubtokenDecoder
	customOptions["ggnn_subtoken_with_call_model.py"] = noSubtokenDecoder
	customOptions["filtering_with_call_model.py"] = noSubtokenDecoder

	customProduct["kitebasic.py"] = licensing.Free
}

func runOneTestFromCorpus(t *testing.T, timeout time.Duration, file string, testName string, matchedStates ...string) {
	file = filepath.Join(corpusDir, file)
	base := filepath.Base(file)
	opts, custom := customOptions[base]
	if !custom {
		opts = api.IDCCCompleteOptions
		opts.MixOptions.GGNNSubtokenEnabled = true
		opts.ScheduleOptions.GGNNSubtokenEnabled = true
	}
	product, custom := customProduct[base]
	if !custom {
		product = licensing.Pro
	}
	runTestCases(t, base, timeout, opts, product, testName, matchedStates...)

}

func runFromCorpus(t *testing.T, timeout time.Duration, matchedStates ...string) {
	files, err := filepath.Glob(filepath.Join(corpusDir, "*.py"))
	require.NoError(t, err)

	for _, file := range files {
		// a sub-test for each file
		base := filepath.Base(file)
		t.Run(base, func(t *testing.T) {
			opts, custom := customOptions[base]
			if !custom {
				opts = api.IDCCCompleteOptions
				opts.MixOptions.GGNNSubtokenEnabled = true
				opts.ScheduleOptions.GGNNSubtokenEnabled = true
			}
			product, custom := customProduct[base]
			if !custom {
				product = licensing.Pro
			}
			runTestCases(t, base, timeout, opts, product, "", matchedStates...)
		})
	}
}

func printNode(n pythonast.Node) string {
	var b bytes.Buffer
	pythonast.PrintPositions(n, &b, "\t")
	return b.String()
}

func requireTestFile(t *testing.T, filename string) []byte {
	buf, err := ioutil.ReadFile(filepath.Join(corpusDir, filename))
	require.NoError(t, err)
	return buf
}

func requireTestCase(t *testing.T, filename string, buf []byte, fn *pythonast.FunctionDefStmt) corpustests.TestCase {
	require := func(cond bool, fstr string, args ...interface{}) {
		if !cond {
			t.Errorf("error parsing test case %s/%s:\n%s\n", filename, fn.Name.Ident.Literal, printNode(fn))
			t.Errorf(fstr, args...)
			t.FailNow()
		}
	}

	var se *pythonast.StringExpr
	pythonast.Inspect(fn, func(n pythonast.Node) bool {
		if pythonast.IsNil(n) || se != nil {
			return false
		}

		s, ok := n.(*pythonast.StringExpr)
		if !ok {
			return true
		}

		for _, w := range s.Strings {
			if strings.HasPrefix(w.Literal, "'''TEST") {
				se = s
				break
			}
		}
		return true
	})

	require(se != nil, "unable to find TEST string")
	require(len(se.Strings) == 1, "test string should have only one word, got %d", len(se.Strings))

	lines := strings.Split(se.Strings[0].Literal, "\n")
	testCase := corpustests.CreateTestCase(t, fn.Name.Ident.Literal, filename, "$", lines, require)

	// +1 for colon and +1 for newline
	bodyBegin := int(fn.RightParen.End) + 2

	// place the insert text immediately before the string expression
	// describing the test case, this is so that we ensure that the resulting
	// text is syntactically valid and remove the string test case
	// to avoid polluting the context for the ggnn model
	bodyBuf := bytes.Join([][]byte{
		buf[bodyBegin:se.Begin()],
		[]byte(testCase.Insert + "\n"),
		buf[se.End():fn.End()],
	}, nil)

	// TODO: super hacky
	indent := int(fn.Body[0].Begin()) - bodyBegin
	var trimmed [][]byte
	for _, line := range bytes.Split(bodyBuf, []byte("\n")) {
		if len(line) > indent {
			line = line[indent:]
		}
		trimmed = append(trimmed, line)
	}

	bodyBuf = bytes.Join(trimmed, []byte("\n"))

	// cursor is still in the text so lets remove this
	parts := bytes.Split(bodyBuf, []byte("$"))

	require(len(parts) == 2 || len(parts) == 3, "expected exactly 2 or 3 (got %d) parts in the fn buf:\n%s\n", len(parts), string(bodyBuf))

	bodyBuf = bytes.Join(parts, nil)
	buffer := data.NewBuffer(string(bodyBuf))
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

func runTestCases(t *testing.T, filename string, timeout time.Duration, opts api.CompleteOptions, productAccessor licensing.ProductGetter, testName string, matchedStates ...string) {
	buf := requireTestFile(t, filename)

	var mod *pythonast.Module
	err := kitectx.Background().WithTimeout(500*time.Millisecond, func(ctx kitectx.Context) error {
		var err error
		mod, err = pythonparser.Parse(ctx, buf, pythonparser.Options{
			ScanOptions: scanOpts,
		})
		return err
	})
	require.NoError(t, err)
	var testRun int
	for _, stmt := range mod.Body {
		c := requireTestCase(t, filename, buf, stmt.(*pythonast.FunctionDefStmt))
		if testName != "" && testName != c.Name {
			continue
		}
		testRun++
		t.Run(c.Name, func(t *testing.T) {
			for _, s := range matchedStates {
				if c.Status != s {
					t.Skipf("skipping %s test %s", c.Status, c.Name)
					return
				}
			}

			require.NoError(t, kitectx.Background().WithTimeout(timeout, func(ctx kitectx.Context) error {
				runTestCase(ctx, t, c, opts, productAccessor)
				return nil
			}))
		})
	}
	require.True(t, testName == "" || testRun > 0, "No test found with the name %s", testName)
}

func requireCompletions(ctx kitectx.Context, t *testing.T, tc corpustests.TestCase, options api.CompleteOptions, productAccessor licensing.ProductGetter) []data.RCompletion {
	completionAPI := api.New(ctx.Context(), api.Options{
		ResourceManager: rm,
		Models:          models,
		LexicalModels:   lexicalModels,
	}, productAccessor)
	req := data.APIRequest{
		SelectedBuffer: tc.SB,
		UMF:            data.UMF{Filename: tc.Filename},
	}
	options.BlockDebug = true
	options.DepthLimit = 3
	options.NoExactMatch = true
	resp := completionAPI.Complete(ctx, options, req, nil, nil)
	defer completionAPI.Reset()

	require.NoError(t, resp.ToError(), "got error getting completions for case %s/%s: %v", tc.Filename, tc.Name, resp.Error)

	// cycle through encodings to catch encoding-related bugs
	require.NoError(t, resp.EncodeOffsets(stringindex.UTF16))
	require.NoError(t, resp.EncodeOffsets(stringindex.UTF32))
	require.NoError(t, resp.EncodeOffsets(stringindex.UTF8))

	var compls []data.RCompletion
	for _, nrc := range resp.Completions {
		compls = append(compls, nrc.RCompletion)
	}
	return compls
}

func runTestCase(ctx kitectx.Context, t *testing.T, c corpustests.TestCase, options api.CompleteOptions, productAccessor licensing.ProductGetter) {
	compls := requireCompletions(ctx, t, c, options, productAccessor)
	corpustests.RunTestCase(ctx, t, c, compls)
}
