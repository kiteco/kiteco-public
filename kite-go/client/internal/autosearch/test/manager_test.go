package test

import (
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/auth"
	"github.com/kiteco/kiteco/kite-go/client/internal/autosearch"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/test"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ComponentInterfaces(t *testing.T) {
	m := autosearch.NewManager()
	defer m.Terminate()

	component.TestImplements(t, m, component.Implements{
		Handlers:   true,
		Terminater: true,
	})
}

func Test_Component(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{})
	require.NoError(t, err)
	defer s.Close()

	mgr := autosearch.NewManager()
	defer mgr.Terminate()

	err = auth.SetupWithAuthDefaults(s, mgr)
	require.NoError(t, err)

	autosearchClient, err := test.NewConfiguredClient(*s.Kited.URL, mgr)
	require.NoError(t, err)
	defer autosearchClient.Close()

	// Test with old-style message type
	autosearchClient.BroadcastServerMessage("hello world")
	id, err := autosearchClient.ReceiveClientMessage()
	require.NoError(t, err)
	assert.EqualValues(t, "hello world", id)

	// Test with response.Autosearch in results (e.g kite local)
	autosearchClient.BroadcastServerAutosearchMessage("hello world")
	id, err = autosearchClient.ReceiveClientMessage()
	require.NoError(t, err)
	assert.EqualValues(t, "hello world", id)

	//make sure that Terminate closes connections
	assert.EqualValues(t, 1, mgr.ActiveConnectionCount())
	mgr.Terminate()
	assert.EqualValues(t, 0, mgr.ActiveConnectionCount())
}
