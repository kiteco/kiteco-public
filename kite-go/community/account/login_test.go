package account

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/stretchr/testify/require"
)

type userHTTPCase struct {
	user     string
	password string
	email    string
	status   int
}

func Test_HandleCreateWeb(t *testing.T) {
	ts, _, manager, _ := makeTestServer(t)
	defer ts.Close()
	defer requireCleanup(t, manager)

	longPassword := strings.Repeat("long", 15)
	testCases := []userHTTPCase{
		userHTTPCase{"fred1", "goodpassword", "fred1@example.com", http.StatusOK},        // Everythings good
		userHTTPCase{"fred1", "goodpassword", "fred1@example.com", http.StatusConflict},  // User already exists
		userHTTPCase{"fred2", "short", "fred2@example.com", http.StatusBadRequest},       // Short password
		userHTTPCase{"fred3", longPassword, "fred3@example.com", http.StatusBadRequest},  // Long password
		userHTTPCase{"fred4", "goodpassword", "fred4example.com", http.StatusBadRequest}, // Bad email
		userHTTPCase{"fred5", "goodpassword", "", http.StatusBadRequest},                 // Empty email
	}

	for _, test := range testCases {
		createURL := makeTestURL(ts.URL, "/api/account/create-web")
		vals := url.Values{}
		vals.Set("name", test.user)
		vals.Set("email", test.email)
		vals.Set("password", test.password)

		resp, err := http.PostForm(createURL, vals)
		require.NoError(t, err)
		defer resp.Body.Close()

		checkUserResponse(t, test, resp)
	}
}

func Test_HandleLoginWeb(t *testing.T) {
	ts, _, manager, _ := makeTestServer(t)
	defer ts.Close()
	defer requireCleanup(t, manager)

	requireCreateWeb(t, ts, manager, "fred1", "goodpassword")

	testCases := []userHTTPCase{
		userHTTPCase{"fred1", "goodpassword", "fred1@example.com", http.StatusOK},            // Login succeeds
		userHTTPCase{"fred1", "wrongpassword", "fred1@example.com", http.StatusUnauthorized}, // Wrong password
		userHTTPCase{"fred4", "goodpassword", "fred4example.com", http.StatusUnauthorized},   // Invalid email
		userHTTPCase{"fred4", "goodpassword", "fred4@example.com", http.StatusUnauthorized},  // Email doesn't exist
	}

	for _, test := range testCases {
		loginURL := makeTestURL(ts.URL, "/api/account/login-web")
		vals := url.Values{}
		vals.Set("email", test.email)
		vals.Set("password", test.password)

		resp, err := http.PostForm(loginURL, vals)
		require.NoError(t, err)
		defer resp.Body.Close()

		checkUserResponse(t, test, resp)
	}
}

func Test_HandleLoginDesktop(t *testing.T) {
	ts, _, manager, _ := makeTestServer(t)
	defer ts.Close()
	defer requireCleanup(t, manager)

	requireCreateWeb(t, ts, manager, "fred2", "goodpassword")

	// Make sure invalid logins don't activate referral
	invalidTestCases := []userHTTPCase{
		userHTTPCase{"fred2", "wrongpassword", "fred2@example.com", http.StatusUnauthorized}, // Wrong password
		userHTTPCase{"fred2", "goodpassword", "fred2example.com", http.StatusUnauthorized},   // Invalid email
		userHTTPCase{"fred2", "goodpassword", "fred5@example.com", http.StatusUnauthorized},  // Email doesn't exist
	}

	for _, test := range invalidTestCases {
		loginURL := makeTestURL(ts.URL, "/api/account/login-desktop")
		vals := url.Values{}
		vals.Set("email", test.email)
		vals.Set("password", test.password)

		resp, err := http.PostForm(loginURL, vals)
		require.NoError(t, err)
		defer resp.Body.Close()

		checkUserResponse(t, test, resp)
	}

	// Ok, happy path now
	testCases := []userHTTPCase{
		userHTTPCase{"fred2", "goodpassword", "fred2@example.com", http.StatusOK}, // Everything is awesome!
	}

	for _, test := range testCases {
		loginURL := makeTestURL(ts.URL, "/api/account/login-desktop")
		vals := url.Values{}
		vals.Set("email", test.email)
		vals.Set("password", test.password)

		resp, err := http.PostForm(loginURL, vals)
		require.NoError(t, err)
		defer resp.Body.Close()

		checkUserResponse(t, test, resp)
	}
}

// --

func requireCreateWeb(t *testing.T, ts *httptest.Server, manager *manager, name, password string) community.User {
	createURL := makeTestURL(ts.URL, "/api/account/create-web")
	vals := url.Values{}
	vals.Set("name", name)
	vals.Set("email", fmt.Sprintf("%s@example.com", name))
	vals.Set("password", password)

	resp, err := http.PostForm(createURL, vals)
	require.NoError(t, err)
	defer resp.Body.Close()

	var user community.User
	err = json.NewDecoder(resp.Body).Decode(&user)
	require.NoError(t, err)

	return user
}

// --

func checkUserResponse(t *testing.T, test userHTTPCase, resp *http.Response) {
	buf, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, test.status, resp.StatusCode, string(buf))

	if resp.StatusCode == http.StatusOK {
		var user community.User
		err = json.Unmarshal(buf, &user)
		require.NoError(t, err)
		require.Equal(t, test.user, user.Name)
		require.Equal(t, test.email, user.Email)
	}
}

type mockEmailVerifier struct {
	calledCreate bool
}

func (m *mockEmailVerifier) Create(addr string) (*community.EmailVerification, error) {
	m.calledCreate = true
	return &community.EmailVerification{Email: addr}, nil
}

func (m *mockEmailVerifier) Lookup(email, code string) (*community.EmailVerification, error) {
	return &community.EmailVerification{Email: email}, nil
}

func (m *mockEmailVerifier) Remove(v *community.EmailVerification) error {
	return nil
}

func (m *mockEmailVerifier) checkAndReset(t *testing.T) {
	require.True(t, m.calledCreate)
	m.calledCreate = false
}

func (m *mockEmailVerifier) Migrate() error {
	return nil
}
