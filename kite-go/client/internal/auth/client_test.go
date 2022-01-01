package auth

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const defaultHTTPTimeoutTest = 300 * time.Millisecond

func Test_Component(t *testing.T) {
	authority, err := licensing.NewTestAuthority()
	require.NoError(t, err)

	c := NewClient(licensing.NewStore(authority.CreateValidator(), ""))
	component.TestImplements(t, c, component.Implements{
		Initializer: true,
		Ticker:      true,
		Handlers:    true,
		Settings:    true,
	})

	var comp interface{} = c
	_, ok := comp.(component.AuthClient)
	assert.True(t, ok, "component must implement component.AuthClient")
}

func Test_Token(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{})
	require.NoError(t, err)
	defer s.Close()

	// use our own client, s.AuthClient is using the interface which isn't providing the token anymore
	auth := NewTestClient(defaultHTTPTimeoutTest)
	defer os.Remove(auth.userCacheFile)
	s.SetupWithCustomAuthClient(auth)
	assert.NotNil(t, auth.token)
}

func Test_HTTP_Client(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{})
	require.NoError(t, err)
	defer s.Close()

	auth := NewTestClient(defaultHTTPTimeoutTest)
	defer os.Remove(auth.userCacheFile)
	s.SetupWithCustomAuthClient(auth)
	s.AuthClient.SetTarget(s.Backend.URL)

	assert.NotNil(t, auth.httpClient())
}

func Test_Target(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{})
	assert.NoError(t, err)
	defer s.Close()

	SetupWithAuthDefaults(s)

	assert.EqualValues(t, s.Backend.URL, s.AuthClient.Target())
	s.AuthClient.UnsetTarget()

	_, err = s.AuthClient.Get(context.Background(), "/client-request")
	assert.Error(t, err, "request must be disabled without a valid target server")

	resp := httptest.NewRecorder()
	s.AuthClient.ServeHTTP(resp, httptest.NewRequest("GET", "/client-request", nil))
	assert.EqualValues(t, http.StatusInternalServerError, resp.Result().StatusCode, "expected an error without a valid target server")
}

// tests the AuthClient's login / logout requests using a mocked http backend and a mocked http kited client
func Test_Client(t *testing.T) {
	validUsers := map[string]string{"user1@example.com": "password1"}

	root, err := ioutil.TempDir("", "kite-test")
	require.NoError(t, err)
	defer func() {
		if !t.Failed() {
			os.RemoveAll(root)
		}
	}()

	s, err := mockserver.NewTestClientServerRootFeatures(root, validUsers, nil)
	assert.NoError(t, err)
	defer s.Close()
	SetupWithAuthDefaults(s)
	assert.NoError(t, err)
	assert.NotEmpty(t, s.AuthClient.Name())

	//make sure that no user is logged in at first
	user, err := s.CurrentUser()
	assert.EqualError(t, err, mockserver.ErrUnauthorized.Error())

	client := s.AuthClient.(*Client)
	err = client.checkAuthenticated(context.Background())
	assert.Error(t, err, "Error expected if not yet authenticated")

	//make sure that there's a valid HTTP client
	assert.NotNil(t, client.httpClient())

	// requests /clientapi/login from Kited, which then sends a login request to the backend mock server
	resp, err := s.SendLoginRequest("user1@example.com", validUsers["user1@example.com"], true)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.EqualValues(t, 1, s.Backend.LoginRequestCount(), "The backend's login request must be called once")

	// re-use the the same backend when creating new test clients
	backend := s.Backend

	s, err = mockserver.NewTestClientServerRootFeatures(root, validUsers, nil)
	assert.NoError(t, err)
	defer s.Close()

	s.ReadLoginLogoutEvents = false // handle events manually to avoid races
	s.Backend = backend
	require.NoError(t, SetupWithAuthDefaults(s))

	s.HandleAuthEvent() // handle init login event

	// make sure that /clientapi/user returns the current user
	user, err = s.CurrentUser()
	require.NoError(t, err)
	assert.Equal(t, "user1@example.com", user.Email)

	// requests /clientapi/logout from Kited, which then sends a logout request to the backend mock server
	resp, err = s.SendLogoutRequest(true)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.EqualValues(t, 1, s.Backend.LogoutRequestCount(), "The backend's logout request must be called once")

	s.HandleAuthEvent() // handle logout event

	s, err = mockserver.NewTestClientServerRootFeatures(root, validUsers, nil)
	assert.NoError(t, err)
	defer s.Close()

	s.Backend = backend
	require.NoError(t, SetupWithAuthDefaults(s))

	// make sure that /clientapi/user returns that no user is logged in
	user, err = s.CurrentUser()
	assert.EqualError(t, err, mockserver.ErrUnauthorized.Error())
	assert.Nil(t, user)
}

func Test_ComponentGoTick(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	assert.NoError(t, err)
	defer s.Close()
	s.ReadLoginLogoutEvents = false

	err = SetupWithAuthDefaults(s)
	require.NoError(t, err)
	require.NotEmpty(t, s.AuthClient.Name())

	client := s.AuthClient.(*Client)
	drainLoginChannel(s.AuthClient.LoggedInChan())

	forceTick(context.Background(), client)

	//timeout for user event channel
	ctx, cancel := context.WithTimeout(context.Background(), time.Second/2)
	defer cancel()

	// no message in logged out channel if still logged out
	select {
	case <-client.LoggedInChan():
		assert.Fail(t, "No message expected in logged-in channel")
	case <-client.LoggedOutChan():
		assert.Fail(t, "No message expected in logged-out channel")
	case <-ctx.Done():
	}

	// logged-in
	_, _ = s.SendLoginRequest("user@example.com", "secret", false)

	// emulate a proxy call to trigger authClient.ServeHTTP
	// the request is our only mock request which sends the session cookie back to the client
	_, err = s.DoKitedGet("/api/account/authenticated")
	require.NoError(t, err)

	// read event sent by login handler
	drainLoginChannel(s.AuthClient.LoggedInChan())
	drainLogoutChannel(s.AuthClient.LoggedOutChan())

	// Tick should generate nothing
	forceTick(context.Background(), client)

	//timeout for user event channel
	ctx, cancel = context.WithTimeout(context.Background(), time.Second/2)
	defer cancel()

	select {
	case <-client.LoggedInChan():
		assert.Fail(t, "No message expected in logged-in channel")
	case <-client.LoggedOutChan():
		assert.Fail(t, "No message expected in logged-out channel")
	case <-ctx.Done():
	}

	// Logout
	_, err = s.SendLogoutRequest(false)
	require.NoError(t, err)

	// read event sent by login handler
	drainLoginChannel(s.AuthClient.LoggedInChan())
	drainLogoutChannel(s.AuthClient.LoggedOutChan())

	// Tick should not generate logout
	forceTick(context.Background(), client)

	ctx, cancel = context.WithTimeout(context.Background(), time.Second/2)
	defer cancel()

	log.Println("waiting for logout")
	select {
	case <-client.LoggedInChan():
		assert.Fail(t, "No message expected in logged-in channel")
	case <-client.LoggedOutChan():
		assert.Fail(t, "No message expected in logged-in channel")
	case <-ctx.Done():
	}
}

func Test_ComponentTickDeadline(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	assert.NoError(t, err)
	defer s.Close()
	s.ReadLoginLogoutEvents = false

	err = SetupWithAuthDefaults(s)
	require.NoError(t, err)

	_, err = s.SendLoginRequest("user@example.com", "secret", false)
	require.NoError(t, err)

	// emulate a proxy call to trigger authClient.ServeHTTP
	// the request is our only mock request which sends the session cookie back to the client
	_, err = s.DoKitedGet("/api/account/authenticated")
	require.NoError(t, err)
	assert.EqualValues(t, 1, s.Backend.GetRequestCount("/api/account/authenticated"))

	// first tick must check the auth state
	client := s.AuthClient.(*Client)
	client.GoTick(context.Background())
	assert.EqualValues(t, 0, s.Backend.GetRequestCount("/api/account/user"))
	assert.EqualValues(t, 1, s.Backend.GetRequestCount("/api/account/authenticated"), "no request to authenticated expected")

	// subsequent calls will not call authenticated until interval passes
	client.GoTick(context.Background())
	client.GoTick(context.Background())
	assert.EqualValues(t, 0, s.Backend.GetRequestCount("/api/account/user"))

	//calls after the deadline passed must check the auth state
	forceTick(context.Background(), client)
	assert.EqualValues(t, 0, s.Backend.GetRequestCount("/api/account/user"))
}

// --

func forceTick(ctx context.Context, client *Client) {
	client.lastTickCheck = time.Time{}
	client.GoTick(ctx)
}

func drainLoginChannel(channel chan *community.User) {
	timeout := time.After(time.Second / 4)
	for {
		select {
		case <-channel:
			log.Println("Ignored login event...")
		case <-timeout:
			return
		}
	}
}

func drainLogoutChannel(channel chan struct{}) {
	timeout := time.After(time.Second / 4)
	for {
		select {
		case <-channel:
			log.Println("Ignored logout event...")
		case <-timeout:
			return
		}
	}
}
