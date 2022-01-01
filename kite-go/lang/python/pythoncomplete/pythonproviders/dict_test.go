package pythonproviders

import (
	"fmt"
	"testing"

	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"

	"github.com/stretchr/testify/assert"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/require"
)

func requireTestCaseWithDictKey(t *testing.T, src string) testCase {
	sb := requireSelectedBuffer(t, src)

	global := Global{
		FilePath:        "/src.py",
		ResourceManager: pythonresource.DefaultTestManager(t),
		Models:          models,
		Product:         licensing.Pro,
	}
	global.Lexical.FilePath = global.FilePath
	global.Lexical.Models = lexicalModels

	inputs, err := NewInputs(kitectx.Background(), global, sb, true, false)
	require.NoError(t, err)
	return testCase{
		Orig:   src,
		Global: global,
		Inputs: inputs,
	}
}

func TestDictKeyBracketAccess(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = 25

theDict["$"]`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 2)
	comp := comps[1]
	assert.Equal(t, "[\"theKey\"]", comp.Snippet.Text)
}

func TestDictKeyBracketAccessWithSimpleQuotes(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = 25

theDict['$']`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 2)
	comp := comps[1]
	assert.Equal(t, "['theKey']", comp.Snippet.Text)
}

func TestDictKeyBracketStringPrefix(t *testing.T) {
	// If there's a string prefix, the detection of quote type will fail and we will default to "
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = 25

theDict[r'$']`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 2)
	comp := comps[1]
	assert.Equal(t, "[\"theKey\"]", comp.Snippet.Text)
}

func TestDictKeyBracketKeyWithStringPrefix(t *testing.T) {
	// If there's a string prefix, the detection of quote type will fail and we will default to "
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict[r"theKey"] = 25

theDict[r'$']`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 2)
	comp := comps[1]
	assert.Equal(t, "[\"theKey\"]", comp.Snippet.Text)
}

func TestDictKeyStringFormatPrefixAsKey(t *testing.T) {
	t.Skip()
	// Currently format string are considered as normal string, which shouldn't be the case
	// This test currently doesn't work because of that
	// If there's a string prefix, the detection of quote type will fail and we will default to "
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
id = 5
theDict[f"theKey{id}"] = 25

theDict[r'$']`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 2)
	comp := comps[1]
	// Should we return that or no completion at all?
	assert.Equal(t, "[\"theKey5\"]", comp.Snippet.Text)
}

func TestDictKeyValueType(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = "a string"

theDict["theKey"].$`)
	comps := requireCompletions(t, tc, Attributes{})
	assert.Len(t, comps, 46)
	assertContainsCompletion(t, comps, "format")
	assertContainsCompletion(t, comps, "lower")
}

func TestDictKeyOrder(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = "a string"
theDict["anotherKey"] = 25
theDict[1] = "Nice key!"

theDict["$"]`)
	comps := requireCompletions(t, tc, DictKeys{})
	// As the analysis allows value mutation for the tests, the access `["$"]` adds the key "" in the dict
	// So we have 4 completions instead of 3
	assert.Len(t, comps, 4)
	assert.Equal(t, "[\"\"]", comps[0].Snippet.Text)
	assert.Equal(t, "[\"anotherKey\"]", comps[1].Snippet.Text)
	assert.Equal(t, "[\"theKey\"]", comps[2].Snippet.Text)
	assert.Equal(t, "[1]", comps[3].Snippet.Text)
}

func TestDictKeyAttributeAccess(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = 25

theDict.t$`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 1)
	comp := comps[0]
	assert.Equal(t, "[\"theKey\"]", comp.Snippet.Text)
}

func TestDictKeyGetAccess(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = 25

theDict.get($)`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 1)
	comp := comps[0]
	assert.Equal(t, "\"theKey\"", comp.Snippet.Text)
}

func TestDictKeyGetAccessWithSimpleQuotes(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = 25

theDict.get('$')`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 1)
	comp := comps[0]
	assert.Equal(t, "'theKey'", comp.Snippet.Text)
}

func TestDictKeyPopAccess(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = 25

theDict.pop($)`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 1)
	comp := comps[0]
	assert.Equal(t, "\"theKey\"", comp.Snippet.Text)
}

func TestDictKeyBracketPrefixMatching(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = 25
theDict["anotherKey"] = 24

theDict["a$"]`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 2)
	// There a side effect of adding key during analysis as the access ["a"] is considered and that adds a key
	// in the keyMap, so we get 2 completions instead of 1
	comp := comps[1]
	assert.Equal(t, "[\"anotherKey\"]", comp.Snippet.Text)
}

func TestDictKeyAttributePrefixMatching(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = 25
theDict["anotherKey"] = 24

theDict.a$`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 1)
	comp := comps[0]
	assert.Equal(t, "[\"anotherKey\"]", comp.Snippet.Text)
}

func TestDictKeyGetPrefixMatching(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = 25
theDict["anotherKey"] = 24

theDict.get(a$)`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 1)
	comp := comps[0]
	assert.Equal(t, "\"anotherKey\"", comp.Snippet.Text)
}

func TestDictKeyAttributeIntegerPrefixMatching(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = 25
theDict["anotherKey"] = 24
theDict[18] = "babar"

theDict.1$`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 1)
	comp := comps[0]
	assert.Equal(t, "[18]", comp.Snippet.Text)
}

func TestDictKeyNoCompletionAttributeEmptyPrefix(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {}
theDict["theKey"] = 25
theDict["anotherKey"] = 24
theDict[18] = "babar"

theDict.$`)
	_, err := requireCompletionsOrError(t, tc, DictKeys{})
	assert.Equal(t, data.ProviderNotApplicableError{}, err, "DictKeyProvider should return NotApplicable on empty prefix for attribute access")

}

func TestDictKeyCompletionWithOrderedDict(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
import collections
theDict = collections.OrderedDict()
theDict["theKey"] = 25
theDict["anotherKey"] = 24
theDict[18] = "babar"

theDict["$"]`)
	comps := requireCompletions(t, tc, DictKeys{})
	fmt.Println(comps)
	assert.Len(t, comps, 3)
	comp := comps[0]
	assert.Equal(t, "[\"anotherKey\"]", comp.Snippet.Text)
}

func TestDictKeyCompletionWithDataFrame(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
import pandas
theDict = pandas.DataFrame()
theDict["theKey"] = 25
theDict["anotherKey"] = 24
theDict[18] = "babar"

theDict["$"]`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 4)
	comp := comps[1]
	assert.Equal(t, "[\"anotherKey\"]", comp.Snippet.Text)
}

func TestDictKeyDataFrameAttributeAccess(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
import pandas
theDict = pandas.DataFrame()
theDict["theKey"] = 25
theDict["anotherKey"] = 24
theDict["4invalid ident"] = "that's not right! It's invalid"
theDict[18] = "babar"

theDict.$`)
	comps := requireCompletions(t, tc, DictKeys{})
	assert.Len(t, comps, 4)
	comp := comps[1]
	assert.Equal(t, ".anotherKey", comp.Snippet.Text)
	// When inserting an attribute access, we keep the dot, so replace nothing (ie 186 -> 186)
	assert.Equal(t, data.Selection{Begin: 185, End: 186}, comp.Replace)
	compInvalid := comps[0]
	assert.Equal(t, "[\"4invalid ident\"]", compInvalid.Snippet.Text)
	// When inserting bracket, the replace should replace the dot (so 185 -> 186 for the replace)
	assert.Equal(t, data.NewSelection(185, 186), compInvalid.Replace)

}

func assertContainsCompletion(t *testing.T, comps []MetaCompletion, target string) {
	found := false
	var compList []string
	for _, c := range comps {
		if c.Snippet.Text == target {
			found = true
			break
		}
		compList = append(compList, c.Snippet.Text)
	}
	assert.True(t, found, "Completion %s not found in the list of completions (%v)", target, compList)
}

func TestDictBugSharedKeyFromFunction(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
def dictBuilder():
	return {"theKey": 18}

firstDict = dictBuilder()
firstDict["theSecondKey"] = 19

secondDict = dictBuilder()
secondDict.t$`)
	comps := requireCompletions(t, tc, DictKeys{})
	// This test is NOT representative of the reality, if you try this block of code in kited, you'll get 2 keys
	// in second dict (but not in the test, not sure why.
	// The bug it that the secondDict shouldn't contain the key `secondKey` but the keymap is shared between both instance
	assert.Len(t, comps, 1)
	// assertContainsCompletion(t, comps, "[\"secondKey\"]")
}

func TestDictBugSharedKeyFromFunctionToSingleton(t *testing.T) {
	tc := requireTestCaseWithDictKey(t, `
theDict = {"theKey": 18}
def dictBuilder():
	return theDict

firstDict = dictBuilder()
firstDict["theSecondKey"] = 19

secondDict = dictBuilder()
secondDict.t$`)
	comps := requireCompletions(t, tc, DictKeys{})
	// This time, as the function return a singleton, firstDict and secondDict point both the the same object
	// so we expected that they share there key set
	// It's the case in kited, not in this test
	// It's pretty hard to make the distinction between these 2 cases
	assert.Len(t, comps, 1)
	// assertContainsCompletion(t, comps, "[\"secondKey\"]")
}

func TestDictBugStringQuoting(t *testing.T) {
	src := `
theDict = {"the\nkey": 18}
theDict.the$`
	res, err := runProvider(t, DictKeys{}, src)
	require.NoError(t, err)

	require.True(t, res.containsRoot(data.Completion{
		Replace: data.Selection{Begin: -4},
		Snippet: data.NewSnippet(`["the\nkey"]`),
	}))
}
