package health

import (
	"net/http"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/auth"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ComponentInterfaces(t *testing.T) {
	m := NewManager()
	component.TestImplements(t, m, component.Implements{
		Handlers: true,
	})
}

func Test_Ping(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{})
	require.NoError(t, err)
	defer s.Close()

	m := NewManager()
	auth.SetupWithAuthDefaults(s, m)

	resp, err := s.DoKitedGet("/clientapi/ping")
	require.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
}
