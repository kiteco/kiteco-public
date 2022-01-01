package python

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireCallExample(t *testing.T, src string) (string, int64) {
	parts := strings.Split(src, "<caret>")
	switch len(parts) {
	case 1:
		return src, int64(len(src))
	case 2:
		return parts[0] + parts[1], int64(len(parts[0]))
	default:
		t.Errorf("invalid call example '%s'\n", src)
		t.FailNow()
		return "", -1
	}
}

func assertFindArgsStart(t *testing.T, src string, expected int64) {
	parsed, cursor := requireCallExample(t, src)
	actual := findArgsStart(parsed, cursor)
	assert.Equal(t, expected, actual, "test case: '%s', parsed: '%s'", src, parsed)
}

func TestFindCallStart(t *testing.T) {
	// support complete nested call expressions
	assertFindArgsStart(t, "foo(bar(),", 3)

	// take innermost call
	assertFindArgsStart(t, "foo(bar(,", 7)

	// incomplete tuple initilization.
	assertFindArgsStart(t, "foo((", 4)

	// after a call
	assertFindArgsStart(t, "foo(())<caret>", -1)

	// basic
	assertFindArgsStart(t, "foo(<caret>)", 3)

	// not in a call
	assertFindArgsStart(t, "foo<caret>(", -1)

	// nested call
	assertFindArgsStart(t, "foo(bar()<caret>", 3)
}

func requireTestData(t *testing.T, filename string) string {
	bytes, err := ioutil.ReadFile(fileutil.Join("testdata", filename))
	require.NoError(t, err)
	return string(bytes)
}

// Sends python content and the offset marked by <caret> in the python file to the signature manager
// It checks that the arg_index and in_kwargs status matches the expected values passed to this method
func assertMethodCall(t *testing.T, expectedArgIndex int, expectedInKwargs bool, pythonSrcFilename, responseJSONFilename string) {
	// get cursor and source file
	src, cursor := requireCallExample(t, requireTestData(t, pythonSrcFilename))

	// get json data for function
	raw := requireTestData(t, responseJSONFilename)

	var callee editorapi.CalleeResponse
	require.NoError(t, json.NewDecoder(bytes.NewBufferString(raw)).Decode(&callee))

	manager, err := NewManager(callee.Callee, callee.Signatures, callee.FuncName, pythonSrcFilename, src, cursor, true)
	require.NoError(t, err, "error initializing manager: %v", err)

	response := manager.Handle(src, cursor)

	require.NotNil(t, response, "Handle() response was nil.")

	assert.Equal(t, "python", response.Language)
	require.Len(t, response.Calls, 1)

	call := response.Calls[0]
	assert.Equal(t, expectedArgIndex, call.ArgIndex)
	assert.Equal(t, expectedInKwargs, call.LanguageDetails.Python.InKwargs)
}

func TestParameters_RegularParameter1(t *testing.T) {
	// 1st regular parameter
	assertMethodCall(t, 0, false, "json.loads.0.py", "json.loads.json")
}

func TestParameters_RegularParameter2(t *testing.T) {
	// 2nd regular parameter
	assertMethodCall(t, 1, false, "json.loads.1.py", "json.loads.json")
}

func TestParameters_KwArgParameter1(t *testing.T) {
	// known kwArg name
	assertMethodCall(t, 0, true, "json.loads.kwArg.py", "json.loads.json")
}

func TestParameters_KwArgParameter2(t *testing.T) {
	// an unknown kwarg name should still be recognized as a kwarg
	assertMethodCall(t, -1, true, "json.loads.kwArgUnknown.py", "json.loads.json")
}

func TestParameters_VarArgParameter1(t *testing.T) {
	// argument before a vararg parameter
	assertMethodCall(t, 0, false, "vararg.0.py", "vararg.json")
}

func TestParameters_VarArgParameter2(t *testing.T) {
	// 1st argument value of a vararg parameter
	assertMethodCall(t, 1, false, "vararg.1.py", "vararg.json")
}

func TestParameters_VarArgParameter3(t *testing.T) {
	// 2nd argument value of a vararg parameter
	assertMethodCall(t, 1, false, "vararg.2.py", "vararg.json")
}

func TestParameters_VarArgParameter4(t *testing.T) {
	// kw arg value after a vararg parameter
	assertMethodCall(t, 0, true, "vararg.kwArg.py", "vararg.json")
}

func TestParameters_DefaultValueParameter1(t *testing.T) {
	// no argument for method with default-value parameters
	assertMethodCall(t, 0, false, "regularArg.noArg.py", "regularArg.json")
}

func TestParameters_DefaultValueParameter2(t *testing.T) {
	// regular arguments with default values in the invocation
	assertMethodCall(t, 1, false, "regularArg.defValue.py", "regularArg.json")
}

func TestParameters_DefaultValueParameter3(t *testing.T) {
	// regular arguments with default values in reverse order in the invocation
	assertMethodCall(t, 1, false, "regularArg.defValueReverse.py", "regularArg.json")
}

func TestParameters_DefaultValueParameter4(t *testing.T) {
	// keyword argument of a method with only default-value parameters and a **kwarg
	assertMethodCall(t, 1, true, "regularArg.kwArg.py", "regularArg.json")
}

// issue #4651
func TestParameters_DefaultValueIncomplete1(t *testing.T) {
	// regular argument with incomplete value,
	assertMethodCall(t, 1, false, "json.loads.defValueIncomplete.py", "json.loads.json")
}

func TestParameters_DefaultValueIncomplete2(t *testing.T) {
	// regular argument with incomplete value and defaultValue in language_details.python
	assertMethodCall(t, 1, false, "regularArg.defValueIncomplete.py", "regularArg.json")
}

func TestParameters_ConstructorParameter1(t *testing.T) {
	// regular arg to constructor
	assertMethodCall(t, 0, false, "constructor.0.py", "constructor.json")
}

func TestParameters_ConstructorParameter2(t *testing.T) {
	// vararg (3rd value in the list) to a constructor
	assertMethodCall(t, 1, false, "constructor.1.py", "constructor.json")
}

func TestParameters_ConstructorParameter3(t *testing.T) {
	// kwarg to a constructor
	assertMethodCall(t, 0, true, "constructor.kwArg.py", "constructor.json")
}
