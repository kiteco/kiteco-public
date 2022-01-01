package systeminfo

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/auth"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/stretchr/testify/assert"
)

func Test_Component(t *testing.T) {
	m := NewManager()
	component.TestImplements(t, m, component.Implements{
		Handlers:    true,
		Initializer: true,
	})
}

func Test_SystemInfo(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user1@example.com": "password1"})
	defer s.Close()
	assert.NoError(t, err)

	status := NewManager()
	auth.SetupWithAuthDefaults(s, status)

	resp, err := s.DoKitedGet("/clientapi/systeminfo")
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func Test_Version(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user1@example.com": "password1"})
	defer s.Close()
	assert.NoError(t, err)

	status := NewManager()
	auth.SetupWithAuthDefaults(s, status)

	resp, err := s.DoKitedGet("/clientapi/version")
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	respJSON, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("{\"version\":\"%s\"}\n", s.Platform.ClientVersion), string(respJSON))
}
