package account

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/community/student"
	"github.com/kiteco/kiteco/kite-go/web"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	logOut bool

	cc = card{
		Last4: "1234",
		Brand: "kite",
	}
)

func init() {
	flag.BoolVar(&logOut, "logout", false, "turn on logging")
	web.VerboseLog = logOut
}

// add ?sslmode=disable for testing on a local machine
const testDBPath = "postgres://XXXXXXX:XXXXXXX@localhost/account_test?sslmode=disable"

func requireManager(t *testing.T) *manager {
	app := makeTestApp(t)

	return makeManager(t, app.DB, app.Users)
}

// makeTestApp returns a new application, domain valid-school.com is whitelisted fro educational licenses.
func makeTestApp(t *testing.T) *community.App {
	db := community.DB("postgres", testDBPath)
	db.LogMode(logOut)
	db.DropTableIfExists(&community.User{})
	db.DropTableIfExists(&community.Session{})
	db.DropTableIfExists(&community.PasswordReset{})
	db.DropTableIfExists(&community.EmailVerification{})
	db.DropTableIfExists(&community.Signup{})
	db.DropTableIfExists(&community.Download{})
	db.DropTableIfExists(&community.Nonce{})

	app := community.NewApp(db, community.NewSettingsManager(), &student.DomainLists{
		WhiteList: map[string]struct{}{
			"valid-school.com": struct{}{},
		},
	})
	err := app.Migrate()
	require.NoError(t, err)

	emailVerifier := &mockEmailVerifier{}
	app.EmailVerifier = emailVerifier

	return app
}

func makeTestServer(t *testing.T) (*httptest.Server, *Server, *manager, *community.App) {
	app := makeTestApp(t)

	// instead of using NewServer, we reimplement so that we can use
	// mockStripeManager in both the manager and stripeWebhook
	manager := makeManager(t, app.DB, app.Users)
	server := &Server{
		app: app,
		m:   manager,
	}
	InitStripe("fake_secret", "another_fake_secret",
		"real_secret", "that_s_so_strange", "oh_oh_oh")
	mux := mux.NewRouter().PathPrefix("/api/account").Subrouter()
	server.SetupRoutes(mux)

	httpServer := httptest.NewServer(mux)

	return httpServer, server, manager, app
}

func makeManager(t *testing.T, db gorm.DB, users *community.UserManager) *manager {
	require.NoError(t, db.DropTableIfExists(&account{}).Error)
	require.NoError(t, db.DropTableIfExists(&member{}).Error)
	require.NoError(t, db.DropTableIfExists(&subscription{}).Error)
	require.NoError(t, db.DropTableIfExists(&license{}).Error)
	authority, err := licensing.NewTestAuthority()
	require.NoError(t, err)
	mgr := newManager(db, users, authority)
	require.NoError(t, mgr.Migrate())

	return mgr
}

func makeTestURL(base, endpoint string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		log.Fatal(err)
	}

	endpointURL, err := baseURL.Parse(endpoint)
	if err != nil {
		log.Fatal(err)
	}

	return endpointURL.String()
}

func requireCleanup(t *testing.T, mgr *manager) {
	require.NoError(t, mgr.db.Close())
}

func requireCreateUserSession(t *testing.T, mgr *manager, email string) (*community.User, *community.Session) {

	user, session, err := mgr.users.Create("", email, "password123!", "")

	require.NoError(t, err)
	return user, session
}

func requireStartTrialNoErr(t *testing.T, mgr *manager, user community.UserIdentifier) {
	err := mgr.StartTrial(user, defaultTrialDuration, "", "", user.UserID() != 0, 0)
	require.NoError(t, err)
}

func licenseInfoFromResp(t *testing.T, resp *http.Response) *licensing.LicenseInfo {
	var l licensing.LicenseInfo
	d := json.NewDecoder(resp.Body)
	err := d.Decode(&l)
	if err != nil {
		assert.Fail(t, "Could not decode license info")
		return nil
	}
	return &l
}

func licenseResponseFromResponse(t *testing.T, resp *http.Response) *LicensesResponse {
	var l LicensesResponse
	err := json.NewDecoder(resp.Body).Decode(&l)
	if err != nil {
		assert.Fail(t, "Could not decode license info")
		return nil
	}
	return &l
}

// Does not currently handle body
func doTestRequest(t *testing.T, c *http.Client, method, url string, headers http.Header) *http.Response {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal("Bad test request", err)
	}
	for key, vals := range headers {
		req.Header.Set(key, vals[0])
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal("Bad test request", err)
	}
	return resp
}

func TestManager_User(t *testing.T) {
	mgr := requireManager(t)
	defer requireCleanup(t, mgr)

	// trigger error, no cookie set auth fails
	r, err := http.NewRequest("POST", "http://someurl.com", nil)
	require.NoError(t, err)

	user, err := mgr.users.IsValidLogin(r)
	assert.Nil(t, user)
	assert.NotNil(t, err)

	// create user and set session coookie

	user, session, err := mgr.users.Create("", "e@mail.com", "password123!", "")

	require.NoError(t, err)
	require.NotNil(t, session)
	require.NotNil(t, user)

	cookie := &http.Cookie{
		Name:    "kite-session",
		Value:   session.Key,
		Path:    "/",
		Expires: session.ExpiresAt,
	}
	r.AddCookie(cookie)

	user1, err := mgr.users.IsValidLogin(r)
	require.NotNil(t, user1)
	assert.Nil(t, err)
	assert.Equal(t, user.ID, user1.ID)
}

func TestRequireStartTrialNoErrTimeZone(t *testing.T) {
	mgr := requireManager(t)
	defer requireCleanup(t, mgr)

	user, _, err := mgr.users.Create("", "e@mail.com", "password123!", "")
	require.NoError(t, err)

	offset := -6 * 60 * 60
	clientZone := time.FixedZone("UTC-6", offset)
	nowClient := time.Now().In(clientZone)

	err = mgr.StartTrial(user, 28*24*time.Hour, "", "", user.UserID() != 0, offset)
	require.NoError(t, err)

	licenses, err := mgr.Licenses(user)
	require.NoError(t, err)
	require.EqualValues(t, 1, licenses.Len())

	license := licenses.License()
	expirationClientTime := license.ExpiresAt.In(clientZone)
	// 12:01 am, same day or the day after
	require.EqualValues(t, 0, expirationClientTime.Hour())
	require.EqualValues(t, 0, expirationClientTime.Minute())
	require.EqualValues(t, 1, expirationClientTime.Second())

	o := expirationClientTime.Sub(nowClient.AddDate(0, 0, 28))
	require.True(t, o >= -24*time.Hour && o <= 24*time.Hour, "license must expire 12:00:01 am in the local time of a client")
}

func TestManager_StartTrialActive(t *testing.T) {
	mgr := requireManager(t)
	defer requireCleanup(t, mgr)

	u0, _ := requireCreateUserSession(t, mgr, "a@b.c")
	var users = []community.UserIdentifier{
		u0,
		&community.AnonUser{InstallID: "test-id"},
	}

	for _, user := range users {
		requireStartTrialNoErr(t, mgr, user)
		requireStartTrialNoErr(t, mgr, user)
		ls, err := mgr.license.Licenses(user)
		assert.NoError(t, err)
		l := ls.License()
		assert.True(t, !l.IsExpired() && l.GetPlan() == licensing.ProTrial)
	}
}

func TestManager_StartTrialConsumed(t *testing.T) {
	mgr := requireManager(t)
	defer requireCleanup(t, mgr)

	u0, _ := requireCreateUserSession(t, mgr, "a@b.c")
	var users = []community.UserIdentifier{
		u0,
		&community.AnonUser{InstallID: "test-id"},
	}

	reset := setTrialDuration(time.Millisecond)
	defer reset()

	for _, user := range users {
		requireStartTrialNoErr(t, mgr, user)
		time.Sleep(time.Millisecond)
		err := mgr.StartTrial(user, defaultTrialDuration, "", "", user.UserID() != 0, 0)
		assert.Equal(t, errTrialAlreadyConsumed, err)
	}
}

func TestServer_HandleAPIStartTrial(t *testing.T) {
	ts, _, mgr, _ := makeTestServer(t)
	defer ts.Close()
	defer requireCleanup(t, mgr)

	u0, s0 := requireCreateUserSession(t, mgr, "a@b.c")
	u1, s1 := requireCreateUserSession(t, mgr, "b@c.d")
	depriorIID := "anon-and-user-test"

	var tests = []*struct {
		user    *community.User
		session *community.Session
		iid     string
	}{
		{
			iid: "anon-test-handle-api-start-trial",
		},
		{
			user:    u0,
			session: s0,
		},
		{
			iid:     depriorIID,
			user:    u1,
			session: s1,
		},
	}

	c := ts.Client()
	for i, test := range tests {
		var headers http.Header
		if test.user != nil && test.session != nil {
			headers = community.HmacHeadersFromUserSession(test.user, test.session)
		}

		// Start trial without special params should start a trial of default length
		u := makeTestURL(ts.URL, fmt.Sprintf("/api/account/start-trial?install-id=%s", test.iid))
		resp := doTestRequest(t, c, "POST", u, headers)
		assert.True(t, resp.StatusCode < 400, fmt.Sprintf("Trial did not start, returned code %d for test %d", resp.StatusCode, i))

		u = makeTestURL(ts.URL, fmt.Sprintf("/api/account/license-info?install-id=%s", test.iid))
		resp = doTestRequest(t, c, "GET", u, headers)
		require.True(t, resp.StatusCode < 400, fmt.Sprintf("License info did not succeed, returned code %d for test %d", resp.StatusCode, i))

		l := licenseInfoFromResp(t, resp)
		if l != nil {
			assert.EqualValues(t, defaultTrialDuration.Hours()/24, l.DaysRemaining, "First trial is not default length")
		}
	}

	// For both anon and user included in a request, start-trial should start the trial for the user only.
	u := makeTestURL(ts.URL, fmt.Sprintf("/api/account/license-info?install-id=%s", depriorIID))
	resp := doTestRequest(t, c, "GET", u, nil)
	require.True(t, resp.StatusCode < 400, fmt.Sprintf("License info did not succeed, returned code %d", resp.StatusCode))
	l := licenseInfoFromResp(t, resp)
	if l != nil {
		assert.EqualValues(t, l.Plan, licensing.FreePlan, "Expected free license for anon user when anon id is included with a user")
	}

	u = makeTestURL(ts.URL, "/api/account/license-info")
	headers := community.HmacHeadersFromUserSession(u1, s1)
	resp = doTestRequest(t, c, "GET", u, headers)
	require.True(t, resp.StatusCode < 400, fmt.Sprintf("License info did not succeed, returned code %d", resp.StatusCode))
	l = licenseInfoFromResp(t, resp)
	if l != nil {
		assert.EqualValues(t, l.Plan, licensing.ProTrial, "Expect pro license for user when user is included with an anon id")
	}

	// Allow custom duration passed in as a parameter.
	sixWeekInHours := time.Hour * 24 * 7 * 6
	cdurid := "custom-duration-anon-id"
	u = makeTestURL(ts.URL, fmt.Sprintf("/api/account/start-trial?install-id=%s&trial-duration=%s", cdurid, sixWeekInHours.String()))
	resp = doTestRequest(t, c, "POST", u, nil)
	assert.True(t, resp.StatusCode < 400, fmt.Sprintf("Trial did not start, returned code %d for test %s", resp.StatusCode, cdurid))

	u = makeTestURL(ts.URL, fmt.Sprintf("/api/account/license-info?install-id=%s", cdurid))
	resp = doTestRequest(t, c, "GET", u, nil)
	require.True(t, resp.StatusCode < 400, fmt.Sprintf("License info did not succeed, returned code %d", resp.StatusCode))
	l = licenseInfoFromResp(t, resp)
	if l != nil {
		assert.EqualValues(t, l.DaysRemaining, sixWeekInHours.Hours()/24, "Expected license duration to match passed query duration")
	}

	cdurid = "attempt-long-trial-anon-id"
	hundredDays := time.Hour * 24 * 100
	u = makeTestURL(ts.URL, fmt.Sprintf("/api/account/start-trial?install-id=%s&trial-duration=%s", cdurid, hundredDays.String()))
	resp = doTestRequest(t, c, "POST", u, nil)
	require.True(t, resp.StatusCode < 400, fmt.Sprintf("Trial did not start, returned code %d for test %s", resp.StatusCode, hundredDays))

	u = makeTestURL(ts.URL, fmt.Sprintf("/api/account/license-info?install-id=%s", cdurid))
	resp = doTestRequest(t, c, "GET", u, nil)
	require.True(t, resp.StatusCode < 400, fmt.Sprintf("License info did not succeed, returned code %d", resp.StatusCode))
	l = licenseInfoFromResp(t, resp)
	if l != nil {
		assert.EqualValues(t, l.DaysRemaining, 90, "Expected trial duration to cap at 90 days")
	}

}

func TestManager_AutomaticEducationalLicense(t *testing.T) {
	mgr := requireManager(t)
	defer requireCleanup(t, mgr)

	// fetching licenses for a new user automatically adds an educational license
	eduUser, _ := requireCreateUserSession(t, mgr, "kite-student@valid-school.com")
	licenses, err := mgr.Licenses(eduUser)
	require.NoError(t, err)
	require.EqualValues(t, 1, licenses.Len())
	license := licenses.License()
	require.True(t, license.IsPlanActive())
	require.EqualValues(t, licensing.ProEducation, license.Plan)

	// fetching licenses twice isn't adding a 2nd educational license
	licenses, err = mgr.Licenses(eduUser)
	require.NoError(t, err)
	require.EqualValues(t, 1, licenses.Len())

	// no educational license for regular users
	regularUser, _ := requireCreateUserSession(t, mgr, "kite-user@not-a-school.com")
	licenses, err = mgr.Licenses(regularUser)
	require.NoError(t, err)
	require.EqualValues(t, 0, licenses.Len())
}

func setTrialDuration(d time.Duration) func() {
	x := defaultTrialDuration

	defaultTrialDuration = d

	return func() {
		defaultTrialDuration = x
	}
}
