package autocorrect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/auth"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Component(t *testing.T) {
	m := NewManager(1024, true)

	component.TestImplements(t, m, component.Implements{
		Initializer: true,
		Handlers:    true,
	})
}

func Test_Metrics(t *testing.T) {
	testClientProxy(t, "/clientapi/editor/autocorrect/metrics", "/api/editor/autocorrect/metrics")
	testClientTimeout(t, "/clientapi/editor/autocorrect/metrics", "/api/editor/autocorrect/metrics")
}

func Test_Feedback(t *testing.T) {
	testClientProxy(t, "/clientapi/editor/autocorrect/feedback", "/api/editor/autocorrect/feedback")
	testClientTimeout(t, "/clientapi/editor/autocorrect/feedback", "/api/editor/autocorrect/feedback")
}

func Test_OnSave(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	mgr := newTestManager(1024, true, time.Second, time.Second)
	auth.SetupWithAuthDefaults(s, mgr)

	s.Backend.AddPrefixRequestHandler("/api/editor/autocorrect/validation/on-save", []string{"POST"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	data := requestJSON(s.GetFilePath("test.py"), "content", "")
	resp, err := s.DoKitedPost("/clientapi/editor/autocorrect/validation/on-save", data)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusOK, resp.StatusCode, "Response: "+readResponse(resp))

	//bad json
	resp, err = s.DoKitedPost("/clientapi/editor/autocorrect/validation/on-save", strings.NewReader("not-json"))
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode, "Response: "+readResponse(resp))
}

func Test_Autocorrect(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	mgr := newTestManager(1024, false, time.Second, time.Second)
	auth.SetupWithAuthDefaults(s, mgr)

	s.Backend.AddPrefixRequestHandler("/api/editor/autocorrect", []string{"POST"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	atomWhitelisted := requestJSON(s.GetFilePath("test.py"), "content", "atom")

	//atom, whitelisted
	resp, err := s.DoKitedPost("/clientapi/editor/autocorrect", atomWhitelisted)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	//invalid JSON
	resp, err = s.DoKitedPost("/clientapi/editor/autocorrect", strings.NewReader("not-json"))
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
}

func Test_AutocorrectSupportedEditors(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	mgr := newTestManager(1024, false, time.Second, time.Second)
	auth.SetupWithAuthDefaults(s, mgr)

	s.Backend.AddPrefixRequestHandler("/api/editor/autocorrect", []string{"POST"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// atom, enabled
	atom := requestJSON(s.GetFilePath("test.py"), "content", "atom")
	resp, err := s.DoKitedPost("/clientapi/editor/autocorrect", atom)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	// vscode, enabled
	vscode := requestJSON(s.GetFilePath("test.py"), "content", "vscode")
	resp, err = s.DoKitedPost("/clientapi/editor/autocorrect", vscode)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	// vim, disabled
	vim := requestJSON(s.GetFilePath("test.py"), "content", "vim")
	resp, err = s.DoKitedPost("/clientapi/editor/autocorrect", vim)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusNotImplemented, resp.StatusCode)

	// intellij, disabled
	intellij := requestJSON(s.GetFilePath("test.py"), "content", "intellij")
	resp, err = s.DoKitedPost("/clientapi/editor/autocorrect", intellij)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusNotImplemented, resp.StatusCode)

	// sublime3, disabled
	sublime3 := requestJSON(s.GetFilePath("test.py"), "content", "sublime3")
	resp, err = s.DoKitedPost("/clientapi/editor/autocorrect", sublime3)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusNotImplemented, resp.StatusCode)
}

func Test_AutocorrectDebugEnabled(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	mgr := newTestManager(1024, true, time.Second, time.Second)
	auth.SetupWithAuthDefaults(s, mgr)

	s.Backend.AddPrefixRequestHandler("/api/editor/autocorrect", []string{"POST"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	atomWhitelisted := requestJSON(s.GetFilePath("test.py"), "content", "atom")
	otherWhitelisted := requestJSON(s.GetFilePath("test.py"), "content", "vim")

	//atom
	resp, err := s.DoKitedPost("/clientapi/editor/autocorrect", atomWhitelisted)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	//other editor
	resp, err = s.DoKitedPost("/clientapi/editor/autocorrect", otherWhitelisted)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode, "Expected request error for unsupported editor")

	//invalid JSON
	resp, err = s.DoKitedPost("/clientapi/editor/autocorrect", strings.NewReader("not-json"))
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusBadRequest, resp.StatusCode)
}

func Test_OnSaveTimeout(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	mgr := newTestManager(1024, true, time.Second, time.Second)
	auth.SetupWithAuthDefaults(s, mgr)

	s.Backend.AddPrefixRequestHandler("/api/editor/autocorrect/validation/on-save", []string{"POST"}, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(mgr.defaultTimeout + time.Millisecond*250)
	})

	data := requestJSON(s.GetFilePath("test.py"), "content", "atom")
	resp, err := s.DoKitedPost("/clientapi/editor/autocorrect/validation/on-save", data)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusRequestTimeout, resp.StatusCode)
}

func Test_Validation(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	mgr := newTestManager(1024, true, time.Second, time.Second)
	auth.SetupWithAuthDefaults(s, mgr)

	small := strings.Repeat("a", 512)
	max := strings.Repeat("a", 512)
	tooLarge := strings.Repeat("a", 1025)

	//whitelisted
	status, err := mgr.validate(s.GetFilePath("test.py"), small)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, status)

	status, err = mgr.validate(s.GetFilePath("test.py"), max)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, status)

	status, err = mgr.validate(s.GetFilePath("test.py"), tooLarge)
	assert.Error(t, err)
	assert.EqualValues(t, http.StatusRequestEntityTooLarge, status)

	//whitelisted, not a python file
	status, err = mgr.validate(s.GetFilePath("test.txt"), small)
	assert.Error(t, err)
	assert.EqualValues(t, http.StatusForbidden, status)
}

func testClientProxy(t *testing.T, clientURL, backendURL string) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	mgr := newTestManager(1024, true, time.Second, time.Second)
	auth.SetupWithAuthDefaults(s, mgr)

	//this dummy backend method returns 200 for JSON and 500 otherwise
	s.Backend.AddPrefixRequestHandler(backendURL, []string{"POST"}, func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("unable to unmarshal request: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	//post JSON
	resp, err := s.DoKitedPost(clientURL, strings.NewReader("{}"))
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)

	//post non-JSON and make sure that status is proxied
	resp, err = s.DoKitedPost(clientURL, strings.NewReader("not-json"))
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusInternalServerError, resp.StatusCode)
}

func testClientTimeout(t *testing.T, clientURL, backendURL string) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	mgr := newTestManager(1024, true, time.Second, time.Second)
	auth.SetupWithAuthDefaults(s, mgr)

	s.Backend.AddPrefixRequestHandler(backendURL, []string{"POST"}, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(mgr.defaultTimeout + 250*time.Millisecond)
	})

	//post JSON
	resp, err := s.DoKitedPost(clientURL, strings.NewReader("{}"))
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusRequestTimeout, resp.StatusCode)
}

func readResponse(resp *http.Response) string {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	resp.Body.Close()
	return string(body)
}

func requestJSON(filename, buffer, editorName string) io.Reader {
	type metadata struct {
		Source string `json:"source"`
	}
	type data struct {
		metadata `json:"metadata"`
		Filename string `json:"filename"`
		Buffer   string `json:"buffer"`
	}

	t, err := json.Marshal(data{Filename: filename, Buffer: buffer, metadata: metadata{Source: editorName}})
	if err != nil {
		return bytes.NewReader([]byte{})
	}
	return bytes.NewReader(t)
}
