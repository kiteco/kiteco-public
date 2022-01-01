package test

import (
	"crypto/md5"
	"fmt"
	"math"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/clientapp"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_All(t *testing.T) {
	// this test has subtests to share the setup of kitelocal
	// Travis is very slow with this kind of data-intensive tests
	// we want to cut down on runtime and thus walk the extra mile

	// most tests only need BuiltinDistribution3, we're loading NumpyDistribution for the numpy test
	project, err := startKiteLocal(keytypes.BuiltinDistribution3, keytypes.NumpyDistribution)
	require.NoError(t, err)
	defer project.Close()

	run(t, "numpy", project, hoverNumpy)
	run(t, "hover", project, hover)
	run(t, "hoverContentAttached", project, hoverContentAttached)
	run(t, "hoverOutdatedHash", project, hoverOutdatedHash)
	run(t, "hoverNoToken", project, hoverNoToken)
	run(t, "hoverUnavailableFile", project, hoverUnavailableFile)
	run(t, "hoverNoEvents", project, hoverNoEvents)
	run(t, "editorSwitch", project, editorSwitch)
	run(t, "hoverCursorRunes", project, hoverCursorRunes)
	run(t, "hoverCursorBytes", project, hoverCursorBytes)
	run(t, "invalidRequests", project, invalidRequests)
	run(t, "onboardingCompletions", project, onboardingCompletions)

	assert.Empty(t, project.MockTracker.TrackedFilteredByEvent("cta_shown"), "No cta_shown event should be sent in the test env")
}

func hoverNumpy(t *testing.T, project *clientapp.TestEnvironment) {
	r := NewEditorRemoteControl("test_client", project, t)

	r.OpenNewFile("file.py")
	r.Input("import numpy as np")
	hover := r.Hover(false)
	assertHoverBasics(t, hover, "name", "python;;;;;numpy", "numpy")
	assertReportBasics(t, hover.Report)
}

func run(t *testing.T, name string, project *clientapp.TestEnvironment, test func(*testing.T, *clientapp.TestEnvironment)) {
	t.Run(name, func(t *testing.T) {
		test(t, project)
	})
}

func hover(t *testing.T, project *clientapp.TestEnvironment) {
	r := NewEditorRemoteControl("test_client", project, t)
	r.OpenNewFile("file.py")

	r.Input("import json")
	hover := r.Hover(false)
	assertHoverBasics(t, hover, "name", "python;;;;;json", "json")

	r.Input("\nimport math")
	hover = r.Hover(false)
	assertHoverBasics(t, hover, "name", "python;;;;;math", "math")
	assertReportBasics(t, hover.Report)

	r.MoveCursorAbs(7) // on token "json"
	hover = r.Hover(false)
	assertHoverBasics(t, hover, "name", "python;;;;;json", "json")
}

func hoverContentAttached(t *testing.T, project *clientapp.TestEnvironment) {
	r := NewEditorRemoteControl("test_client", project, t)
	r.OpenNewFile("file.py")
	r.Input("import json")

	hover := r.Hover(true)
	assertHoverBasics(t, hover, "name", "python;;;;;json", "json")

	// now send a hover request without a prior matching edit event
	// this must work because we attach the buffer content in the hover request
	// offset 22 is on the 'math' token
	content := "import json\nimport math"
	hash := fmt.Sprintf("%x", md5.Sum([]byte(content)))

	req := mockserver.EditorBufferRequest{
		RequestType:  "hover",
		Editor:       r.Editor,
		File:         r.File,
		Hash:         hash,
		FileContent:  content,
		CursorOffset: 22,
		UseRunes:     true,
	}
	resp, err := project.KitedClient.EditorBufferRequest(req, &hover)
	require.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	assertHoverBasics(t, hover, "name", "python;;;;;math", "math")

	// a mismatch of a previously unused hash and attached content must result in a 200 because the buffer is used in this case
	req.FileContent = "import json\nimport concurrent"
	req.Hash = "invalid-hash"
	resp, err = project.KitedClient.EditorBufferRequest(req, &hover)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	assertHoverBasics(t, hover, "name", "python;;;;;concurrent", "concurrent")
}

func hoverOutdatedHash(t *testing.T, project *clientapp.TestEnvironment) {
	r := NewEditorRemoteControl("test_client", project, t)
	r.OpenNewFile("file.py")
	r.Input("import math")

	oldHash := r.CurrentHash()
	r.Input("\nimport json")

	req := mockserver.EditorBufferRequest{
		RequestType:  "hover",
		Editor:       r.Editor,
		File:         r.File,
		Hash:         oldHash,
		FileContent:  "",
		CursorOffset: r.RunesOffset,
		UseRunes:     !r.UseByteOffsets,
	}

	assertTrueN(t, 5,
		func() bool {
			resp, err := project.KitedClient.EditorBufferRequest(req, nil)
			return err != nil && resp != nil && resp.StatusCode == http.StatusBadRequest
		},
		"current offset if out of bounds for the status identified by the old hash",
	)
}

func hoverNoToken(t *testing.T, project *clientapp.TestEnvironment) {
	r := NewEditorRemoteControl("test_client", project, t)
	r.OpenNewFile("file.py")
	r.Input("import math")
	r.MoveCursorAbs(3) // on 'import'

	_, resp, err := r.HoverError(false)
	assert.Error(t, err)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

func hoverUnavailableFile(t *testing.T, project *clientapp.TestEnvironment) {
	// the file isn't available on disk
	r := NewEditorRemoteControl("test_client", project, t)
	r.OpenNewFile("file.py")
	r.Input("import math")
	r.MoveCursorAbs(3) // on 'import'

	_, resp, err := r.HoverError(false)
	assert.Error(t, err)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

func hoverNoEvents(t *testing.T, project *clientapp.TestEnvironment) {
	// the file isn't available on disk
	r := NewEditorRemoteControl("test_client", project, t)
	r.CreateFile("file.py", "")
	r.OpenFile("file.py")

	_, resp, err := r.HoverError(false)
	assert.Error(t, err)
	require.NotNil(t, resp)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode)
}

// tests that events of an editor is not mixed with other editor states
func editorSwitch(t *testing.T, project *clientapp.TestEnvironment) {
	// the file isn't available on disk
	r := NewEditorRemoteControl("atom", project, t)
	r.OpenNewFile("file.py")
	r.Input("import json")

	hover := r.Hover(false)
	assertHoverBasics(t, hover, "name", "python;;;;;json", "json")

	// switch editor and test again
	r.Editor = "intellij"
	_, resp, err := r.HoverError(false)
	assert.Error(t, err)
	require.NotNil(t, resp)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode, "request with an another editor id must not return results")

	// there has to be a result with attached content
	hover = r.Hover(true)
	assertHoverBasics(t, hover, "name", "python;;;;;json", "json")
}

func hoverCursorRunes(t *testing.T, project *clientapp.TestEnvironment) {
	r := NewEditorRemoteControl("test_client", project, t)
	r.OpenNewFile("file.py")
	r.Input("print(\"你好 你好 你好 你好\")\n")
	r.Input("import json")

	hover := r.Hover(false)
	assertHoverBasics(t, hover, "name", "python;;;;;json", "json")

	hover = r.Hover(true)
	assertHoverBasics(t, hover, "name", "python;;;;;json", "json")

	req := mockserver.EditorBufferRequest{
		RequestType:  "hover",
		Editor:       r.Editor,
		File:         r.File,
		Hash:         r.CurrentHash(),
		FileContent:  "",
		CursorOffset: math.MinInt64,
		UseRunes:     true,
	}
	// testing invalid rune offsets
	_, err := project.KitedClient.EditorBufferRequest(req, nil)
	require.Error(t, err)
	req.CursorOffset = -1
	_, err = project.KitedClient.EditorBufferRequest(req, nil)
	require.Error(t, err)
	req.CursorOffset = 1024
	_, err = project.KitedClient.EditorBufferRequest(req, nil)
	require.Error(t, err)
	req.CursorOffset = math.MinInt64
	_, err = project.KitedClient.EditorBufferRequest(req, nil)
	require.Error(t, err)
}

func hoverCursorBytes(t *testing.T, project *clientapp.TestEnvironment) {
	r := NewEditorRemoteControl("test_client", project, t)
	r.UseByteOffsets = true

	r.OpenNewFile("file.py")
	r.Input("print(\"你好 你好 你好 你好\")\n")
	r.Input("import json")

	// without attached content
	hover := r.Hover(false)
	assertHoverBasics(t, hover, "name", "python;;;;;json", "json")

	// with attached content
	hover = r.Hover(true)
	assertHoverBasics(t, hover, "name", "python;;;;;json", "json")

	req := mockserver.EditorBufferRequest{
		RequestType:  "hover",
		Editor:       r.Editor,
		File:         r.File,
		Hash:         r.CurrentHash(),
		FileContent:  "",
		CursorOffset: math.MinInt64,
		UseRunes:     false,
	}

	// testing invalid byte offsets
	_, err := project.KitedClient.EditorBufferRequest(req, &hover)
	require.Error(t, err)
	req.CursorOffset = -1
	_, err = project.KitedClient.EditorBufferRequest(req, &hover)
	require.Error(t, err)
	req.CursorOffset = 1024
	_, err = project.KitedClient.EditorBufferRequest(req, &hover)
	require.Error(t, err)
	req.CursorOffset = math.MaxInt64
	_, err = project.KitedClient.EditorBufferRequest(req, &hover)
	require.Error(t, err)
}

func invalidRequests(t *testing.T, project *clientapp.TestEnvironment) {
	r := NewEditorRemoteControl("test_client", project, t)
	r.OpenNewFile("file.py")
	r.Input("import math")

	req := mockserver.EditorBufferRequest{
		RequestType:  "hover",
		Editor:       "",
		File:         r.File,
		Hash:         r.CurrentHash(),
		FileContent:  "",
		CursorOffset: math.MaxInt64,
		UseRunes:     false,
	}

	resp, err := project.KitedClient.EditorBufferRequest(req, nil)
	require.Error(t, err)
	require.NotNil(t, resp)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode, "expected an error for empty editor")

	req.Editor = "test_client"
	req.File = ""
	resp, err = project.KitedClient.EditorBufferRequest(req, nil)
	require.Error(t, err)
	require.NotNil(t, resp)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode, "expected an error for empty filename")

	req.File = r.File
	req.Hash = ""
	resp, err = project.KitedClient.EditorBufferRequest(req, nil)
	require.Error(t, err)
	require.NotNil(t, resp)
	assert.EqualValues(t, http.StatusNotFound, resp.StatusCode, "expected an error for empty hash")
}

func onboardingCompletions(t *testing.T, project *clientapp.TestEnvironment) {
	r := NewEditorRemoteControl("atom", project, t)

	// open onboarding file
	var onboardingFile string
	err := project.KitedClient.GetJSON("/clientapi/plugins/onboarding_file?editor="+r.Editor, &onboardingFile)
	require.NoError(t, err)
	defer os.Remove(onboardingFile)

	r.OpenFile(onboardingFile)
	r.MoveCursorAbs(0)
	r.Input("\n")
	r.MoveCursorAbs(0)
	r.Input("import")
	r.Input(" ")
	r.Input("js")
	r.validate()

	time.Sleep(1 * time.Second)
	completions, _, err := r.CompletionsError()
	require.NoError(t, err, "completions expected for the onboarding file, path: %s", onboardingFile)
	require.NotEmpty(t, completions.Completions, "completions expected for the onboarding file, path: %s", onboardingFile)
}

func assertHoverBasics(t *testing.T, hover *editorapi.HoverResponse, partOfSyntax string, id string, name string) {
	assert.EqualValues(t, "python", hover.Language)
	assert.EqualValues(t, partOfSyntax, hover.PartOfSyntax)
	assertReportBasics(t, hover.Report)
	// fixme is . correct as parent id?
	assertSymbolExtBasics(t, hover.Symbol[0], id, name, ".")
}

func assertTrueN(t *testing.T, n int, check func() bool, msg string, objs ...interface{}) {
	for i := 0; i < n; i++ {
		if check() {
			return
		}
		time.Sleep(time.Second)
	}
	assert.Fail(t, msg, objs...)
}
