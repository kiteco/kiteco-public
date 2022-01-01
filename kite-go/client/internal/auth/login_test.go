package auth

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_BackendLoginTimeout(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	assert.NoError(t, err)
	defer s.Close()

	err = SetupWithAuthDefaults(s)
	assert.NoError(t, err)

	s.Backend.AddPrefixRequestHandler("/api/account/login-desktop", []string{"POST"}, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(defaultHTTPTimeoutTest + 500*time.Millisecond)
	})

	resp, err := s.SendLoginRequest("user@example.com", "secret", false)
	require.NoError(t, err)
	assert.EqualValues(t, http.StatusInternalServerError, resp.StatusCode, "expected error response when the backend request timed out")
}

func Test_BackendCreateAccountTimeout(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	assert.NoError(t, err)
	defer s.Close()

	err = SetupWithAuthDefaults(s)
	assert.NoError(t, err)

	s.Backend.AddPrefixRequestHandler("/api/account/create-web", []string{"POST"}, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(defaultHTTPTimeoutTest + 500*time.Millisecond)
	})

	resp, err := s.SendAccountCreationRequest("user_next@example.com", "longEnough", false)
	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusInternalServerError, resp.StatusCode, "expected error response when the backend request timed out")
}

func Test_PlanLoginWithKiteLocal(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	assert.NoError(t, err)
	defer s.Close()

	err = SetupWithAuthDefaults(s)
	assert.NoError(t, err)

	_, err = s.SendLoginRequest("user@example.com", "secret", true)
	require.NoError(t, err)

	plan, err := s.CurrentPlan()
	assert.NoError(t, err)
	assert.EqualValues(t, &ProDefaultPlan, plan)
}

func Test_PlanCreateAccountWithKiteLocal(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	assert.NoError(t, err)
	defer s.Close()

	err = SetupWithAuthDefaults(s)
	assert.NoError(t, err)

	_, err = s.SendAccountCreationRequest("arealemail@something.com", "longEnough", true)
	assert.NoError(t, err)

	plan, err := s.CurrentPlan()
	assert.NoError(t, err)
	assert.EqualValues(t, &ProDefaultPlan, plan)
}

func Test_LogoutTimeout(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	assert.NoError(t, err)
	defer s.Close()

	err = SetupWithAuthDefaults(s)
	assert.NoError(t, err)

	s.Backend.AddPrefixRequestHandler("/api/account/logout", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(defaultHTTPTimeoutTest + 500*time.Millisecond)
	})

	_, err = s.SendLoginRequest("user@example.com", "secret", false)
	require.NoError(t, err)

	resp, err := s.SendLogoutRequest(false)
	require.NoError(t, err)
	assert.EqualValues(t, http.StatusInternalServerError, resp.StatusCode, "expected error status when the logout request timed out")
}

func Test_FetchUserInvalidServerData(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	assert.NoError(t, err)
	defer s.Close()

	err = SetupWithAuthDefaults(s)
	assert.NoError(t, err)

	_, err = s.SendLoginRequest("user@example.com", "secret", true)
	require.NoError(t, err)

	// don't timeout at first to make the initial login pass
	s.Backend.AddPrefixRequestHandler("/api/account/user", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		// no data sent
	})

	_, err = s.AuthClient.FetchUser(context.Background())
	require.Error(t, err, "expected error for invalid data sent in server response")
}

func Test_CheckAuthenticated(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	assert.NoError(t, err)
	defer s.Close()

	auth := NewTestClient(defaultHTTPTimeoutTest)
	defer os.Remove(auth.userCacheFile)
	s.SetupWithCustomAuthClient(auth)
	auth.SetTarget(s.Backend.URL)

	_, err = s.SendLoginRequest("user@example.com", "secret", true)
	require.NoError(t, err)

	err = auth.checkAuthenticated(context.Background())
	require.NoError(t, err, "no error expected if backend request returns on time")

	// test unauthorized
	s.Backend.AddPrefixRequestHandler("/api/account/authenticated", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	err = auth.checkAuthenticated(context.Background())
	require.EqualError(t, err, ErrNotAuthenticated.Error(), "error expected when backend request returned an error")

	// test backend response
	s.Backend.AddPrefixRequestHandler("/api/account/authenticated", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	err = auth.checkAuthenticated(context.Background())
	require.Error(t, err, "error expected when backend request returned an error")

	// test timeout
	s.Backend.AddPrefixRequestHandler("/api/account/authenticated", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(defaultHTTPTimeoutTest + 500*time.Millisecond)
	})
	err = auth.checkAuthenticated(context.Background())
	require.Error(t, err, "error expected when backend request timed out")
}
