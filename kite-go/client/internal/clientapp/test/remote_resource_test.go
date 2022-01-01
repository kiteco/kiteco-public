package test

import (
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"

	"github.com/kiteco/kiteco/kite-go/client/internal/client"
	"github.com/kiteco/kiteco/kite-go/client/internal/clientapp"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/test"
	"github.com/stretchr/testify/require"
)

func Test_RemotePythonResourceManager(t *testing.T) {
	pythonOpts := pythonresource.DefaultOptions
	server, addr, err := pythonresource.StartServer("127.0.0.1:0", false, pythonOpts)
	require.NoError(t, err)
	defer server.Close()

	p, err := clientapp.StartDefaultTestEnvironment(true, &client.Options{
		LocalOpts: kitelocal.Options{RemoteResourceManager: addr.String()},
	})
	require.NoError(t, err)
	defer p.Close()

	p.WaitForReady(5 * time.Second)

	// now emulate editor interaction
	editor := test.NewEditorRemoteControl("atom", p, t)
	editor.OpenNewFile("file1.py")
	for i := 0; i <= 5; i++ {
		editor.Input("\nimport numpy as np\n")
		editor.Input("np.")
		editor.Completions()
	}

	// fixme: the completions driver is performing background work,
	// 	which uses the python resourcemanager
	//  the manager is closed when the test terminates, but the background go routine might still be running a bit longer
	// 	this results in a panic, atm I don't see a good way to fix this
	time.Sleep(5 * time.Second)
}
