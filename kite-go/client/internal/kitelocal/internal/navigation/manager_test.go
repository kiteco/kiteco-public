package navigation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/client/sidebar"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/navigation/codebase"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-golib/wstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/websocket"
)

var (
	github     = filepath.Join(os.Getenv("GOPATH"), "src", "github.com")
	nongit     = filepath.Join(github, "kiteco", "foo.py")
	kiteco     = filepath.Join(github, "kiteco", "kiteco")
	testDir    = filepath.Join(kiteco, "kite-go", "navigation", "offline", "testdata")
	astPath    = filepath.Join(testDir, "astgo.py")
	parserPath = filepath.Join(testDir, "parsergo.py")
)

func TestMain(m *testing.M) {
	clienttelemetry.Disable()
	os.Exit(m.Run())
}

func Test_Component(t *testing.T) {
	m, err := newManager(codebase.Options{ComputedCommitsLimit: 100})
	require.NoError(t, err)

	component.TestImplements(t, m, component.Implements{
		Initializer:      true,
		Handlers:         true,
		ProcessedEventer: true,
		Terminater:       true,
	})
}

func Test_MultipleWithIndexingFinished(t *testing.T) {
	r, m := makeTestRouterManager(t, defaultUnloadInterval)
	sidebar.TestInit(&sidebar.TestController{
		StartReturns: errors.New("Do not wait for ws connection"),
	})
	// This takes a while, so we reuse the resources for subsequent tests
	requireIndexingDone(t, m)
	t.Run("Test_HandleDecorationLine", THandleDecorationLine(r, m))
	t.Run("Test_HandleRequestRelated", THandleRequestRelated(r, m))
	t.Run("Test_HandleFetchRelated", THandleFetchRelated(r, m))
	testNavigate(t, m)
}

func THandleDecorationLine(r *mux.Router, m *Manager) func(*testing.T) {
	return func(t *testing.T) {
		rec := httptest.NewRecorder()
		b, err := json.Marshal(map[string]string{"filename": astPath})
		require.NoError(t, err)
		req := httptest.NewRequest("POST", "/codenav/decoration/line", bytes.NewBuffer(b))
		r.ServeHTTP(rec, req)

		respb, err := ioutil.ReadAll(rec.Result().Body)
		require.NoError(t, err)
		var jsonresp map[string]interface{}
		err = json.Unmarshal(respb, &jsonresp)
		require.NoError(t, err)

		assert.EqualValues(t, http.StatusOK, rec.Code)
		assert.Contains(t, jsonresp, "project_ready")
		assert.Contains(t, jsonresp, "inline_message")
		assert.Contains(t, jsonresp, "hover_message")
		assert.True(t, jsonresp["project_ready"].(bool))
		assert.EqualValues(t, jsonresp["inline_message"], "Find related code in kiteco")
		assert.EqualValues(t, jsonresp["hover_message"], "Search for code in kiteco which may be related to this line")

		rec = httptest.NewRecorder()
		b = json.RawMessage(`{"filename": invalidjson}`)
		req = httptest.NewRequest("POST", "/codenav/decoration/line", bytes.NewBuffer(b))
		r.ServeHTTP(rec, req)
		assert.EqualValues(t, http.StatusBadRequest, rec.Code, "projectinfo should 400 for malformed json")
	}
}

func THandleRequestRelated(r *mux.Router, m *Manager) func(*testing.T) {
	return func(t *testing.T) {
		/* Copilot connects to /codenav/subscribe websocket */
		s := httptest.NewServer(r)
		defer s.Close()

		serverURL, err := url.Parse(s.URL)
		require.NoError(t, err)
		serverURL.Scheme = "ws"
		serverURL.Path = "/codenav/subscribe"

		wsclient, err := websocket.Dial(serverURL.String(), "", "http://localhost")
		require.NoError(t, err)
		defer wsclient.Close()

		wstest.WaitForConnections(t, m.ws, 1)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		wscdata := make(chan string)
		go wstest.ReadFromWS(ctx, wsclient, wscdata)

		/* Editor initiates a request for related files */
		rec := httptest.NewRecorder()
		b, err := json.Marshal(map[string]interface{}{
			"location": map[string]string{
				"filename": astPath,
			},
			"editor": "vim",
		})
		require.NoError(t, err)
		req := httptest.NewRequest("POST", "/codenav/editor/related", bytes.NewBuffer(b))
		r.ServeHTTP(rec, req)
		assert.EqualValues(t, http.StatusOK, rec.Code, "should 200 if indexing was successful")

		/* Copilot should receive a push message */
		var msg codenavPushMsg
		select {
		case <-ctx.Done():
			require.Fail(t, "did not receive expected message from websocket connection")
		case payload := <-wscdata:
			err := json.Unmarshal([]byte(payload), &msg)
			require.NoError(t, err, "/codenav/subscribe should respend with a codeNavPushMsg")
		}
		rel, err := filepath.Rel(kiteco, astPath)
		require.NoError(t, err)
		relPath, filename := filepath.Split(rel)
		assert.EqualValues(t, relPath, msg.RelPath, "relative file path should be correct")
		assert.EqualValues(t, filename, msg.Filename, "filename should be correct")

		// Cleanup
		assert.EqualValues(t, 1, len(m.ws.ActiveConnections()))
		err = wsclient.Close()
		assert.NoError(t, err)

		// allow the connection to close
		time.Sleep(100 * time.Millisecond)
		m.ws.CloseConnections()
		assert.EqualValues(t, 0, len(m.ws.ActiveConnections()))
	}
}

func THandleFetchRelated(r *mux.Router, m *Manager) func(*testing.T) {
	return func(t *testing.T) {
		rec := httptest.NewRecorder()
		b, err := json.Marshal(map[string]interface{}{
			"location": map[string]string{
				"filename": astPath,
			},
			"num_files": 5,
		})
		require.NoError(t, err)
		req := httptest.NewRequest("POST", "/codenav/related", bytes.NewBuffer(b))
		r.ServeHTTP(rec, req)
		var resp fetchRelatedResponse
		b, er := ioutil.ReadAll(rec.Result().Body)
		require.NoError(t, er, "")
		err = json.Unmarshal(b, &resp)

		assert.EqualValues(t, http.StatusOK, rec.Code, "should 200 if indexing was successful")
		require.NoError(t, err, "/codenav/related should respond with a fetchRelatedResponse")
		rel, err := filepath.Rel(kiteco, astPath)
		require.NoError(t, err)
		relPath, filename := filepath.Split(rel)
		assert.EqualValues(t, relPath, resp.RelPath, "relative file path should be correct")
		assert.EqualValues(t, filename, resp.Filename, "filename should be correct")
		assert.NotEqual(t, resp.ProjRoot, "", "project root should be non-empty")
		assert.False(t, strings.HasPrefix(resp.RelatedFiles[0].RelPath, resp.ProjRoot), "RelatedFiles' RelPath should be relative")
		assert.True(t, strings.HasPrefix(resp.RelatedFiles[0].File.Path, resp.ProjRoot), "RelatedFiles' File path should be absolute")
	}
}

type errTest struct {
	path  string
	data  map[string]interface{}
	bytes []byte
}

func requireMakeErrTests(t *testing.T, filename string) []errTest {
	tests := []errTest{
		{
			path: "/codenav/decoration/line",
			data: map[string]interface{}{
				"filename": filename,
			},
		},
		{
			path: "/codenav/editor/related",
			data: map[string]interface{}{
				"location": map[string]interface{}{
					"filename": filename,
				},
				"editor": "vim",
			},
		},
		{
			path: "/codenav/related",
			data: map[string]interface{}{
				"location": map[string]interface{}{
					"filename": filename,
				},
				"num_files": 5,
				"offset":    0,
			},
		},
		{
			path: "/codenav/related",
			data: map[string]interface{}{
				"location": map[string]interface{}{
					"filename": filename,
				},
				"num_files": 5,
				"offset":    1,
			},
		},
	}

	for i, test := range tests {
		b, err := json.Marshal(test.data)
		require.NoError(t, err)
		tests[i].bytes = b
	}

	return tests
}

func Test_ErrPathNotInSupportedProject(t *testing.T) {
	r, _ := makeTestRouterManager(t, defaultUnloadInterval)

	tests := requireMakeErrTests(t, nongit)

	for _, test := range tests {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", test.path, bytes.NewBuffer(test.bytes))
		r.ServeHTTP(rec, req)

		switch test.path {
		case "/codenav/decoration/line":
			require.EqualValues(t, http.StatusOK, rec.Code, "should 200 even if project is not loaded", test.path)

			var jsonresp map[string]interface{}
			respb, err := ioutil.ReadAll(rec.Result().Body)
			require.NoError(t, err)
			err = json.Unmarshal(respb, &jsonresp)
			require.NoError(t, err)

			require.NotContains(t, jsonresp, "project_ready")
			require.Contains(t, jsonresp, "inline_message")
			require.Contains(t, jsonresp, "hover_message")

			assert.EqualValues(t, jsonresp["inline_message"], "")
			assert.EqualValues(t, jsonresp["hover_message"], "")
		case "/codenav/editor/related":
			assert.EqualValues(t, http.StatusMethodNotAllowed, rec.Code, "should 405 if not in git repo", test.path, nongit)
			var resp map[string]string
			require.NoError(t, json.NewDecoder(rec.Result().Body).Decode(&resp))
			assert.NotEmpty(t, resp["message"])
		case "/codenav/related":
			assert.EqualValues(t, http.StatusMethodNotAllowed, rec.Code, "should 405 if not in git repo", test.path, nongit)
			respb, err := ioutil.ReadAll(rec.Result().Body)
			require.NoError(t, err)
			assert.EqualValues(t, codebase.ErrPathNotInSupportedProject.Error(), strings.Trim(string(respb), "\n"), test.path, nongit)
		}
	}
}

func Test_ErrProjectNotLoaded(t *testing.T) {
	tests := requireMakeErrTests(t, parserPath)

	for _, test := range tests {
		r, m := makeTestRouterManager(t, defaultUnloadInterval)
		rec := httptest.NewRecorder()

		// Request comes in before any editor events in the project
		req := httptest.NewRequest("POST", test.path, bytes.NewBuffer(test.bytes))
		r.ServeHTTP(rec, req)

		switch test.path {
		case "/codenav/decoration/line":
			require.EqualValues(t, http.StatusOK, rec.Code, "should 200 even if project is not loaded", test.path)

			var jsonresp map[string]interface{}
			respb, err := ioutil.ReadAll(rec.Result().Body)
			require.NoError(t, err)
			err = json.Unmarshal(respb, &jsonresp)
			require.NoError(t, err)

			time.Sleep(200 * time.Millisecond)
			_, _, err = m.navigator.ProjectInfo(parserPath)

			// Path /codenav/projectinfo does not have auto-load behavior
			require.Error(t, err, "indexing should not have been started automatically")
			require.Contains(t, jsonresp, "project_ready")
			require.Contains(t, jsonresp, "inline_message")
			require.Contains(t, jsonresp, "hover_message")

			assert.False(t, jsonresp["project_ready"].(bool))
			assert.EqualValues(t, jsonresp["inline_message"], "")
			assert.EqualValues(t, jsonresp["hover_message"], "")
		case "/codenav/editor/related":
			require.EqualValues(t, http.StatusServiceUnavailable, rec.Code, "should 503 if project is not loaded", test.path)

			time.Sleep(200 * time.Millisecond)
			_, _, err := m.navigator.ProjectInfo(parserPath)

			require.NoError(t, err, "indexing should have been started automatically", parserPath)

			var resp map[string]string
			require.NoError(t, json.NewDecoder(rec.Result().Body).Decode(&resp))
			assert.NotEmpty(t, resp["message"])
		case "/codenav/related":
			if test.data["offset"] == 0 {
				require.EqualValues(t, http.StatusOK, rec.Code, "should load project and 200 if project is not loaded", test.path)
				status, _, err := m.navigator.ProjectInfo(parserPath)
				require.NoError(t, err)
				require.EqualValues(t, codebase.Active, status, "codebase should be active")
			} else {
				require.EqualValues(t, http.StatusMethodNotAllowed, rec.Code, "should 405 if project is not loaded", test.path)
				respb, err := ioutil.ReadAll(rec.Result().Body)
				require.NoError(t, err)

				time.Sleep(200 * time.Millisecond)
				status, _, err := m.navigator.ProjectInfo(parserPath)
				require.NoError(t, err)

				// Loads in the background, but responds that it was not loaded
				assert.EqualValues(t, codebase.InProgress, status, parserPath)
				assert.EqualValues(t, codebase.ErrProjectNotLoaded.Error(), strings.Trim(string(respb), "\n"), test.path)
			}
		}
	}
}

func Test_ErrProjectStillIndexing(t *testing.T) {
	tests := requireMakeErrTests(t, parserPath)

	for _, test := range tests {
		r, m := makeTestRouterManager(t, defaultUnloadInterval)
		rec := httptest.NewRecorder()

		// Request comes in immediately after an edit
		m.ProcessedEvent(nil, &component.EditorEvent{Filename: parserPath})
		time.Sleep(time.Second)
		req := httptest.NewRequest("POST", test.path, bytes.NewBuffer(test.bytes))
		r.ServeHTTP(rec, req)
		status, _, err := m.navigator.ProjectInfo(parserPath)
		require.NoError(t, err)

		switch test.path {
		case "/codenav/decoration/line":
			require.EqualValues(t, codebase.InProgress, status, "indexing should still be in progress for subsequent test")
			require.EqualValues(t, http.StatusOK, rec.Code, "should 200 even if project is not loaded", test.path)

			var jsonresp map[string]interface{}
			respb, err := ioutil.ReadAll(rec.Result().Body)
			require.NoError(t, err)
			err = json.Unmarshal(respb, &jsonresp)
			require.NoError(t, err)

			require.Contains(t, jsonresp, "project_ready")
			require.Contains(t, jsonresp, "inline_message")
			require.Contains(t, jsonresp, "hover_message")

			assert.False(t, jsonresp["project_ready"].(bool))
			assert.EqualValues(t, jsonresp["inline_message"], "")
			assert.EqualValues(t, jsonresp["hover_message"], "")
		case "/codenav/editor/related":
			require.EqualValues(t, codebase.InProgress, status, "indexing should still be in progress for subsequent test")
			require.EqualValues(t, http.StatusServiceUnavailable, rec.Code, "should 503 if still indexing")

			var resp map[string]string
			require.NoError(t, json.NewDecoder(rec.Result().Body).Decode(&resp))
			assert.NotEmpty(t, resp["message"])
		case "/codenav/related":
			if test.data["offset"] != 0 {
				require.EqualValues(t, codebase.InProgress, status, "indexing should still be in progress for subsequent test")
				require.EqualValues(t, http.StatusMethodNotAllowed, rec.Code, "should 405 if still indexing")
				respb, err := ioutil.ReadAll(rec.Result().Body)
				require.NoError(t, err)
				assert.EqualValues(t, codebase.ErrProjectStillIndexing.Error(), strings.Trim(string(respb), "\n"))
			} else {
				require.EqualValues(t, codebase.Active, status, "requests to /codenav/related with 0 offset should block until indexing is done")
				require.EqualValues(t, http.StatusOK, rec.Code)
			}
		}
	}
}

func TestProcessedEventAndTerminate(t *testing.T) {
	_, m := makeTestRouterManager(t, defaultUnloadInterval)
	m.ProcessedEvent(nil, &component.EditorEvent{Filename: astPath})
	time.Sleep(time.Second)
	m.Terminate()
	time.Sleep(time.Second)
	_, _, err := m.navigator.ProjectInfo(astPath)
	require.Equal(t, codebase.ErrProjectNotLoaded, err)
	require.Equal(t, astPath, m.activePath)
}

func TestManagerUnload(t *testing.T) {
	_, m := makeTestRouterManager(t, 5*time.Second)
	requireIndexingDone(t, m)
	m.ProcessedEvent(nil, &component.EditorEvent{Filename: astPath})
	time.Sleep(10 * time.Second)
	_, _, err := m.navigator.ProjectInfo(astPath)

	require.Equal(t, err, codebase.ErrProjectNotLoaded)
}

func TestCachedBufferUsed(t *testing.T) {
	_, m := makeTestRouterManager(t, defaultUnloadInterval)

	// Request without cached buffer
	recreq := m.toRecommendRequest(&fetchRelatedRequest{
		Location: recommend.Location{
			CurrentPath: "/src/test.py",
		},
	})
	assert.EqualValues(t, "", m.bufcache.contents)
	assert.Nil(t, recreq.BufferContents,
		"When no buffer has been cached, calls Navigate with nil BufferContents")

	// Request with cached buffer
	bufferContents := "import os\nimport heapq"
	m.ProcessedEvent(&event.Event{}, &component.EditorEvent{
		Filename: "/src/test.py",
		Text:     bufferContents,
	})
	recreq = m.toRecommendRequest(&fetchRelatedRequest{
		Location: recommend.Location{
			CurrentPath: "/src/test.py",
		},
	})
	assert.EqualValues(t, bufferContents, m.bufcache.contents)
	assert.EqualValues(t, bufferContents, recreq.BufferContents,
		"When buffer has been cached, calls Navigate with cached contents")

	// Request with cached buffer for different file
	recreq = m.toRecommendRequest(&fetchRelatedRequest{
		Location: recommend.Location{
			CurrentPath: "/src/not/match.js",
		},
	})
	assert.EqualValues(t, bufferContents, m.bufcache.contents)
	assert.Nil(t, recreq.BufferContents,
		"When request is for different file than cached, calls Navigate with nil BufferContents")
}

func makeTestRouterManager(t *testing.T, unloadInterval time.Duration) (*mux.Router, *Manager) {
	r := mux.NewRouter()
	m, err := newManager(codebase.Options{ComputedCommitsLimit: 100})
	require.NoError(t, err)
	m.unloadInterval = unloadInterval
	s := settings.NewTestManager()
	m.Initialize(component.InitializerOptions{Settings: s})
	m.cohort = component.MockCohortManager{}
	m.RegisterHandlers(r)
	return r, m
}

func requireIndexingDone(t *testing.T, m *Manager) {
	m.ProcessedEvent(nil, &component.EditorEvent{Filename: parserPath})
	for {
		time.Sleep(time.Second)
		status, _, err := m.navigator.ProjectInfo(parserPath)
		if err == nil && status != codebase.InProgress {
			return
		}
	}
}

type navigateTC struct {
	currentPath  string
	expectedPath string
}

func testNavigate(t *testing.T, m *Manager) {
	tcs := []navigateTC{
		navigateTC{
			currentPath:  astPath,
			expectedPath: parserPath,
		},
		navigateTC{
			currentPath:  strings.Replace(astPath, "C:", "c:", 1),
			expectedPath: parserPath,
		},
		navigateTC{
			currentPath:  strings.Replace(astPath, "c:", "C:", 1),
			expectedPath: parserPath,
		},
	}

	for _, tc := range tcs {
		iter, err := m.navigator.Navigate(recommend.Request{
			Location: recommend.Location{
				CurrentPath: tc.currentPath,
			},
			MaxFileRecs:      defaultMaxFileRecs,
			MaxBlockRecs:     defaultMaxBlockRecs,
			MaxFileKeywords:  defaultMaxFileKeywords,
			MaxBlockKeywords: defaultMaxBlockKeywords,
		})
		require.NoError(t, err)
		files, err := iter.Next(5)
		require.NoError(t, err)
		var paths []string
		for _, file := range files {
			paths = append(paths, file.Path)
		}
		status, _, err := m.navigator.ProjectInfo(tc.currentPath)

		require.NoError(t, err)
		require.Equal(t, 5, len(files))
		require.Equal(t, codebase.Active, status)
		require.Contains(t, paths, tc.expectedPath)
		require.Equal(t, defaultMaxBlockRecs, len(files[0].Blocks))
		for _, file := range files {
			require.NotZero(t, len(file.Blocks))
			require.NotZero(t, len(file.Keywords))
			for _, block := range file.Blocks {
				require.NotZero(t, len(block.Keywords))
			}
		}
	}
}
