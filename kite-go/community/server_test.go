package community

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//////////////////////////////////////////////////////////////////////////////
// Test account endpoints

type userHTTPCase struct {
	user     string
	password string
	email    string
	status   int
}

type mockEmailVerifier struct {
	calledCreate bool
}

func (m *mockEmailVerifier) Create(addr string) (*EmailVerification, error) {
	m.calledCreate = true
	return &EmailVerification{Email: addr}, nil
}

func (m *mockEmailVerifier) Lookup(email, code string) (*EmailVerification, error) {
	return &EmailVerification{Email: email}, nil
}

func (m *mockEmailVerifier) Remove(v *EmailVerification) error {
	return nil
}

func (m *mockEmailVerifier) Migrate() error {
	return nil
}

func (m *mockEmailVerifier) checkAndReset(t *testing.T) {
	assert.True(t, m.calledCreate, "EmailVerifier.Create was not called")
	m.calledCreate = false
}

func Test_HandleCreate(t *testing.T) {
	ts, _, app := makeTestServer()
	defer requireCleanupApp(t, app)

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
		createURL := makeTestURL(ts.URL, "/api/account/create")
		vals := url.Values{}
		vals.Set("name", test.user)
		vals.Set("email", test.email)
		vals.Set("password", test.password)
		vals.Set("invite_code", BypassInviteCode)

		resp, err := http.PostForm(createURL, vals)
		if err != nil {
			t.Fatal("error posting to /api/account/create:", err)
		}
		defer resp.Body.Close()

		checkUserResponse(test, resp, t)
	}
}

func Test_HandleLogin(t *testing.T) {
	ts, _, app := makeTestServer()
	defer requireCleanupApp(t, app)

	_, _, err := app.Users.Create("fred1", "fred1@example.com", "goodpassword", "")

	if err != nil {
		t.Fatal(err)
	}

	testCases := []userHTTPCase{
		userHTTPCase{"fred1", "goodpassword", "fred1@example.com", http.StatusOK},            // Login succeeds
		userHTTPCase{"fred1", "wrongpassword", "fred1@example.com", http.StatusUnauthorized}, // Wrong password
		userHTTPCase{"fred4", "goodpassword", "fred4example.com", http.StatusUnauthorized},   // Invalid email
		userHTTPCase{"fred4", "goodpassword", "fred4@example.com", http.StatusUnauthorized},  // Email doesn't exist
	}

	for _, test := range testCases {
		loginURL := makeTestURL(ts.URL, "/api/account/login")
		vals := url.Values{}
		vals.Set("email", test.email)
		vals.Set("password", test.password)

		resp, err := http.PostForm(loginURL, vals)
		if err != nil {
			t.Fatal("error posting to /api/account/login:", err)
		}
		defer resp.Body.Close()

		checkUserResponse(test, resp, t)
	}
}

func Test_HandleLogout(t *testing.T) {
	ts, _, app := makeTestServer()
	defer requireCleanupApp(t, app)

	client := makeTestClient()

	resp := createHTTPUser(client, ts.URL, "fred1", "fred1@example.com", "goodpassword", t)
	defer resp.Body.Close()

	resp, err := client.Get(makeTestURL(ts.URL, "/api/account/logout"))
	if err != nil {
		t.Fatal("error getting to /api/account/logout:", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("could not log out of account; got status code:", resp.StatusCode)
	}

	resp, err = client.Get(makeTestURL(ts.URL, "/api/account/user"))
	if err != nil {
		t.Fatal("error getting /api/account/user:", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatal("apparently logging out did not actually work; instead of 401 got:", resp.StatusCode)
	}
}

func Test_HandleValidateSession_ValidSession(t *testing.T) {
	ts, _, app := makeTestServer()
	defer requireCleanupApp(t, app)

	client := makeTestClient()

	resp := createHTTPUser(client, ts.URL, "fred1", "fred1@example.com", "goodpassword", t)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		text, _ := ioutil.ReadAll(resp.Body)
		t.Fatalf("failed to create account: '%s'", text)
	}

	validateURL := makeTestURL(ts.URL, "/api/account/user")
	resp, err := client.Get(validateURL)
	if err != nil {
		t.Fatal("error getting /api/account/user:", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatal("could not validate user after creating account. validate returned:", resp.StatusCode)
	}
}

func Test_HandleValidateSession_NoCookie(t *testing.T) {
	ts, _, app := makeTestServer()
	defer requireCleanupApp(t, app)

	client := makeTestClient()
	validateURL := makeTestURL(ts.URL, "/api/account/user")
	resp, err := client.Get(validateURL)
	if err != nil {
		t.Fatal("error posting to /api/account/user:", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected validation to return: %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func Test_HandleCheckEmail(t *testing.T) {
	ts, _, app := makeTestServer()
	defer requireCleanupApp(t, app)

	_, _, err := app.Users.Create("fred1", "fred1@example.com", "goodpassword", "")
	require.NoError(t, err, "error creating user: %v", err)

	checkEmailURL := makeTestURL(ts.URL, "/api/account/check-email")

	post := func(email string) *http.Response {
		type request struct {
			Email string `json:"email"`
		}
		buf, err := json.Marshal(request{Email: email})
		require.NoError(t, err, "error marshalling email request: %v", err)

		resp, err := http.Post(checkEmailURL, "application/json", bytes.NewBuffer(buf))
		require.NoError(t, err, "error posting response for check email: %v", err)
		require.NotNil(t, resp, "nil response posting for check email")
		return resp
	}

	type response struct {
		FailReason string `json:"fail_reason"`
	}

	unmarshal := func(r io.ReadCloser) response {
		buf, err := ioutil.ReadAll(r)
		require.NoError(t, err, "error reading check email response: %v", err)

		var resp response
		require.NoError(t, json.Unmarshal(buf, &resp), "error unmarshalling check email response: %v", err)
		return resp
	}

	// existing email
	resp := post("fred1@example.com")
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, unmarshal(resp.Body), "expected non empty fail reason for existing email check")

	// bad email
	resp = post("fred1example.com")
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, unmarshal(resp.Body), "expected non empty fail reason for bad email check")

	// valid email
	resp = post("fred@example.com")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "", resp.Header.Get("Content-Type"))
}

func Test_HandleCheckPassword(t *testing.T) {
	ts, _, app := makeTestServer()
	defer requireCleanupApp(t, app)

	checkPasswordURL := makeTestURL(ts.URL, "/api/account/check-password")

	post := func(password string) *http.Response {
		type request struct {
			Password string `json:"password"`
		}
		buf, err := json.Marshal(request{Password: password})
		require.NoError(t, err, "error marshalling password request: %v", err)

		resp, err := http.Post(checkPasswordURL, "application/json", bytes.NewBuffer(buf))
		require.NoError(t, err, "error posting response for check password: %v", err)
		require.NotNil(t, resp, "nil response posting for check password")
		return resp
	}

	type response struct {
		FailReason string `json:"fail_reason"`
	}

	unmarshal := func(r io.ReadCloser) response {
		buf, err := ioutil.ReadAll(r)
		require.NoError(t, err, "error reading check password response: %v", err)

		var resp response
		require.NoError(t, json.Unmarshal(buf, &resp), "error unmarshalling check password response: %v", err)
		return resp
	}

	// too short
	resp := post("a")
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, unmarshal(resp.Body).FailReason, "expected non empty fail reason for too short password")

	// too long
	resp = post(strings.Repeat("a", maxPasswordLength+1))
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, unmarshal(resp.Body).FailReason, "expected non empty fail reason for too long password")

	// just right
	resp = post(strings.Repeat("a", minPasswordLength+1))
	assert.Equal(t, "", resp.Header.Get("Content-Type"))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func Test_RemoteIP(t *testing.T) {
	req, err := http.NewRequest("GET", "http://www.example.com/", nil)
	require.NoError(t, err)

	req.Header.Set("X-Forwarded-For", "1.1.1.1")
	ip := remoteIP(req)
	require.Equal(t, "1.1.1.1", ip)

	req.Header.Set("X-Forwarded-For", "1.1.1.1, 2.2.2.2")
	ip = remoteIP(req)
	require.Equal(t, "1.1.1.1", ip)

	req.Header.Set("X-Forwarded-For", "1.1.1.1:1234, 2.2.2.2")
	ip = remoteIP(req)
	require.Equal(t, "1.1.1.1", ip)

	req.Header.Del("X-Forwarded-For")
	ip = remoteIP(req)
	require.Equal(t, "", ip)
}

// --

func checkUserResponse(test userHTTPCase, resp *http.Response, t *testing.T) {
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("error reading body:", err)
	}

	if resp.StatusCode != test.status {
		t.Fatalf("expected status code: %d, got %d, for: %+v, body: %s", test.status, resp.StatusCode, test, buf)
	}

	if resp.StatusCode == http.StatusOK {
		var user User
		err = json.Unmarshal(buf, &user)
		if err != nil {
			t.Fatal("error unmarshalling user:", err)
		}

		if user.Name != test.user {
			t.Fatalf("expected name: %s, got: %s, from: %+v", test.user, user.Name, test)
		}
		if user.Email != test.email {
			t.Fatalf("expected email: %s, got: %s, from: %+v", test.email, user.Email, test)
		}
	}
}

func createHTTPUser(client *http.Client, base, name, email, password string, t *testing.T) *http.Response {
	createURL := makeTestURL(base, "/api/account/create")
	vals := url.Values{}
	vals.Set("name", name)
	vals.Set("email", email)
	vals.Set("password", password)

	resp, err := client.PostForm(createURL, vals)
	if err != nil {
		t.Fatal("error posting to /api/account/create:", err)
	}
	return resp
}
