package desktoplogin

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/auth"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Component(t *testing.T) {
	m := NewManager()

	component.TestImplements(t, m, component.Implements{
		Initializer: true,
		Handlers:    true,
	})
}

// tests a successful redirection to the backend server
func Test_SuccessfulRedirect(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	err = auth.SetupWithAuthDefaults(s, NewManager())
	assert.NoError(t, err)

	_, err = s.SendLoginRequest("user@example.com", "secret", true)
	require.NoError(t, err)

	s.Backend.AddPrefixRequestHandler("/account/login-nonce", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Value":"my-nonce-value"}`))
	})

	response, err := s.DoKitedGet("/clientapi/desktoplogin?d=/docs/")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusFound, response.StatusCode)

	dst, err := s.Backend.URL.Parse("/docs/")
	require.NoError(t, err)

	backendHost := s.Backend.URL.Host
	v := make(url.Values)
	v.Add("d", dst.String())
	v.Add("n", "my-nonce-value")

	assert.Equal(t, fmt.Sprintf("http://%s/account/desktop-login?%s", backendHost, v.Encode()), response.Header.Get("Location"))
}

func Test_SuccessfulRedirectNotLoggedIn(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	err = auth.SetupWithAuthDefaults(s, NewManager())
	assert.NoError(t, err)

	s.Backend.AddPrefixRequestHandler("/account/login-nonce", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Value":"my-nonce-value"}`))
	})

	response, err := s.DoKitedGet("/clientapi/desktoplogin?d=/docs/")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusFound, response.StatusCode)

	backendHost := s.Backend.URL.Host
	assert.Equal(t, fmt.Sprintf("http://%s/docs/", backendHost), response.Header.Get("Location"))
}

// Tests with a missing target url, i.e. with a missing d= paramter value
func Test_MissingTargetUrl(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	err = auth.SetupWithAuthDefaults(s, NewManager())
	assert.NoError(t, err)

	response, err := s.DoKitedGet("/clientapi/desktoplogin")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

// Tests redirection when the backend nonce request failed, in this case the client will redirect to the location (which will require a login)
func Test_MissingBackendUrl(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	err = auth.SetupWithAuthDefaults(s, NewManager())
	assert.NoError(t, err)

	//unset target URL configuration
	s.AuthClient.UnsetTarget()

	response, err := s.DoKitedGet("/clientapi/desktoplogin?d=/docs/")
	assert.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
}

func Test_InvalidBackendJson(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	err = auth.SetupWithAuthDefaults(s, NewManager())
	assert.NoError(t, err)

	//returns invalid json response
	s.Backend.AddPrefixRequestHandler("/account/login-nonce", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid-json"))
	})

	//expect a redirect to the original target location (which will display a login page)
	response, err := s.DoKitedGet("/clientapi/desktoplogin?d=/docs/")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusFound, response.StatusCode)

	assert.Equal(t, fmt.Sprintf("http://%s/docs/", s.Backend.URL.Host), response.Header.Get("Location"))
}

func Test_BackendInvalidNonce(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	err = auth.SetupWithAuthDefaults(s, NewManager())
	assert.NoError(t, err)

	//returns empty json response
	s.Backend.AddPrefixRequestHandler("/account/login-nonce", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{}"))
	})

	//empty nonce: expect a redirect to the original target location (which will display a login page)
	response, err := s.DoKitedGet("/clientapi/desktoplogin?d=/docs/")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusFound, response.StatusCode)

	assert.Equal(t, fmt.Sprintf("http://%s/docs/", s.Backend.URL.Host), response.Header.Get("Location"))
}

func Test_BackendFailedResponse(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	err = auth.SetupWithAuthDefaults(s, NewManager())
	assert.NoError(t, err)

	//returns internal server error status code
	s.Backend.AddPrefixRequestHandler("/account/login-nonce", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	//expect a redirect to the original target location (which will display a login page)
	response, err := s.DoKitedGet("/clientapi/desktoplogin?d=/docs/")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusFound, response.StatusCode)

	assert.Equal(t, fmt.Sprintf("http://%s/docs/", s.Backend.URL.Host), response.Header.Get("Location"))
}

// tests a redirect to an invalid target URL
func Test_InvalidTargetUrl(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	err = auth.SetupWithAuthDefaults(s, NewManager())
	assert.NoError(t, err)

	//returns empty json response
	s.Backend.AddPrefixRequestHandler("/account/login-nonce", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{}"))
	})

	// request a redirect to %INVALID_URL_URL%, which must trigger a bad request response
	response, err := s.DoKitedGet("/clientapi/desktoplogin?d=%INVALID_URL_HERE%")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}
