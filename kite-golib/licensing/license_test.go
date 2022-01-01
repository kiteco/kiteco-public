package licensing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Workflow(t *testing.T) {
	mgr, err := NewTestAuthority()
	require.NoError(t, err)

	expiration := time.Now().Add(10 * time.Minute)
	license, err := mgr.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: expiration,
	})
	require.NoError(t, err)
	require.EqualValues(t, "user@example.com", license.UserID)

	err = license.Valid()
	require.NoError(t, err, "The token claims must be valid")

	require.False(t, license.ExpiresAt.IsZero())
	require.False(t, license.IssuedAt.IsZero())
	require.True(t, license.IssuedAt.Before(license.ExpiresAt))
	require.EqualValues(t, expiration, license.ExpiresAt)

	validator := mgr.CreateValidator()
	validatedLicense, err := validator.Parse(license.Token)
	require.NoError(t, err)
	require.NotNil(t, validatedLicense)

	// validate expiration with a granularity of 1s
	require.EqualValues(t, expiration.Unix(), validatedLicense.ExpiresAt.Unix())
}

func Test_WrongValidationKey(t *testing.T) {
	mgr, err := NewTestAuthority()
	require.NoError(t, err)
	now := time.Now()
	license, err := mgr.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: now.Add(10 * time.Minute),
	})
	require.NoError(t, err)

	validator, err := newTestValidator()
	require.NoError(t, err)
	_, err = validator.Parse(license.Token)
	require.Error(t, err, "validation using a public key, which isn't matching the private key, has to fail")
}

func Test_ExpiredLicense(t *testing.T) {
	mgr, err := NewTestAuthority()
	require.NoError(t, err)

	now := time.Now()
	// this license expires very soon
	license, err := mgr.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: now.Add(100 * time.Millisecond),
	})
	require.NoError(t, err)
	// wait until license is expired, the granularity of the timestamps is 1s
	time.Sleep(2 * time.Second)

	validator := mgr.CreateValidator()
	license, err = validator.Parse(license.Token)
	require.NoError(t, err, "validation must not fail for an expired license")
	require.True(t, license.IsExpired())
}

func Test_InvalidExpiration(t *testing.T) {
	mgr, err := NewTestAuthority()
	require.NoError(t, err)
	now := time.Now()
	_, err = mgr.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: now.Add(-1 * time.Minute),
	})
	require.Error(t, err, "creating a license with expiration in the past has to fail")
}

func Test_MissingUserID(t *testing.T) {
	mgr, err := NewTestAuthority()
	require.NoError(t, err)
	now := time.Now()
	_, err = mgr.CreateLicense(Claims{
		UserID:    "",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: now.Add(1 * time.Hour),
	})
	require.Error(t, err, "creating a license without a user id has to fail")
}

func Test_MissingProduct(t *testing.T) {
	mgr, err := NewTestAuthority()
	require.NoError(t, err)
	now := time.Now()
	_, err = mgr.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Product(""),
		Plan:      ProMonthly,
		ExpiresAt: now.Add(1 * time.Hour),
	})
	require.Error(t, err, "creating a license without a product has to fail")
}

func Test_MissingPlan(t *testing.T) {
	mgr, err := NewTestAuthority()
	require.NoError(t, err)
	now := time.Now()
	_, err = mgr.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      Plan(""),
		ExpiresAt: now.Add(1 * time.Hour),
	})
	require.Error(t, err, "creating a license without a plan has to fail")
}

func Test_MissingSubscription(t *testing.T) {
	mgr, err := NewTestAuthority()
	require.NoError(t, err)
	now := time.Now()
	_, err = mgr.CreateLicense(Claims{
		Product:   "pro",
		UserID:    "user@example.com",
		Plan:      "",
		ExpiresAt: now.Add(1 * time.Hour),
	})
	require.Error(t, err, "creating a license without a subscription has to fail")
}
