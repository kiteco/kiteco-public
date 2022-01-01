package status

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/auth"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/navigation/codebase"
	"github.com/kiteco/kiteco/kite-go/response"
	constants "github.com/kiteco/kiteco/kite-golib/conversion"
	"github.com/kiteco/kiteco/kite-golib/enginestatus"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Component(t *testing.T) {
	m := NewTestManager()
	component.TestImplements(t, m, component.Implements{
		Initializer:    true,
		Handlers:       true,
		EventResponser: true,
	})
}

func Test_Status(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user1@example.com": "password1"})
	require.NoError(t, err)
	defer s.Close()

	statusMgr := NewTestManager()
	statusMgr.SetNav(mockNav{returnedErr: nil})
	statusMgr.models = &mockIsLoaded{returns: true}
	err = auth.SetupWithAuthDefaults(s, statusMgr)
	require.NoError(t, err)

	// set up files for status checks
	file := filepath.Join(s.BasePath, "file.py")
	err = ioutil.WriteFile(file, []byte(""), 0666)
	require.NoError(t, err)
	defer os.Remove(file)

	jsfile := filepath.Join(s.BasePath, "jsfile.js")
	err = ioutil.WriteFile(jsfile, []byte(""), 0666)
	require.NoError(t, err)
	defer os.Remove(jsfile)

	gofile := filepath.Join(s.BasePath, "gofile.go")
	err = ioutil.WriteFile(gofile, []byte(""), 0666)
	require.NoError(t, err)
	defer os.Remove(gofile)

	// status for a file without a user must return a 200 response
	resp, err := s.DoKitedGet("/clientapi/status?filename=" + file)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// login
	s.SendLoginRequest("user1@example.com", "password1", true)

	var sResp enginestatus.Response

	// indexing
	resp, err = s.DoKitedGet("/clientapi/status?filename=" + file)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&sResp))
	assert.Equal(t, sResp.Status, "indexing")

	// unsupported
	resp, err = s.DoKitedGet("/clientapi/status?filename=" + filepath.Join(s.BasePath, "file.txt"))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&sResp))
	assert.Equal(t, sResp.Status, "unsupported")

	resp, err = s.DoKitedGet("/clientapi/status?filetype=javascript")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&sResp))
	assert.Equal(t, sResp.Status, "unsupported")

	// noIndex file (unsaved)
	resp, err = s.DoKitedGet("/clientapi/status?filetype=python")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&sResp))
	assert.Equal(t, sResp.Status, "noIndex")

	resp, err = s.DoKitedGet("/clientapi/status?filename=" + url.QueryEscape(filepath.Join(s.BasePath, "unsaved.py")))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&sResp))
	assert.Equal(t, sResp.Status, "noIndex")

	statusMgr.models = &mockIsLoaded{returns: false}
	resp, err = s.DoKitedGet("/clientapi/status?filename=" + file)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&sResp))
	assert.Equal(t, "initializing", sResp.Status)

	// indexing (codenav)
	statusMgr.SetNav(mockNav{returnedErr: codebase.ErrProjectStillIndexing})
	resp, err = s.DoKitedGet("/clientapi/status?checkloaded=false&filename=" + file)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&sResp))
	assert.Equal(t, sResp.Status, "indexing")
}

func Test_PaywallAllFeatProStatus(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user1@example.com": "password1"})
	require.NoError(t, err)
	defer s.Close()

	file := filepath.Join(s.BasePath, "file.py")
	err = ioutil.WriteFile(file, []byte(""), 0666)
	require.NoError(t, err)
	defer os.Remove(file)

	var sResp enginestatus.Response
	statusMgr := NewTestManager()
	err = auth.SetupWithAuthDefaults(s, statusMgr)
	require.NoError(t, err)

	statusMgr.license = &licensing.MockLicense{
		Plan:    licensing.FreePlan,
		Product: licensing.Free,
	}
	statusMgr.cohort = component.MockCohortManager{
		Convcohort: constants.UsagePaywall,
	}
	statusMgr.settings.SetBool(settings.AllFeaturesPro, false)
	statusMgr.settings.Set(settings.PaywallCompletionsRemaining, "1")
	resp, err := s.DoKitedGet("/clientapi/status?checkloaded=false&filename=" + file)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&sResp))
	assert.False(t, strings.HasSuffix(sResp.Status, "(1 completion left today)"), "Only usage-paywall with all_features_pro should see completions left")
	assert.False(t, strings.HasSuffix(sResp.Short, "(1 completion left today)"), "Only usage-paywall with all_features_pro should see completions left")

	statusMgr.settings.SetBool(settings.AllFeaturesPro, true)

	statusMgr.settings.Set(settings.PaywallCompletionsRemaining, "0")
	resp, err = s.DoKitedGet("/clientapi/status?checkloaded=false&filename=" + file)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&sResp))
	assert.Equal(t, "locked (upgrade to Pro to unlock)", sResp.Status)
	assert.Equal(t, "locked (upgrade to Pro to unlock)", sResp.Short)

	statusMgr.settings.Set(settings.PaywallCompletionsRemaining, "1")
	resp, err = s.DoKitedGet("/clientapi/status?checkloaded=false&filename=" + file)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&sResp))
	assert.True(t, strings.HasSuffix(sResp.Status, "(1 completion left today)"))
	assert.True(t, strings.HasSuffix(sResp.Short, "(1 completion left today)"))

	statusMgr.settings.Set(settings.PaywallCompletionsRemaining, "2")
	resp, err = s.DoKitedGet("/clientapi/status?checkloaded=false&filename=" + file)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&sResp))
	assert.True(t, strings.HasSuffix(sResp.Status, "(2 completions left today)"))
	assert.True(t, strings.HasSuffix(sResp.Short, "(2 completions left today)"))

	statusMgr.license = &licensing.MockLicense{
		Plan:    licensing.ProYearly,
		Product: licensing.Pro,
	}
	resp, err = s.DoKitedGet("/clientapi/status?checkloaded=false&filename=" + file)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&sResp))
	assert.False(t, strings.HasSuffix(sResp.Status, "(2 completions left today)"), "Pro license owners should not see completions left")
	assert.False(t, strings.HasSuffix(sResp.Short, "(2 completions left today)"), "Pro license owners should not see completions left")
}

func Test_StatusInvalidPath(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user1@example.com": "password1"})
	require.NoError(t, err)
	defer s.Close()

	statusMgr := NewTestManager()
	auth.SetupWithAuthDefaults(s, statusMgr)

	resp, err := s.DoKitedGet("/clientapi/status?filename=home")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	//login
	s.SendLoginRequest("user1@example.com", "password1", true)

	//check invalid path
	resp, err = s.DoKitedGet("/clientapi/status?filename=home.py")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	//check empty path
	resp, err = s.DoKitedGet("/clientapi/status?filename=")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func Test_LocalStatusUpdate(t *testing.T) {
	statusMgr := NewTestManager()
	assert.EqualValues(t, localcode.StatusResponse{}, statusMgr.localCodeStatus)

	// make sure nil response doesn't cause issues
	assert.NotPanics(t, func() {
		statusMgr.EventResponse(nil)
	})

	// make sure nil localcode.StatusResponse doesn't cause issues
	assert.NotPanics(t, func() {
		r := &response.Root{}
		statusMgr.EventResponse(r)
	})

	// make sure status updates correctly via EventResponse

	assert.EqualValues(t, localcode.StatusResponse{}, statusMgr.localCodeStatus)
	expected := localcode.StatusResponse{
		Indices: []localcode.IndexResponse{
			{Path: "foo"},
		},
	}

	r := &response.Root{
		LocalIndexStatus: &expected,
	}
	statusMgr.EventResponse(r)
	assert.EqualValues(t, expected, statusMgr.localCodeStatus)
}

type mockIsLoaded struct {
	returns bool
}

// IsLoaded mocks whether a model is loaded
func (m *mockIsLoaded) IsLoaded(fext string) bool {
	return m.returns
}

type mockNav struct {
	returnedErr error
}

func (m mockNav) Validate(path string) error {
	return m.returnedErr
}
