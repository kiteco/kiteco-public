package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/client"
	"github.com/kiteco/kiteco/kite-go/client/internal/clientapp"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// starts a kite local test instance which loads the given distributions.
// if no distribution was passed, then stdlib 2.7 will be loaded
func startKiteLocal(dists ...keytypes.Distribution) (*clientapp.TestEnvironment, error) {
	if dists == nil {
		dists = []keytypes.Distribution{keytypes.BuiltinDistribution3}
	}

	opts := client.Options{
		LocalOpts: kitelocal.Options{
			Dists:                 dists,
			DisableDynamicLoading: true,
		},
	}

	env, err := clientapp.StartDefaultTestEnvironment(true, &opts)
	if err != nil {
		return env, err
	}

	return env, env.WaitForReady(10 * time.Second)
}

// symbol report

func symbolReport(project *clientapp.TestEnvironment, symbolID string) (*editorapi.ReportResponse, error) {
	var resp editorapi.ReportResponse
	err := project.KitedClient.GetJSON(fmt.Sprintf("/api/editor/symbol/%s", symbolID), &resp)
	return &resp, err
}

func assertSymbolReport(t *testing.T, project *clientapp.TestEnvironment, symbolID string) {
	_, err := symbolReport(project, symbolID)
	assert.NoError(t, err, "symbol data must be available for %s", symbolID)
}

func assertMissingSymbolReport(t *testing.T, project *clientapp.TestEnvironment, symbolID string) {
	_, err := symbolReport(project, symbolID)
	assert.Error(t, err, "symbol data must not be available for %s", symbolID)
}

// value reports

func valueReport(project *clientapp.TestEnvironment, valudID string) (*editorapi.ReportResponse, error) {
	var resp editorapi.ReportResponse
	err := project.KitedClient.GetJSON(fmt.Sprintf("/api/editor/value/%s", valudID), &resp)
	return &resp, err
}

func assertValueReport(t *testing.T, project *clientapp.TestEnvironment, valudID string) {
	_, err := valueReport(project, valudID)
	assert.NoError(t, err, "value data must be available for %s", valudID)
}

func assertMissingValueReport(t *testing.T, project *clientapp.TestEnvironment, valudID string) {
	_, err := valueReport(project, valudID)
	assert.Error(t, err, "value data must not be available for %s", valudID)
}

// value members

func valueMembers(project *clientapp.TestEnvironment, valueID string) (*editorapi.MembersResponse, error) {
	var resp editorapi.MembersResponse
	err := project.KitedClient.GetJSON(fmt.Sprintf("/api/editor/value/%s/members", valueID), &resp)
	return &resp, err
}

func assertValueMembers(t *testing.T, project *clientapp.TestEnvironment, valueID string) {
	_, err := valueMembers(project, valueID)
	assert.NoError(t, err, "value data must not be available for value %s", valueID)
}

func assertMissingValueMembers(t *testing.T, project *clientapp.TestEnvironment, valueID string) {
	_, err := valueMembers(project, valueID)
	assert.Error(t, err, "members must not be available for value %s", valueID)
}

// symbol assertions
func assertFunctionDetails(t *testing.T, f *editorapi.FunctionDetails, paramsCount int, returnCount int, signaturesCount int) {
	if paramsCount == 0 {
		assert.Empty(t, f.Parameters, "No parameters expected")
	} else {
		assert.Len(t, f.Parameters, paramsCount)
	}

	if returnCount == 0 {
		assert.Empty(t, f.ReturnValue, "No return values expected")
	} else {
		assert.Len(t, f.ReturnValue, returnCount)
	}

	if signaturesCount == 0 {
		assert.Empty(t, f.Signatures, "No signatures expected")
	} else {
		assert.Len(t, f.Signatures, signaturesCount)
	}
}

func search(project *clientapp.TestEnvironment, query string) (*editorapi.SearchResults, error) {
	var resp editorapi.SearchResults
	err := project.KitedClient.GetJSON(fmt.Sprintf("/api/editor/search?q=%s", query), &resp)
	return &resp, err
}

func assertIsFunction(t *testing.T, v *editorapi.ValueExt) {
	require.NotNil(t, v.Details)
	assert.NotNil(t, v.Details.Function)
	assert.Nil(t, v.Details.Type)
	assert.Nil(t, v.Details.Module)
	assert.Nil(t, v.Details.Instance)
}

func assertIsModule(t *testing.T, v *editorapi.ValueExt) {
	require.NotNil(t, v.Details)
	assert.NotNil(t, v.Details.Module)
	assert.Nil(t, v.Details.Function)
	assert.Nil(t, v.Details.Type)
	assert.Nil(t, v.Details.Instance)
}

func assertIsType(t *testing.T, v *editorapi.ValueExt) {
	require.NotNil(t, v.Details)
	assert.NotNil(t, v.Details.Type)
	assert.Nil(t, v.Details.Module)
	assert.Nil(t, v.Details.Function)
	assert.Nil(t, v.Details.Instance)
}

func assertIsInstance(t *testing.T, v *editorapi.ValueExt) {
	require.NotNil(t, v.Details)
	assert.NotNil(t, v.Details.Instance)
	assert.Nil(t, v.Details.Type)
	assert.Nil(t, v.Details.Module)
	assert.Nil(t, v.Details.Function)
}

func assertIsSymbolReport(t *testing.T, resp *editorapi.ReportResponse) {
	assert.EqualValues(t, "python", resp.Language)
	assert.Nil(t, resp.Value, "value details must be empty when a symbol report was requested")
}

func assertReportBasics(t *testing.T, r *editorapi.Report) {
	// fixme kited's report descriptions are empty, probably incorrect data? (both in remote/local environments)
	// assert.NotEmpty(t, r.DescriptionHTML, "descriptionHTML should NOT be empty")
	// assert.NotEmpty(t, r.DescriptionText, "descriptionText should NOT be empty")
	assert.Nil(t, r.Definition, "no definition expected for global symbols")
	assert.Empty(t, r.Examples, "examples are unavailable in kite local")
	assert.Nil(t, r.Links, "stackoverflow links were removed")
	assert.EqualValues(t, 0, r.TotalLinks, "stackoverflow links were removed")
	assert.Nil(t, r.Usages, "usages aren't present without local code")
	assert.EqualValues(t, 0, r.TotalUsages, "usages aren't present without local code")

}

func assertSymbolExtBasics(t *testing.T, s *editorapi.SymbolExt, id string, name string, parentName string) {
	assertSymbolBaseBasics(t, &s.SymbolBase, id, name, parentName)
}

func assertSymbolBasics(t *testing.T, s *editorapi.Symbol, id string, name string, parentName string) {
	assertSymbolBaseBasics(t, &s.SymbolBase, id, name, parentName)
}

func assertSymbolBaseBasics(t *testing.T, s *editorapi.SymbolBase, id string, name string, parentName string) {
	require.NotNil(t, s, "symbol must not be nil")
	assert.EqualValues(t, id, s.ID.String())
	assert.EqualValues(t, name, s.Name)

	if parentName == "" {
		require.True(t, s.Parent == nil || s.Parent.ID.String() == id, "parent of %s must NOT be present, but was %v", id, s.Parent)
	} else {
		require.NotNil(t, s.Parent, "parent of %s must be present", id)
		require.EqualValues(t, parentName, s.Parent.Name, "parent of %s must be present", id)
	}
}

func assertValueExtBasics(t *testing.T, v *editorapi.ValueExt, id string, repr string, kind string, typeValue string, typeID string) {
	require.NotNil(t, v, "value must not be nil")
	assert.EqualValues(t, id, v.ID.String())
	assert.EqualValues(t, repr, v.Repr)
	assert.EqualValues(t, kind, v.Kind)
	assert.EqualValues(t, typeValue, v.Type)
	assert.EqualValues(t, typeID, v.TypeID.String())
}

func assertTypeDetails(t *testing.T, d *editorapi.TypeDetails, expectMembers bool, expectConstructor bool) {
	require.NotNil(t, d)

	if expectMembers {
		assert.NotEmpty(t, d.Members)
		assert.True(t, d.TotalMembers > 0)
	} else {
		assert.Empty(t, d.Members)
		assert.EqualValues(t, 0, d.TotalMembers)
	}

	assert.NotNil(t, d.LanguageDetails)
	assert.NotNil(t, d.LanguageDetails.Python)
	if expectConstructor {
		assert.NotNil(t, d.LanguageDetails.Python.Constructor)
	} else {
		assert.Nil(t, d.LanguageDetails.Python.Constructor)
	}
}

func assertModuleBasics(t *testing.T, v *editorapi.ValueExt, membersCount int) {
	require.NotEmpty(t, v.Details.Module)

	if membersCount == 0 {
		assert.Empty(t, v.Details.Module.Members)
		assert.EqualValues(t, 0, v.Details.Module.TotalMembers)
	} else {
		memberCount := len(v.Details.Module.Members)
		assert.True(t, memberCount > 0 && memberCount <= membersCount)
		assert.EqualValues(t, membersCount, v.Details.Module.TotalMembers)
	}
}

func assertParameter(t *testing.T, p *editorapi.Parameter, name string, kwOnlyArg bool, expectSynopis bool, expectInferredValues bool, defaultValue string) {
	require.NotNil(t, p)
	assert.EqualValues(t, name, p.Name)
	require.NotNil(t, p.LanguageDetails)
	require.NotNil(t, p.LanguageDetails.Python)
	assert.EqualValues(t, kwOnlyArg, p.LanguageDetails.Python.KeywordOnly)

	if expectSynopis {
		assert.NotEmpty(t, p.Synopsis)
	} else {
		assert.Empty(t, p.Synopsis)
	}

	if expectInferredValues {
		assert.NotEmpty(t, p.InferredValue)
	} else {
		assert.Empty(t, p.InferredValue)
	}

	if defaultValue == "" {
		assert.Empty(t, p.LanguageDetails.Python.DefaultValue)
	} else {
		require.NotEmpty(t, p.LanguageDetails.Python.DefaultValue)
		assert.EqualValues(t, defaultValue, p.LanguageDetails.Python.DefaultValue[0].Repr)
	}
}
