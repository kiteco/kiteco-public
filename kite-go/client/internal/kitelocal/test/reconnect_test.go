package test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/clientapp"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
	"github.com/stretchr/testify/require"
)

const editor = "client_test"

func Test_Reconnect(t *testing.T) {
	// project without login
	// a failing component must not break the main loop
	p, err := startKiteLocal(keytypes.BuiltinDistribution3)
	require.NoError(t, err)
	defer p.Close()

	// Allow time for the main loop to start. If something went wrong, the checks below will trigger a failure
	// startKiteLocal is already calling WaitForReady()
	require.EqualValues(t, p.Backend.URL, p.Kited.AuthClient.Target(), "kited's loop must update the target URL")
	require.EqualValues(t, 1, p.MockUpdater.GetCheckedCount(), "kited's loop must check for updates. Requests: \n%s", p.Backend.CountDebugString())

	// #6622: disconnect a few times and make sure that completions are still working after that
	server := p.Backend.URL.String()
	connected := p.Kited.Connected()
	require.True(t, connected, "kited must be connected")

	file := p.Files[0]
	content := ""
	assertEditEvent(t, file, content, p)

	for i := 0; i < 5; i++ {
		p.Kited.Disconnect()
		err = p.WaitForNotReady(10 * time.Second)
		require.NoError(t, err)

		go p.Kited.Connect(server)

		// wait until it's connected again
		err = p.WaitForReady(10 * time.Second)
		require.NoError(t, err, "client must reconnect")

		content += "\nimport j"
		assertEditEvent(t, file, content, p)
	}

	assertEditEvent(t, file, content, p)

	content += "\nimport j"
	assertEditEvent(t, file, content, p)

	for n := 0; n < 10; n++ {
		if ok, _ := assertCompletions(file, content, p); ok {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	ok, err := assertCompletions(file, content, p)
	require.NoError(t, err, "error handling the request")
	require.True(t, ok, "expected completions after successful reconnect.")
}

func assertEditEvent(t *testing.T, file string, content string, p *clientapp.TestEnvironment) {
	_, err := p.KitedClient.PostEditEvent(editor, file, content, int64(len(content)))
	require.NoError(t, err)
}

func assertCompletions(file string, content string, p *clientapp.TestEnvironment) (bool, error) {
	req := completionsRequest{
		Editor:      editor, // same value as used by PostEditEvent
		Filename:    file,
		CursorRunes: int64(len(content)),
		Text:        content,
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return false, err
	}

	resp, err := p.KitedClient.Post("/clientapi/editor/completions", bytes.NewReader(reqJSON))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	return len(bytes) > 0, nil
}

type completionsRequest struct {
	Editor      string `json:"editor"`
	Filename    string `json:"filename"`
	Text        string `json:"text"`
	CursorBytes int64  `json:"cursor_bytes"`
	CursorRunes int64  `json:"cursor_runes"`
}
