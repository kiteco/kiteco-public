package debug

import (
	_ "net/http/pprof"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/auth"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Component(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	m := NewManager()
	component.TestImplements(t, m, component.Implements{
		Initializer: true,
		Handlers:    true,
	})
}

func Test_manager(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	//setup component
	mgr := NewManager()
	err = auth.SetupWithAuthDefaults(s, mgr)
	require.NoError(t, err)

	//status when not logged in
	data := &userMachine{}
	err = s.KitedClient.GetJSON("/debug/user-machine", data)
	require.NoError(t, err)
	require.Equal(t, s.Platform.MachineID, data.Machine)
	require.Nil(t, data.User)

	//status after login
	_, err = s.SendLoginRequest("user@example.com", "secret", true)
	require.NoError(t, err)
	err = s.KitedClient.GetJSON("/debug/user-machine", data)
	require.NoError(t, err)
	require.Equal(t, s.Platform.MachineID, data.Machine)
	require.NotNil(t, data.User)
	require.Equal(t, "user@example.com", data.User.Email)

	//make sure that pprof (imported above) is available at /debug/...
	resp, err := s.DoKitedGet("/debug/pprof/heap")
	assert.NoError(t, err)
	assert.EqualValues(t, 200, resp.StatusCode)
}
