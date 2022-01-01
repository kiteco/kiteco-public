package account

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/community"

	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Licenses(t *testing.T) {
	ts, _, manager, _ := makeTestServer(t)
	defer ts.Close()
	defer requireCleanup(t, manager)

	user, _ := requireCreateUserSession(t, manager, "e@mail.com")

	authority, err := licensing.NewTestAuthority()
	require.NoError(t, err)

	now := time.Now()

	trialExpiry := now.Add(42 * 24 * time.Hour)
	trialLicense, err := authority.CreateLicense(licensing.Claims{
		UserID:    "licensed_user@example.com",
		Product:   licensing.Pro,
		Plan:      licensing.ProTrial,
		ExpiresAt: trialExpiry,
	})
	require.NoError(t, err)
	require.NotEmpty(t, trialLicense.Token)

	dbLicense := requireCreateLicense(t, manager, user, *trialLicense)
	require.NotNil(t, dbLicense)

	proExpiry := now.Add(30 * 24 * time.Hour)
	proLicense, err := authority.CreateLicense(licensing.Claims{
		UserID:    "licensed_user@example.com",
		Product:   licensing.Pro,
		Plan:      licensing.ProMonthly,
		ExpiresAt: proExpiry,
	})
	require.NoError(t, err)
	require.NotEmpty(t, trialLicense.Token)

	dbExpiredLicense := requireCreateLicense(t, manager, user, *proLicense)
	require.NotNil(t, dbExpiredLicense)

	// licenses account 1
	licenses, err := manager.Licenses(user)
	require.Equal(t, licenses.Len(), 2)

	expected := []string{
		proLicense.Token,
		trialLicense.Token,
	}
	var actual []string
	for l, next := licenses.Iterate(); l != nil; l = next() {
		actual = append(actual, l.Token)
	}
	require.ElementsMatch(t, expected, actual)
	require.Equal(t, proLicense.Token, licenses.License().Token)
}

func Test_LicensesMultipleUsers(t *testing.T) {
	ts, _, manager, _ := makeTestServer(t)
	defer ts.Close()
	defer requireCleanup(t, manager)

	user1, _ := requireCreateUserSession(t, manager, "licensed_user1@example.com")
	user2, _ := requireCreateUserSession(t, manager, "licensed_user2@example.com")
	user3, _ := requireCreateUserSession(t, manager, "licensed_user3@example.com")

	now := time.Now()

	authority, err := licensing.NewTestAuthority()
	require.NoError(t, err)

	licenseAcct1, err := authority.CreateLicense(licensing.Claims{
		UserID:    "licensed_user1@example.com",
		Product:   licensing.Pro,
		Plan:      licensing.ProMonthly,
		ExpiresAt: now.Add(1 * time.Hour),
	})
	require.NoError(t, err)

	licenseAcct2, err := authority.CreateLicense(licensing.Claims{
		UserID:    "licensed_user2@example.com",
		Product:   licensing.Pro,
		Plan:      licensing.ProMonthly,
		ExpiresAt: now.Add(1 * time.Hour),
	})
	require.NoError(t, err)

	licenseAcct3, err := authority.CreateLicense(licensing.Claims{
		UserID:    "licensed_user3@example.com",
		Product:   licensing.Pro,
		Plan:      licensing.ProMonthly,
		ExpiresAt: now.Add(1 * time.Hour),
	})
	require.NoError(t, err)

	dbLicense1 := requireCreateLicense(t, manager, user1, *licenseAcct1)
	dbLicense2 := requireCreateLicense(t, manager, user2, *licenseAcct2)
	dbLicense3 := requireCreateLicense(t, manager, user3, *licenseAcct3)

	dbLicenses1, _ := manager.Licenses(user1)
	require.Equal(t, dbLicenses1.Len(), 1)
	require.EqualValues(t, dbLicense1.Token, dbLicenses1.License().Token)

	dbLicenses2, _ := manager.Licenses(user2)
	require.Equal(t, dbLicenses2.Len(), 1)
	require.EqualValues(t, dbLicense2.Token, dbLicenses2.License().Token)

	dbLicenses3, _ := manager.Licenses(user3)
	require.Equal(t, dbLicenses3.Len(), 1)
	require.EqualValues(t, dbLicense3.Token, dbLicenses3.License().Token)
}

func Test_LicensesRequests(t *testing.T) {
	ts, _, manager, _ := makeTestServer(t)
	defer ts.Close()
	defer requireCleanup(t, manager)

	now := time.Now()

	// license and db setup
	user, _ := requireCreateUserSession(t, manager, "licensed_user1@example.com")
	authority, err := licensing.NewTestAuthority()
	require.NoError(t, err)
	licenseAcct1, err := authority.CreateLicense(licensing.Claims{
		UserID:    "licensed_user1@example.com",
		Product:   licensing.Pro,
		Plan:      licensing.ProMonthly,
		ExpiresAt: now.Add(1 * time.Hour),
	})
	require.NoError(t, err)

	_, err = manager.CreateLicense(user, *licenseAcct1, "ref")
	require.NoError(t, err)

	// http request tests
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	httpClient := ts.Client()
	httpClient.Jar = jar

	resp, err := httpClient.Get(makeTestURL(ts.URL, "/api/account/licenses"))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.EqualValues(t, http.StatusUnauthorized, resp.StatusCode, "a license request without a valid session must fail")

	resp, err = httpClient.PostForm(makeTestURL(ts.URL, "/api/account/login-web"), url.Values{
		"email": []string{user.Email},
		// as used in requireCreateUser
		"password": []string{"password123!"},
	})
	require.NoError(t, err)
	defer resp.Body.Close()
	require.EqualValues(t, http.StatusOK, resp.StatusCode, "the login must succeed")

	// get licenses of logged-in user
	resp, err = httpClient.Get(makeTestURL(ts.URL, "/api/account/licenses"))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.EqualValues(t, http.StatusOK, resp.StatusCode, "a license request with a valid session must succeed")

	var licenseResponse LicensesResponse
	err = json.NewDecoder(resp.Body).Decode(&licenseResponse)
	require.NoError(t, err)

	expected := LicensesResponse{LicenseTokens: []string{licenseAcct1.Token}}
	require.EqualValues(t, expected, licenseResponse, "the license request must return a slice of strings")
}

func Test_LicensesWithAnon(t *testing.T) {
	ts, _, mgr, _ := makeTestServer(t)
	defer ts.Close()
	defer requireCleanup(t, mgr)

	authority, err := licensing.NewTestAuthority()
	require.NoError(t, err)

	jar, err := cookiejar.New(nil)
	httpClient := ts.Client()
	httpClient.Jar = jar

	// Anon only
	anonUser := community.AnonUser{InstallID: "test-anon-user-0"}
	requireInsertTestLicense(t, authority, mgr, &anonUser)
	u := makeTestURL(ts.URL, "/api/account/licenses?install-id="+anonUser.AnonID())
	resp := doTestRequest(t, httpClient, "GET", u, nil)
	require.True(t, resp.StatusCode < 400, fmt.Sprintf("GET Licenses failed, returned code %d", resp.StatusCode))
	licenses := licenseResponseFromResponse(t, resp)
	assert.EqualValues(t, 1, len(licenses.LicenseTokens), "Did not get back singular license")

	// Both anon and user in a single request
	u1, _ := requireCreateUserSession(t, mgr, "b@c.d")
	resp, err = httpClient.PostForm(makeTestURL(ts.URL, "/api/account/login-web"), url.Values{
		"email":    []string{u1.Email},
		"password": []string{"password123!"},
	})
	requireInsertTestLicense(t, authority, mgr, u1)
	anonUser = community.AnonUser{InstallID: "test-anon-user-1"}
	requireInsertTestLicense(t, authority, mgr, &anonUser)

	resp, err = httpClient.Get(makeTestURL(ts.URL, "/api/account/licenses?install-id="+anonUser.AnonID()))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.EqualValues(t, http.StatusOK, resp.StatusCode, "a license request with a valid session must succeed")
	licenses = licenseResponseFromResponse(t, resp)
	assert.EqualValues(t, 2, len(licenses.LicenseTokens), "Failed to get both anon and user licenses in a single api call")
}

func requireInsertTestLicense(t *testing.T, authority *licensing.Authority, mgr *manager, ids community.UserIdentifier) {
	license, err := authority.CreateLicense(licensing.Claims{
		UserID:    ids.UserIDString(),
		InstallID: ids.AnonID(),
		Product:   licensing.Pro,
		Plan:      licensing.ProTrial,
		ExpiresAt: time.Now().Add(defaultTrialDuration),
	})
	require.NoError(t, err)
	requireCreateLicense(t, mgr, ids, *license)
}

func requireCreateLicense(t *testing.T, mgr *manager, ids community.UserIdentifier, license licensing.License) license {
	dbLicense, err := mgr.CreateLicense(ids, license, "ref")
	require.EqualValues(t, nil, err)
	require.EqualValues(t, license.Token, dbLicense.Token)

	return *dbLicense
}
