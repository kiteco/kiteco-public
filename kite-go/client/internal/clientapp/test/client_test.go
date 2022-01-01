package test

import (
	"strings"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/clientapp"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/test"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DebugEndpoints(t *testing.T) {
	p, err := clientapp.StartEmptyTestEnvironment()
	require.NoError(t, err)
	defer p.Close()

	resp, err := p.KitedClient.Get("/debug/user-machine")
	require.Equal(t, 200, resp.StatusCode)
}

func Test_BackendServerChange(t *testing.T) {
	s, err := clientapp.StartEmptyTestEnvironment()
	assert.NoError(t, err)
	defer s.Close()

	kitedTarget := s.Kited.AuthClient.Target()
	require.NotNil(t, kitedTarget, "kited target url must be valid")

	backendURL := s.Backend.URL
	require.NotNil(t, backendURL, "backend url must be valid")

	require.EqualValues(t, backendURL.String(), kitedTarget.String())

	backend2, err := mockserver.NewBackend(map[string]string{})
	require.NoError(t, err)

	// make sure that the client reconnects to the new URL when the settings were modified
	s.Kited.Settings.Set(settings.ServerKey, backend2.URL.String())
	waitFor(t, func() bool {
		// the client starts a go routine to connect to the new target server
		return s.Kited.AuthClient.Target().String() == backend2.URL.String()
	})
}

func Test_UpdaterEndpoints(t *testing.T) {
	p, err := clientapp.StartEmptyTestEnvironment()
	require.NoError(t, err)
	defer p.Close()

	resp, err := p.KitedClient.Get("/clientapi/update/check")
	require.NoError(t, err)
	require.NotEqual(t, 404, resp.StatusCode)

	resp, err = p.KitedClient.Get("/clientapi/update/readyToRestart")
	require.NoError(t, err)
	require.NotEqual(t, 404, resp.StatusCode)

	resp, err = p.KitedClient.Get("/clientapi/update/restartAndUpdate")
	require.NoError(t, err)
	require.NotEqual(t, 404, resp.StatusCode)
}

func Test_AutosearchGlobalSymbols(t *testing.T) {
	cleanupTimeout := setEventTimeout()
	defer cleanupTimeout()

	p, err := clientapp.StartDefaultTestEnvironment(false, nil)
	require.NoError(t, err)
	defer p.Close()
	p.SetOffline(true)

	autosearchClient, err := test.NewClient(p)
	require.NoError(t, err)
	defer autosearchClient.Close()

	editor := test.NewEditorRemoteControl("atom", p, t)
	editor.OpenNewFile("file1.py")
	editor.Input("import json\n")
	editor.Input("json")

	autosearchID, err := autosearchClient.ReceiveClientMessage()
	require.NoError(t, err)
	require.NotEmpty(t, autosearchID)
	require.True(t, strings.Contains(autosearchID, "python;;;;;json"), "autosearch ID does not contain expected id. value: %s", autosearchID)

	// retrieve the documentation for the given autosearchID to replicate what Copilot is doing
	report, err := p.KitedClient.SymbolReport(autosearchID)
	require.NoError(t, err)
	require.NotEmpty(t, report.Report.DescriptionHTML)

	members, err := p.KitedClient.Members(autosearchID)
	require.NoError(t, err)
	require.NotEmpty(t, members.Members, "members expected for json module")
}

func Test_AutosearchUnsaved(t *testing.T) {
	cleanupTimeout := setEventTimeout()
	defer cleanupTimeout()

	p, err := clientapp.StartDefaultTestEnvironment(false, nil)
	require.NoError(t, err)
	defer p.Close()
	p.SetOffline(true)

	autosearchClient, err := test.NewClient(p)
	require.NoError(t, err)
	defer autosearchClient.Close()

	editor := test.NewEditorRemoteControl("atom", p, t)
	editor.OpenNewFile("file1.py")
	editor.Input("import json\n")
	editor.Input("abc = \"hi\"\n")
	editor.Input("abc")

	autosearchID, err := autosearchClient.ReceiveClientMessage()
	require.NoError(t, err)
	require.NotEmpty(t, autosearchID)
	require.True(t, strings.Contains(autosearchID, "file1.py;;abc"), "autosearch ID does not contain expected id. value: %s", autosearchID)

	// retrieve the documentation for the given autosearchID to replicate what Copilot is doing
	report, err := p.KitedClient.SymbolReport(autosearchID)
	require.NoError(t, err)
	require.NotEmpty(t, report.Report.DescriptionHTML)

	members, err := p.KitedClient.Members(autosearchID)
	require.NoError(t, err)
	require.NotNil(t, members)
}

func Test_AutosearchFunctionCall(t *testing.T) {
	cleanupTimeout := setEventTimeout()
	defer cleanupTimeout()

	p, err := clientapp.StartDefaultTestEnvironment(false, nil)
	require.NoError(t, err)
	defer p.Close()
	p.SetOffline(true)

	autosearchClient, err := test.NewClient(p)
	require.NoError(t, err)
	defer autosearchClient.Close()

	editor := test.NewEditorRemoteControl("atom", p, t)
	editor.OpenNewFile("file1.py")
	editor.Input("import json\n")
	editor.Input("json.dumps(")

	autosearchID, err := autosearchClient.ReceiveClientMessage()
	require.NoError(t, err)
	require.NotEmpty(t, autosearchID)
	require.True(t, strings.Contains(autosearchID, "python;;;;json;dumps"), "autosearch ID does not contain expected id. value: %s", autosearchID)

	// retrieve the documentation for the given autosearchID to replicate what Copilot is doing
	report, err := p.KitedClient.SymbolReport(autosearchID)
	require.NoError(t, err)
	require.NotEmpty(t, report.Report.DescriptionHTML)

	members, err := p.KitedClient.Members(autosearchID)
	require.NoError(t, err)
	require.NotEmpty(t, members.Members, "members expected for json module")
}

func setEventTimeout() func() {
	old := kitelocal.TestSetFileEventTimeout(10 * time.Second)
	return func() {
		kitelocal.TestSetFileEventTimeout(old)
	}
}
