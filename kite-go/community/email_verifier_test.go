package community

import (
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestEmailVerificationManager() EmailVerifier {
	db := DB("sqlite3", ":memory:")
	db.LogMode(logOut)
	manager := newEmailVerificationManager(db)
	manager.Migrate()
	return manager
}

func requireCleanupEmailVerificationManager(t *testing.T, manager EmailVerifier) {
	if m, ok := manager.(*emailVerificationManager); ok {
		require.NoError(t, m.db.Close())
	}
}

func Test_VerificationCreate(t *testing.T) {
	emailVerifier := makeTestEmailVerificationManager()
	defer requireCleanupEmailVerificationManager(t, emailVerifier)

	verification, err := emailVerifier.Create("fred@example.com")
	assert.NoError(t, err, "failed to create verification")

	assert.Equal(t, "fred@example.com", verification.Email)
	assert.NotEmpty(t, verification.Code, "verification code should not be empty")
	assert.True(t, verification.Expiration.After(time.Now()), "expiration should occur after the present")
}

func Test_VerificationLookup(t *testing.T) {
	emailVerifier := makeTestEmailVerificationManager()
	defer requireCleanupEmailVerificationManager(t, emailVerifier)

	vCreate, err := emailVerifier.Create("fred@example.com")
	assert.NoError(t, err, "failed to create verification")

	_, err = emailVerifier.Lookup(vCreate.Email, "wrong code")
	assert.Equal(t, ErrVerificationInvalid, err, "verification should be invalid for wrong code")
	_, err = emailVerifier.Lookup("wrong email", vCreate.Code)
	assert.Equal(t, ErrVerificationInvalid, err, "verification should be invalid for wrong email")
	_, err = emailVerifier.Lookup("wrong email", "wrong code")
	assert.Equal(t, ErrVerificationInvalid, err, "verification should be invalid for wrong email and code")

	vLookup, err := emailVerifier.Lookup(vCreate.Email, vCreate.Code)
	assert.NoError(t, err, "failed to look up verification")
	assert.Equal(t, vCreate.Email, vLookup.Email)
	assert.Equal(t, vCreate.Code, vLookup.Code)
	assert.Equal(t, vCreate.Expiration.Unix(), vLookup.Expiration.Unix())
}

func Test_VerificationRemove(t *testing.T) {
	emailVerifier := makeTestEmailVerificationManager()
	defer requireCleanupEmailVerificationManager(t, emailVerifier)

	v, err := emailVerifier.Create("fred@example.com")
	assert.NoError(t, err, "failed to create verification")

	v, err = emailVerifier.Lookup(v.Email, v.Code)
	assert.NoError(t, err, "failed to look up verification")

	err = emailVerifier.Remove(v)
	assert.NoError(t, err, "failed to remove verification")

	_, err = emailVerifier.Lookup(v.Email, v.Code)
	assert.Equal(t, ErrVerificationInvalid, err, "verification should be invalid after removal")
}
