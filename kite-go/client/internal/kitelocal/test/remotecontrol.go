package test

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/clientapp"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/require"
)

// RemoteControl allows to emulate work in an editor
type RemoteControl struct {
	// state
	RunesOffset int64
	BytesOffset int64
	Content     string
	File        string

	// static

	Editor string
	// if true then bytes offsets will be used instead of runes offsets
	UseByteOffsets bool
	testEnv        *clientapp.TestEnvironment
	client         *mockserver.KitedClient
	t              *testing.T
}

// NewEditorRemoteControl returns a new remote control instance
func NewEditorRemoteControl(editor string, testEnv *clientapp.TestEnvironment, t *testing.T) *RemoteControl {
	return &RemoteControl{
		Editor:      editor,
		Content:     "",
		RunesOffset: 0,
		BytesOffset: 0,
		testEnv:     testEnv,
		client:      testEnv.KitedClient,
		t:           t,
	}
}

// CreateFile creates a new file on disk, it doesn't open it. Use OpenFile() for this.
// file is the relative path of the file, based on the data directory of the test environment
func (r *RemoteControl) CreateFile(file, content string) {
	var absPath string
	if filepath.IsAbs(file) {
		absPath = file
	} else {
		absPath = filepath.Join(r.testEnv.DataDirPath, file)
	}

	_ = os.Remove(absPath)

	err := ioutil.WriteFile(absPath, []byte(content), 0600)
	require.NoError(r.t, err)
}

// OpenFile loads a file from disk and initializes the editor with the new content and an offset of 0
// file is the relative path of the file, based on the data directory of the test environment
func (r *RemoteControl) OpenFile(file string) {
	if filepath.IsAbs(file) {
		r.File = file
	} else {
		r.File = filepath.Join(r.testEnv.DataDirPath, file)
	}

	require.FileExists(r.t, r.File)

	b, err := ioutil.ReadFile(r.File)
	require.NoError(r.t, err)

	r.RunesOffset = 0
	r.BytesOffset = 0
	r.Content = string(b)
	r.validate()
}

// OpenNewFile creates a file with empty content first and then opens it. It also sends a focus event.
// file is the relative path of the file, based on the data directory of the test environment
func (r *RemoteControl) OpenNewFile(file string) {
	r.CreateFile(file, "")
	r.OpenFile(file)
	r.SendFocusEvent()
}

// Save writes the content of the current editor to disk
func (r *RemoteControl) Save() {
	err := ioutil.WriteFile(r.File, []byte(r.Content), 0600)
	require.NoError(r.t, err)
}

// MoveCursor moves the cursor from the current position by a relative offset to a new position
func (r *RemoteControl) MoveCursor(relRuneOffset int64) {
	r.RunesOffset += relRuneOffset
	// r.BytesOffset += relRuneOffset
	r.validate()

	r.SendSelectionEvent()
}

// MoveCursorAbs moves the cursor from the current position by an absolute offset to a new position
func (r *RemoteControl) MoveCursorAbs(relRuneOffset int64) {
	r.RunesOffset = relRuneOffset
	//r.RunesOffset = relRuneOffset
	r.validate()

	r.SendSelectionEvent()
}

// Input enters text at the current offset and sends an edit event
func (r *RemoteControl) Input(typed string) {
	r.Content = r.Content[0:r.BytesOffset] + typed + r.Content[r.BytesOffset:]
	r.RunesOffset += int64(len([]rune(typed)))
	r.BytesOffset += int64(len(typed))
	r.validate()

	r.SendEditEvent()
}

// SendFocusEvent sends a focus event to kited
func (r *RemoteControl) SendFocusEvent() {
	r.validate()
	_, err := r.client.PostFocusEvent(r.Editor, r.File, r.Content, r.RunesOffset)
	r.noError(err)
}

// SendSelectionEvent sends a selection event to kited
func (r *RemoteControl) SendSelectionEvent() {
	_, err := r.client.PostSelectionEvent(r.Editor, r.File, r.Content, r.RunesOffset)
	r.noError(err)
}

// SendEditEvent sends an edit event to kited
func (r *RemoteControl) SendEditEvent() {
	_, err := r.client.PostEditEvent(r.Editor, r.File, r.Content, r.RunesOffset)
	r.noError(err)
}

// Hover requests hover data for the token at the current offset
// it retries a few times to make this work on slower machines
func (r *RemoteControl) Hover(attachContent bool) *editorapi.HoverResponse {
	for i := 0; i < 10; i++ {
		if hover, _, err := r.HoverError(attachContent); err == nil {
			return hover
		}
		time.Sleep(500 * time.Millisecond)
	}

	require.FailNow(r.t, "Hover failed to response with status 200 after several retries")
	return nil
}

// HoverError requests hover data and also returns an error if it occurred
func (r *RemoteControl) HoverError(attachContent bool) (*editorapi.HoverResponse, *http.Response, error) {
	hash := r.CurrentHash()

	content := ""
	if attachContent {
		content = r.Content
	}

	var offset int64
	if r.UseByteOffsets {
		offset = r.BytesOffset
	} else {
		offset = r.RunesOffset
	}

	var hover editorapi.HoverResponse
	req := mockserver.EditorBufferRequest{
		RequestType:  "hover",
		Editor:       r.Editor,
		File:         r.File,
		Hash:         hash,
		FileContent:  content,
		CursorOffset: offset,
		UseRunes:     !r.UseByteOffsets,
	}
	resp, err := r.client.EditorBufferRequest(req, &hover)
	return &hover, resp, err
}

// Completions requests completions data for the token at the current offset
// it retries a few times to make this work on slower machines
func (r *RemoteControl) Completions() *data.APIResponse {
	for i := 0; i < 10; i++ {
		if hover, _, err := r.CompletionsError(); err == nil {
			return hover
		}
		time.Sleep(500 * time.Millisecond)
	}

	require.FailNow(r.t, "Completions failed to response with status 200 after several retries")
	return nil
}

// CompletionsError requests completions data and also returns an error if it occurred
func (r *RemoteControl) CompletionsError() (*data.APIResponse, *http.Response, error) {
	var offset int64
	if r.UseByteOffsets {
		offset = r.BytesOffset
	} else {
		offset = r.RunesOffset
	}

	type position struct {
		Begin int `json:"begin"`
		End   int `json:"end"`
	}

	req := struct {
		Editor   string   `json:"editor"`
		Filename string   `json:"filename"`
		Text     string   `json:"text"`
		Position position `json:"position"`
	}{
		Editor:   r.Editor,
		Filename: r.File,
		Text:     r.Content,
		Position: position{Begin: int(offset), End: int(offset)},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, nil, err
	}

	var completions data.APIResponse
	resp, err := r.client.PostJSON("/clientapi/editor/complete", bytes.NewReader(body), &completions)
	return &completions, resp, err
}

// CurrentHash returns the md5 hash for the current content
func (r *RemoteControl) CurrentHash() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(r.Content)))
}

func (r *RemoteControl) validate() {
	require.NotEmpty(r.t, r.File)
	require.True(r.t, r.RunesOffset >= 0 && r.RunesOffset <= int64(len(r.Content)), fmt.Sprintf("cursor offset outside of allowed range: %d, content: %s", r.RunesOffset, r.Content))
}

// stops execution when t was passed to NewEditorRemoteControl and err is non-nil
func (r *RemoteControl) noError(err error) {
	require.NoError(r.t, err)
}
