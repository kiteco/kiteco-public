package pythonresource_test

// TODO(naman) we should eventually augment this with statistical checks comparing new vs old data

import (
	"encoding/json"
	"testing"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/stretchr/testify/require"
)

var expectedArgSpecs = map[string]string{
	"requests.api.get": `{
			"args": [
				{"name": "url", "default_type": "", "default_value": "", "keyword_only": false, "types": null},
				{"name": "params", "default_type": "types.NoneType", "default_value": "None", "keyword_only": false, "types": null}
			], "vararg": "", "kwarg": "kwargs"
		}`,
	"json.dumps": `{
			"args": [
				{"name": "obj", "default_type": "", "default_value": "", "keyword_only": false, "types": null},
				{"name": "skipkeys", "default_type": "builtins.bool", "default_value": "False", "keyword_only": true, "types": null},
				{"name": "ensure_ascii", "default_type": "builtins.bool", "default_value": "True", "keyword_only": true, "types": null},
				{"name": "check_circular", "default_type": "builtins.bool", "default_value": "True", "keyword_only": true, "types": null},
				{"name": "allow_nan", "default_type": "builtins.bool", "default_value": "True", "keyword_only": true, "types": null},
				{"name": "cls", "default_type": "types.NoneType", "default_value": "None", "keyword_only": true, "types": null},
				{"name": "indent", "default_type": "types.NoneType", "default_value": "None", "keyword_only": true, "types": null},
				{"name": "separators", "default_type": "types.NoneType", "default_value": "None", "keyword_only": true, "types": null},
				{"name": "default", "default_type": "types.NoneType", "default_value": "None", "keyword_only": true, "types": null},
				{"name": "sort_keys", "default_type": "builtins.bool", "default_value": "False", "keyword_only": true, "types": null}
			], "vararg": "", "kwarg": "kw"
		}`,
}

var expectedReturnTypes = map[string][]string{
	// TODO(naman) why does analysis of requests source code yield requests.models.PreparedRequest here?
	"requests.api.get": []string{"requests.models.Response", "requests.models.PreparedRequest", "builtins.dict"},

	"os.path.join": []string{"builtins.str", "builtins.str"},
	"json.dumps":   []string{"builtins.str", "builtins.str"},
}

var expectedSymbolCountMins = map[string]int{
	"builtins.str.format": 1000,
	"json.dumps":          660000,
}

func TestSymbolCounts(t *testing.T) {
	rm := pythonresource.DefaultTestManager(t)

	for pathStr, minCount := range expectedSymbolCountMins {
		sym, err := rm.PathSymbol(pythonimports.NewDottedPath(pathStr))
		require.Nil(t, err)

		counts := rm.SymbolCounts(sym)
		require.NotNil(t, counts, "count should not be nil for %s", pathStr)
		require.True(t, counts.Expr >= minCount, "count too low for %s", pathStr)
	}
}

// TestArgSpecs tests that we can create a new Manager and use it to query ArgSpecs
func TestArgSpecs(t *testing.T) {
	rm := pythonresource.DefaultTestManager(t)

	for pathStr, specJSON := range expectedArgSpecs {
		sym, err := rm.PathSymbol(pythonimports.NewDottedPath(pathStr))
		require.Nil(t, err)

		actual := rm.ArgSpec(sym)

		expected := &pythonimports.ArgSpec{}
		require.NoError(t, json.Unmarshal([]byte(specJSON), expected))

		require.Equal(t, expected, actual, "incorrect argspec for %s", pathStr)
	}
}

func TestReturnTypes(t *testing.T) {
	rm := pythonresource.DefaultTestManager(t)

	for pathStr, expected := range expectedReturnTypes {
		sym, err := rm.PathSymbol(pythonimports.NewDottedPath(pathStr))
		require.Nil(t, err)

		var actual []string
		syms := rm.ReturnTypes(sym)
		for _, sym := range syms {
			actual = append(actual, sym.PathString())
		}

		for _, e := range expected {
			require.Contains(t, actual, e, "incorrect return type for %s (%v doesn't contain %v)", pathStr, actual, e)
		}
	}
}

// TestDistributionIndex tests that we can lookup distributions using a path, and successfully lookup "prefix" symbols
func TestDistributionIndex(t *testing.T) {
	rm := pythonresource.DefaultTestManager(t)

	expected := []string{
		"dogpile-cache",
		"dogpile-core",
	}
	var actual []string
	for _, dist := range rm.DistsForPkg("dogpile") {
		actual = append(actual, dist.Name)
	}
	require.ElementsMatch(t, expected, actual)

	// only dogpile.cache has a symbol graph in testdata
	_, err := rm.NewSymbol(keytypes.RequestsDistribution, pythonimports.NewPath("requests"))
	require.Nil(t, err)
}

func TestPopularSignatures(t *testing.T) {
	rm := pythonresource.DefaultTestManager(t)

	symDumps, _ := rm.NewSymbol(keytypes.BuiltinDistribution3, pythonimports.NewDottedPath("json.dumps"))
	psDumps := rm.PopularSignatures(symDumps)
	require.Len(t, psDumps, 4, "json.Dumps should have 2 popular signatures")
	require.Equal(t, "dict", psDumps[0].Args[0].Types[0].Name, "Most frequent type of first arg of dumps should be dict")
	require.Equal(t, "list", psDumps[0].Args[0].Types[1].Name, "Second most frequent type of first arg of dumps should be list")
	require.Equal(t, "int", psDumps[1].LanguageDetails.Python.Kwargs[0].Types[0].Name, "Most frequent type for indent keyword arg should be int")
}

func TestKeywordArgFrequency(t *testing.T) {
	rm := pythonresource.DefaultTestManager(t)

	symOpen, _ := rm.NewSymbol(keytypes.BuiltinDistribution3, pythonimports.NewDottedPath("builtins.open"))
	openModeFreq, _ := rm.KeywordArgFrequency(symOpen, "mode")
	require.True(t, openModeFreq > 500, "Frequency of mode named arg for open should be > 500")

	symDumps, _ := rm.NewSymbol(keytypes.BuiltinDistribution3, pythonimports.NewDottedPath("json.dumps"))
	dumpsIndentFreq, _ := rm.KeywordArgFrequency(symDumps, "indent")
	dumpsEnsureASCIIFreq, _ := rm.KeywordArgFrequency(symDumps, "ensure_ascii")

	require.True(t, dumpsIndentFreq > dumpsEnsureASCIIFreq, "For dumps, frequency of indent should be > than ensure_ascii freq")

}
