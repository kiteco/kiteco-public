package test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_KiteLocalSetup(t *testing.T) {
	project, err := startKiteLocal()
	require.NoError(t, err)
	defer project.Close()
}

func Test_SymbolReports2(t *testing.T) {
	project, err := startKiteLocal()
	require.NoError(t, err)
	defer project.Close()

	assertSymbolReport(t, project, "python;;;;datetime")
	assertSymbolReport(t, project, "python;;;;calendar")
	assertSymbolReport(t, project, "python;;;;collections")
	assertSymbolReport(t, project, "python;;;;numbers")
	assertSymbolReport(t, project, "python;;;;math")
	assertSymbolReport(t, project, "python;;;;math.pi")
	assertSymbolReport(t, project, "python;;;;random")
	assertSymbolReport(t, project, "python;;;;pickle")
	assertSymbolReport(t, project, "python;;;;zlib")
	assertSymbolReport(t, project, "python;;;;gzip")
	assertSymbolReport(t, project, "python;;;;os")
	assertSymbolReport(t, project, "python;;;;io")
	assertSymbolReport(t, project, "python;;;;logging")
	assertSymbolReport(t, project, "python;;;;json")
	assertSymbolReport(t, project, "python;;;;json;loads")
	assertSymbolReport(t, project, "python;;;;formatter")
	assertSymbolReport(t, project, "python;;;;threading")
	assertSymbolReport(t, project, "python;;;;ssl")
	assertSymbolReport(t, project, "python;;;;mmap")
	assertSymbolReport(t, project, "python;;;;email")
	assertSymbolReport(t, project, "python;;;;wave")
	assertSymbolReport(t, project, "python;;;;locale")
	assertSymbolReport(t, project, "python;;;;sys")
	assertSymbolReport(t, project, "python;;;;builtins;str")

	// Added in Python 3.x
	assertSymbolReport(t, project, "python;;;;concurrent;futures")
	assertSymbolReport(t, project, "python;;;;http")
	assertSymbolReport(t, project, "python;;;;http;client")
	assertSymbolReport(t, project, "python;;;;hashlib")

	// invalid ids
	assertMissingSymbolReport(t, project, "python;;;;;not_present")
	assertMissingSymbolReport(t, project, "javascript;console")
	assertMissingSymbolReport(t, project, "invalid-id")
}

func Test_SymbolReports3(t *testing.T) {
	project, err := startKiteLocal(keytypes.BuiltinDistribution3)
	require.NoError(t, err)
	defer project.Close()

	assertSymbolReport(t, project, "python;;;;datetime")
	assertSymbolReport(t, project, "python;;;;calendar")
	assertSymbolReport(t, project, "python;;;;collections")
	assertSymbolReport(t, project, "python;;;;numbers")
	assertSymbolReport(t, project, "python;;;;math")
	assertSymbolReport(t, project, "python;;;;math.pi")
	assertSymbolReport(t, project, "python;;;;random")
	assertSymbolReport(t, project, "python;;;;pickle")
	assertSymbolReport(t, project, "python;;;;zlib")
	assertSymbolReport(t, project, "python;;;;gzip")
	assertSymbolReport(t, project, "python;;;;os")
	assertSymbolReport(t, project, "python;;;;io")
	assertSymbolReport(t, project, "python;;;;logging")
	assertSymbolReport(t, project, "python;;;;json")
	assertSymbolReport(t, project, "python;;;;json;loads")
	assertSymbolReport(t, project, "python;;;;formatter")
	assertSymbolReport(t, project, "python;;;;threading")
	assertSymbolReport(t, project, "python;;;;concurrent.futures")
	assertSymbolReport(t, project, "python;;;;ssl")
	assertSymbolReport(t, project, "python;;;;mmap")
	assertSymbolReport(t, project, "python;;;;email")
	assertSymbolReport(t, project, "python;;;;http")
	assertSymbolReport(t, project, "python;;;;http;client")
	assertSymbolReport(t, project, "python;;;;wave")
	assertSymbolReport(t, project, "python;;;;locale")
	assertSymbolReport(t, project, "python;;;;sys")

	// invalid ids
	assertMissingSymbolReport(t, project, "javascript;console")
	assertMissingSymbolReport(t, project, "invalid-id")
}

func Test_SymbolReportFunction(t *testing.T) {
	project, err := startKiteLocal()
	require.NoError(t, err)
	defer project.Close()

	resp, err := symbolReport(project, "python;;;;json;loads")
	require.NoError(t, err)

	assertIsSymbolReport(t, resp)
	assertReportBasics(t, resp.Report)
	assertSymbolExtBasics(t, resp.Symbol, "python;;;;json;loads", "loads", "json")

	require.NotEmpty(t, resp.Symbol.Value)
	v := resp.Symbol.Value[0]
	// artifact of pkgexploration is that function types get associated with an arbitrary function.__class__
	assertValueExtBasics(t, v, "python;;;;json.loads", "json.loads", "function", "__class__", "python;;;;json.detect_encoding.__class__")

	assertIsFunction(t, v)
	assertFunctionDetails(t, v.Details.Function, 8, 7, 1) // returns int | float | list | unicode | dict | collections.OrderedDict | complex

	// data validation: symbol.value[0].details.function
	f := v.Details.Function

	// data validation: symbol.value[0].details.function.Parameters
	// 1st param of json.loads(s, encoding=None, ...)
	// encoding is a keyword-only arg as of 3.1, will be deprecated in 3.9
	require.NotEmpty(t, f.Parameters)
	assertParameter(t, f.Parameters[0], "s", false, false, false, "")
	assertParameter(t, f.Parameters[1], "encoding", true, false, false, "None")

	require.NotEmpty(t, f.Signatures)
	s := f.Signatures[0]
	assert.NotEmpty(t, s.Args, "expected args in the first signature")

	d := f.LanguageDetails.Python
	require.NotNil(t, d.Kwarg, "expected Python kw parameter")
	require.EqualValues(t, "kw", d.Kwarg.Name, "expected Python kw parameter")
	require.NotEmpty(t, d.KwargParameters, "Python kwargs must not be empty in kite local")
}

func Test_SymbolReportModule(t *testing.T) {
	project, err := startKiteLocal()
	require.NoError(t, err)
	defer project.Close()

	resp, err := symbolReport(project, "python;;;;json")
	require.NoError(t, err)

	assertIsSymbolReport(t, resp)
	// fixme kite is returning "." as the parent's name (both local and remote), possibly incorrect data?
	assertSymbolExtBasics(t, resp.Symbol, "python;;;;;json", "json", ".")

	v := resp.Symbol.Value[0]
	assertValueExtBasics(t, v, "python;;;;json", "json", "module", "", "")
	assertIsModule(t, v)
	assertModuleBasics(t, v, 15) // TODO: (hrysoula) why did this value change?
	assertSymbolBasics(t, v.Details.Module.Members[0], "python;;;;json;dumps", "dumps", "json")
}

func Test_SymbolReportType(t *testing.T) {
	project, err := startKiteLocal()
	require.NoError(t, err)
	defer project.Close()

	resp, err := symbolReport(project, "python;;;;string;Formatter")
	require.NoError(t, err)
	assertReportBasics(t, resp.Report)
	assert.Empty(t, resp.Report.Examples, "curated examples are empty in kite local")

	require.NotEmpty(t, resp.Symbol.Value)
	v := resp.Symbol.Value[0]
	assertIsType(t, v)
	assertValueExtBasics(t, v, "python;;;;string.Formatter", "string.Formatter", "type", "type", "python;;;;builtins.type")

	assertTypeDetails(t, v.Details.Type, true, true)
}

func Test_SymbolReportInstance(t *testing.T) {
	project, err := startKiteLocal()
	require.NoError(t, err)
	defer project.Close()

	resp, err := symbolReport(project, "python;;;;math.pi")
	require.NoError(t, err)
	assertReportBasics(t, resp.Report)

	assertSymbolExtBasics(t, resp.Symbol, "python;;;;math;pi", "pi", "math")

	v := resp.Symbol.Value[0]
	assertIsInstance(t, v)
	assertValueExtBasics(t, v, "python;;;;math.pi", "math.pi", "instance", "float", "python;;;;builtins.float")

	// fixme is repr=float in the instance details correct for math.pi?
	// assertInstanceDetails(t, v.Details.Instance, "python;;;;builtins.float", "float", "type", "type", "python;;;;builtins.type")
}

func Test_ValueReports(t *testing.T) {
	project, err := startKiteLocal()
	require.NoError(t, err)
	defer project.Close()

	assertValueReport(t, project, "python;;;;json")
	assertValueReport(t, project, "python;;;;json;loads")
	assertValueReport(t, project, "python;;;;formatter")
	assertValueReport(t, project, "python;;;;builtins;str")

	assertMissingValueReport(t, project, "python;;;;;not_present")
	assertMissingValueReport(t, project, "javascript;console")
	assertMissingValueReport(t, project, "unknownLang;invalid-id")
	assertMissingValueReport(t, project, "invalid-id")
}

func Test_ValueMembers(t *testing.T) {
	project, err := startKiteLocal()
	require.NoError(t, err)
	defer project.Close()

	assertValueMembers(t, project, "python;;;;json")
	assertValueMembers(t, project, "python;;;;json;loads")
	assertValueMembers(t, project, "python;;;;formatter")
	assertValueMembers(t, project, "python;;;;builtins;str")

	assertMissingValueMembers(t, project, "python;;;;;not_present")
	assertMissingValueMembers(t, project, "javascript;console")
	assertMissingValueMembers(t, project, "invalid-id")
}

func Test_ValueLinks(t *testing.T) {
	project, err := startKiteLocal()
	require.NoError(t, err)
	defer project.Close()

	resp, err := project.KitedClient.Get(fmt.Sprintf("/api/editor/value/%s/links", "python;;;;json"))
	require.NoError(t, err)
	require.EqualValues(t, http.StatusGone, resp.StatusCode, "usages has been deprecated")
}

func Test_Search(t *testing.T) {
	project, err := startKiteLocal()
	require.NoError(t, err)
	defer project.Close()

	_, err = search(project, "jso")
	require.NoError(t, err, "search data should be available")
}
