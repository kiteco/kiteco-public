package community

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_CreateRedeemNonce(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	user, session, err := app.Users.Create("fred1", "fred1@example.com", "goodpassword", "")

	require.Nil(t, err, "expected user to be created successfully")

	nonce, err := app.Users.CreateNonceForUser(user.ID, "machine1")
	require.Nil(t, err, "expected nonce for user to be created succcessfully")
	require.Equal(t, user.ID, nonce.UserID, "expected userid to match nonce.userid")
	require.Equal(t, "machine1", nonce.MachineID, "expected machineid to match nonce.machineid")
	require.True(t, nonce.ID > 0, "expected nonce id to be > 0")

	user1, session1, machine1, err := app.Users.RedeemNonce(nonce.Value)
	require.Nil(t, err, "expected redeem to return successfully")
	require.Equal(t, user.Email, user1.Email)
	require.Equal(t, "machine1", machine1)
	require.NotEqual(t, session.Key, session1.Key, "expect redeem to generate new session")

	_, _, _, err = app.Users.RedeemNonce(nonce.Value)
	require.NotNil(t, err, "expected error redeeming nonce twice")
	require.Equal(t, errNonceNotFound, err)
}

func Test_ExpiredNonce(t *testing.T) {
	app := makeTestApp()
	defer requireCleanupApp(t, app)

	user, _, err := app.Users.Create("fred1", "fred1@example.com", "goodpassword", "")

	require.Nil(t, err, "expected user to be created successfully")

	nonce, err := app.Users.CreateNonceForUser(user.ID, "machine1")
	require.Nil(t, err, "expected nonce for user to be created succcessfully")
	require.Equal(t, user.ID, nonce.UserID, "expected userid to match nonce.userid")
	require.Equal(t, "machine1", nonce.MachineID, "expected machineid to match nonce.machineid")
	require.True(t, nonce.ID > 0, "expected nonce id to be > 0")

	nonce.ExpiresAt = time.Now()
	err = app.Users.db.Save(nonce).Error
	require.Nil(t, err, "unable to save nonce with test expiration")

	time.Sleep(time.Second)

	_, _, _, err = app.Users.RedeemNonce(nonce.Value)
	require.NotNil(t, err, "expected redeem to fail")
	require.Equal(t, errNonceExpired, err)
}
