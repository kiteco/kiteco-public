package dynamicanalysis

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertMapsEqual(t *testing.T, expected map[string]interface{}, actual map[string]interface{}) bool {
	return AssertMapsEqualInternal(t, expected, actual, "")
}

func AssertListsEqualInternal(t *testing.T, expected []interface{}, actual []interface{}, stem string) bool {
	assert.Equal(t, len(expected), len(actual), "at actual%s", stem)
	for i, v := range expected {
		path := fmt.Sprintf("%s[%d]", stem, i)
		if !AssertObjectsEqualInternal(t, v, actual[i], path) {
			return false
		}
	}
	return true
}

func AssertMapsEqualInternal(t *testing.T, expected map[string]interface{}, actual map[string]interface{}, stem string) bool {
	for k, v := range expected {
		path := fmt.Sprintf("%s[%q]", stem, k)

		actualVal, ok := actual[k]
		if !ok {
			return assert.Fail(t, "actual%s was missing", path)
		}

		if !AssertObjectsEqualInternal(t, v, actualVal, path) {
			return false
		}
	}
	return true
}

func AssertObjectsEqualInternal(t *testing.T, expected interface{}, actual interface{}, path string) bool {
	switch expectedVal := expected.(type) {
	case map[string]interface{}:
		if !assert.IsType(t, map[string]interface{}{}, actual, "at actual%s", path) {
			return false
		}
		return AssertMapsEqualInternal(t, expectedVal, actual.(map[string]interface{}), path)
	case []interface{}:
		if !assert.IsType(t, []interface{}{}, actual, "at actual%s", path) {
			return false
		}
		return AssertListsEqualInternal(t, expectedVal, actual.([]interface{}), path)
	default:
		return assert.EqualValues(t, expected, actual, "at actual%s", path)
	}
}

func Test(t *testing.T) {
	if !dockerTests {
		t.Skip("use --docker to run tests that require docker")
	}

	code := `
import collections
y = collections.defaultdict(list)
print(y)
`

	expected := Tree{"RootArray": []interface{}{map[string]interface{}{"Import": map[string]interface{}{"names": []interface{}{map[string]interface{}{"alias": map[string]interface{}{"k_lineno": 7., "k_col_offset": 1., "name": "collections", "asname": interface{}(nil)}}}, "k_lineno": 5, "k_col_offset": 5.}}, map[string]interface{}{"Assign": map[string]interface{}{"targets": []interface{}{map[string]interface{}{"Name": map[string]interface{}{"id": "y", "ctx": map[string]interface{}{"Store": map[string]interface{}{"k_lineno": 26, "k_col_offset": 14.}}, "k_lineno": 8, "k_col_offset": 1}}}, "value": map[string]interface{}{"Call": map[string]interface{}{"func": map[string]interface{}{"Attribute": map[string]interface{}{"ctx": map[string]interface{}{"Load": map[string]interface{}{"k_lineno": 37, "k_col_offset": 17}}, "k_type": "type", "k_num_evals": 1, "k_lineno": 8, "k_col_offset": 5, "value": map[string]interface{}{"Name": map[string]interface{}{"k_col_offset": 5, "id": "collections", "ctx": map[string]interface{}{"Load": map[string]interface{}{"k_lineno": 37, "k_col_offset": 17}}, "k_type": "module", "k_module_fqn": "collections", "k_num_evals": 1, "k_lineno": 8}}, "attr": "defaultdict"}}, "keywords": map[string]interface{}{"RootArray": []interface{}{}}, "k_type": "collections.defaultdict", "k_num_evals": 1, "k_col_offset": 5, "args": map[string]interface{}{"RootArray": []interface{}{map[string]interface{}{"Name": map[string]interface{}{"ctx": map[string]interface{}{"Load": map[string]interface{}{"k_lineno": 37, "k_col_offset": 17}}, "k_type": "type", "k_num_evals": 1, "k_lineno": 8, "k_col_offset": 29, "id": "list"}}}}, "starargs": interface{}(nil), "kwargs": interface{}(nil), "k_lineno": 8}}, "k_lineno": 8, "k_col_offset": 1}}, map[string]interface{}{"Print": map[string]interface{}{"dest": interface{}(nil), "values": map[string]interface{}{"RootArray": []interface{}{map[string]interface{}{"Name": map[string]interface{}{"id": "y", "ctx": map[string]interface{}{"Load": map[string]interface{}{"k_col_offset": 17, "k_lineno": 37}}, "k_type": "collections.defaultdict", "k_num_evals": 1, "k_lineno": 9, "k_col_offset": 7}}}}, "nl": "True", "k_lineno": 9, "k_col_offset": 1}}}}

	trace, err := Trace(code, DefaultTraceOptions)
	require.NoError(t, err)

	AssertMapsEqual(t, expected, trace.Tree)
}
