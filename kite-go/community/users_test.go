package community

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type userTestCase struct {
	user     string
	password string
	email    string
	valid    bool
}

func Test_NewUserValidation(t *testing.T) {
	longPassword := strings.Repeat("long", 15)
	testCases := []userTestCase{
		userTestCase{"fred", "password", "fred1@example.com", true},   // Good user
		userTestCase{"fred", "password", "fred1example.com", false},   // Bad email
		userTestCase{"fred", "short", "fred1example.com", false},      // Short password
		userTestCase{"fred", longPassword, "fred1example.com", false}, // Long password
	}

	for _, test := range testCases {
		user, err := NewUser(test.user, test.email, test.password)
		switch {
		case test.valid && err != nil:
			t.Errorf("expected %s:%s to be valid. returned invalid", test.user, test.email)
		case !test.valid && err == nil:
			t.Errorf("expected %s:%s to be invalid, returned valid", test.user, test.email)
		}

		if test.valid && err == nil {
			checkUser(test, user, t)
		}
	}
}

func Test_UserCreatePasswordless(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	testCases := []userTestCase{
		userTestCase{"fred1", "", "fred1@example.com", true}, // Everythings good
		userTestCase{"fred2", "", "fred2example.com", false}, // Bad email
	}

	for _, test := range testCases {
		_, _, err := app.Users.CreatePasswordless(test.email, "", true)
		switch {
		case test.valid && err != nil:
			t.Error("expected", test, "creation to be valid. returned invalid")
		case !test.valid && err == nil:
			t.Error("expected", test, "creation to be invalid, returned valid")
		}

		_, _, err = app.Users.CreatePasswordless(test.email, "", true)
		if err == nil {
			t.Error("expected", test, "second creation to fail, but succeeded")
		}
	}
}

func Test_UserCreateAndLogin(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	longPassword := strings.Repeat("long", 15)
	testCases := []userTestCase{
		userTestCase{"fred1", "goodpassword", "fred1@example.com", true}, // Everythings good
		userTestCase{"fred2", "short", "fred2@example.com", false},       // Short password
		userTestCase{"fred3", longPassword, "fred3@example.com", false},  // Long password
		userTestCase{"fred4", "goodpassword", "fred4example.com", false}, // Bad email
	}

	for _, test := range testCases {
		_, _, err := app.Users.Create(test.user, test.email, test.password, "")
		switch {
		case test.valid && err != nil:
			t.Error("expected", test, "creation to be valid. returned invalid")
		case !test.valid && err == nil:
			t.Error("expected", test, "creation to be invalid, returned valid")
		}

		_, _, err = app.Users.Create(test.user, test.email, test.password, "")
		if err == nil {
			t.Error("expected", test, "second creation to fail, but succeeded")
		}

		_, _, err = app.Users.Login(test.email, test.password)
		switch {
		case test.valid && err != nil:
			t.Error("expected", test, "login to be valid. returned:", err)
		case !test.valid && err == nil:
			t.Error("expected", test, "login to be invalid, returned valid")
		}

		_, _, err = app.Users.Login(test.email, strings.Repeat("wrong", 2))
		if err == nil {
			t.Error("expected", test, "wrong password to fail, but succeeded")
		}
	}
}

func Test_UserCaseInsensitiveLogin(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	_, _, err := app.Users.Create("fred", "fred@example.com", "password", "")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = app.Users.Login("FrEd@eXamPlE.com", "password")
	assert.NoError(t, err)
}

func Test_UserSessions(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	test := userTestCase{"fred", "goodpassword", "fred@example.com", true}
	// Create a user...
	user1, session1, err := app.Users.Create(test.user, test.email, test.password, "")

	if err != nil {
		t.Fatal("expected user", test, "to create successfully")
	}

	checkUser(test, user1, t)

	// Check the session we just created
	user2, _, err := app.Users.ValidateSession(string(session1.Key))
	if err != nil {
		t.Fatal("expected session to validate, got:", err)
	}

	checkUser(test, user2, t)

	// Login, should return a new session..
	user3, session2, err := app.Users.Login(test.email, test.password)
	if err != nil {
		t.Fatal("expected successfull login, got:", err)
	}

	checkUser(test, user3, t)

	// Make sure we have a new session..
	if session1.Key == session2.Key {
		t.Fatal("expected two differrent sessions, but both equal to:", session1.Key)
	}

	// Check the both sessions to make sure they validate. Then, log out of each session
	// and make sure they no longer validate.
	for _, session := range []Session{*session1, *session2} {
		user, _, err := app.Users.ValidateSession(string(session.Key))
		if err != nil {
			t.Fatal("expected session to validate, got:", err)
		}
		checkUser(test, user, t)

		err = app.Users.Logout(string(session.Key))
		if err != nil {
			t.Fatal("expected logout to succeed, got:", err)
		}

		_, _, err = app.Users.ValidateSession(string(session.Key))
		if err == nil {
			t.Fatal("expected session to not validate, but succeeded")
		}
	}
}

func Test_CreatePasswordReset(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	user, _, err := app.Users.Create("fred", "fred@example.com", "goodpassword", "")

	assert.NoError(t, err)
	reset, err := app.Users.CreatePasswordReset(user)
	assert.NoError(t, err)

	assert.NotNil(t, reset, "should have sent password reset email")

	var r PasswordReset
	assert.False(t, app.Users.db.Where(PasswordReset{Token: reset.Token}).First(&r).RecordNotFound(),
		"should have saved password reset to db")
}

func Test_PasswordResetUser(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	createdUser, _, err := app.Users.Create("fred", "fred@example.com", "goodpassword", "")

	assert.NoError(t, err)
	createdReset, err := app.Users.CreatePasswordReset(createdUser)
	assert.NoError(t, err)

	lookupUser, lookupReset, err := app.Users.PasswordResetUser(createdReset.Token)
	if err != nil {
		t.Fatalf("expected Users.PasswordResetUser to succeed; it failed with error: %s", err)
	}
	assert.NoError(t, err)

	assert.Equal(t, createdUser.ID, lookupUser.ID, "user returned by Users.PasswordResetUser should match that returned by Users.Create")
	assert.Equal(t, createdReset.ID, lookupReset.ID)
	assert.Equal(t, createdReset.UserID, lookupReset.UserID)
	assert.Equal(t, createdReset.Token, lookupReset.Token)
	assert.Equal(t, createdReset.Expiration.Unix(), lookupReset.Expiration.Unix())
}

func Test_PerformPasswordReset(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	createdUser, _, err := app.Users.Create("fred", "fred@example.com", "oldpassword", "")

	assert.NoError(t, err)
	reset, err := app.Users.CreatePasswordReset(createdUser)
	assert.NoError(t, err)

	_, err = app.Users.PerformPasswordReset("bad token", "newpassword")
	assert.Equal(t, ErrInvalidReset, err)

	_, err = app.Users.PerformPasswordReset(reset.Token, "newpassword")
	assert.NoError(t, err, "failed to reset password, got error: %s", err)

	loginUser, _, err := app.Users.Login("fred@example.com", "newpassword")
	if err != nil {
		t.Fatalf("expected Users.Login to succeed; it failed with error: %s", err)
	}
	assert.Equal(t, createdUser.ID, loginUser.ID)
}

func Test_UserList(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	testCases := []userTestCase{
		userTestCase{"fred1", "goodpassword", "fred1@example.com", true},
		userTestCase{"fred2", "goodpassword", "fred2@example.com", true},
		userTestCase{"fred3", "goodpassword", "fred3@example.com", true},
		userTestCase{"fred4", "goodpassword", "fred4@example.com", true},
	}

	// Create a few users and put them in a map, keyed by email
	userMap := make(map[string]*User)
	for _, test := range testCases {
		user, _, err := app.Users.Create(test.user, test.email, test.password, "")
		switch {
		case test.valid && err != nil:
			t.Error("expected", test, "creation to be valid. returned invalid")
		case !test.valid && err == nil:
			t.Error("expected", test, "creation to be invalid, returned valid")
		}

		userMap[user.Email] = user
	}

	// List users
	users, err := app.Users.List()
	if err != nil {
		t.Fatal("expected List to succeed")
	}

	// Make sure returned users are in the map. If they are, nil them out.
	for _, user := range users {
		_, exists := userMap[user.Email]
		if !exists {
			t.Fatal("got unexpected user", user.Email)
		}
		userMap[user.Email] = nil
	}

	// Any remaining users (non-nil) in userMap are users that were not returned.
	for _, user := range userMap {
		if user != nil {
			t.Fatal("list did not return entry for user", user.Email)
		}
	}
}

func Test_Unsubscribe(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	user, _, err := app.Users.Create("test", "test@test.com", "testpassword", "")
	require.Nil(t, err, "user creation failed")
	require.NotNil(t, user, "user should not be nil")
	assert.Equal(t, user.Unsubscribed, false, "unsubscribed should be false")

	user, err = app.Users.Unsubscribe("test@test.com")
	require.Nil(t, err, "unsubscribe failed")
	assert.Equal(t, user.Unsubscribed, true, "unsubscribed should be true")

	user, err = app.Users.Unsubscribe("test@test.com")
	require.Nil(t, err, "unsubscribe failed")
	assert.Equal(t, user.Unsubscribed, true, "unsubscribed should be true")
}

func Test_Subscribe(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	user, _, err := app.Users.Create("test", "test@test.com", "testpassword", "")
	require.Nil(t, err, "user creation failed")
	require.NotNil(t, user, "user should not be nil")
	assert.Equal(t, user.Unsubscribed, false, "unsubscribed should be false")

	user, err = app.Users.Unsubscribe("test@test.com")
	require.Nil(t, err, "unsubscribe failed")
	assert.Equal(t, user.Unsubscribed, true, "unsubscribed should be true")

	user, err = app.Users.Subscribe("test@test.com")
	require.Nil(t, err, "subscribe failed")
	assert.Equal(t, user.Unsubscribed, false, "unsubscribed should be false")

	user, err = app.Users.Subscribe("test@test.com")
	require.Nil(t, err, "subscribe failed")
	assert.Equal(t, user.Unsubscribed, false, "unsubscribed should be false")
}

func Test_RecordVerifiedEmail(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	user, _, err := app.Users.Create("fred", "fred@example.com", "goodpassword", "")
	assert.NoError(t, err, "user creation should have succeeded")
	err = app.Users.recordVerifiedEmail("fred@example.com")
	assert.NoError(t, err, "recordVerifiedEmail should have succeeded")

	user, _, err = app.Users.Login("fred@example.com", "goodpassword")
	assert.NoError(t, err, "logging in should have succeeded")
	assert.True(t, user.EmailVerified, "user.EmailVerified should be true after calling recordVerifiedEmail")
}

func Test_UserChangePassword(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	_, _, err := app.Users.Create("testuser", "test@email.com", "password1", "")
	if err != nil {
		t.Error("expected user creation to be valid. returned invalid")
	}

	_, _, err = app.Users.Login("test@email.com", "password1")
	if err != nil {
		t.Error("expected login to be valid. returned:", err)
	}

	err = app.Users.UpdatePassword("test@email.com", "password2")
	if err != nil {
		t.Error("expected change password to be valid. returned:", err)
	}

	_, _, err = app.Users.Login("test@email.com", "password1")
	if err == nil {
		t.Error("expected login with old password to be invalid. returned valid.")
	}

	_, _, err = app.Users.Login("test@email.com", "password2")
	if err != nil {
		t.Error("expected login with new password to be valid. returned:", err)
	}

	err = app.Users.UpdatePassword("test@email.com", "short")
	if err == nil {
		t.Error("expected change password to fail due to short password, but it succeeded")
	}

	err = app.Users.UpdatePassword("test@email.com", "toolongtoolongtoolongtoolongtoolongtoolongtoolongtoolongtoolong")
	if err == nil {
		t.Error("expected change password to fail due to long password, but it succeeded")
	}
}

func Test_CheckEmail(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)
	// test invalid email address
	assert.NotNil(t, app.Users.CheckEmail("badbad.com"), "expected CheckEmail to fail with bad email address badbad.com")

	_, _, err := app.Users.Create("test", "test@email.com", "password1", "")
	require.NoError(t, err, "error creating test user: %v", err)

	// test existing email address
	assert.NotNil(t, app.Users.CheckEmail("test@email.com"), "expected CheckEmail to fail with existing email address")

	// test new email
	assert.Nil(t, app.Users.CheckEmail("test1@email.com"), "expected CheckEmail to succeed with new valid email address")
}

// --

func checkUser(test userTestCase, user *User, t *testing.T) {
	if test.user != user.Name {
		t.Errorf("Username mismatch: expected %s, got %s", test.user, user.Name)
	}
	if test.email != user.Email {
		t.Errorf("Email mismatch: expected %s, got %s", test.email, user.Email)
	}
}
