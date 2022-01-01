package licensing

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Lifecycle(t *testing.T) {
	authority, err := NewTestAuthority()
	require.NoError(t, err)
	now := time.Now()
	license1SameUser, err := authority.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: now.Add(1 * time.Hour),
	})
	require.NoError(t, err)

	license2SameUser, err := authority.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: now.Add(10 * time.Minute),
	})
	require.NoError(t, err)

	license3TrialSameUser, err := authority.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProTrial,
		ExpiresAt: now.Add(10 * time.Minute),
	})
	require.NoError(t, err)

	licenseOtherUser, err := authority.CreateLicense(Claims{
		UserID:    "other_user@example.com",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: now.Add(1 * time.Hour),
	})
	require.NoError(t, err)

	store := NewStore(authority.CreateValidator(), "")
	store.SetUserID("user@example.com")
	require.False(t, store.hasHadPro)

	err = store.Add(license1SameUser.Token)
	require.NoError(t, err)
	require.EqualValues(t, 1, store.Size())
	require.True(t, store.hasHadPro)

	err = store.Add(license2SameUser.Token)
	require.NoError(t, err)
	require.EqualValues(t, 2, store.Size())
	require.EqualValues(t, "user@example.com", store.UserID())
	require.True(t, store.hasHadPro)

	err = store.Add(licenseOtherUser.Token)
	require.Error(t, err, "adding a license of another user has to fail")
	require.EqualValues(t, 2, store.Size())
	require.True(t, store.hasHadPro)

	err = store.Add(license3TrialSameUser.Token)
	require.NoError(t, err)
	require.EqualValues(t, 3, store.Size())
	require.EqualValues(t, "user@example.com", store.UserID())
	require.True(t, store.hasHadPro)

	file, err := ioutil.TempFile("", "kite-licenses")
	require.NoError(t, err)
	filePath := file.Name()
	defer os.Remove(filePath)

	// store and clear
	err = store.Save(file)
	require.NoError(t, err)
	require.EqualValues(t, 3, store.Size())

	store.ClearAll()
	require.EqualValues(t, 0, store.Size())
	require.EqualValues(t, "", store.UserID())
	require.EqualValues(t, FreePlan, store.Plan())
	require.True(t, store.hasHadPro, "The hasHadPro must not be reset by Clear()")

	// prepare for reading, then restore store from file
	_, err = file.Seek(0, 0)
	require.NoError(t, err)
	// force it to false to make sure Load restores it from disk
	store.hasHadPro = false
	err = store.Load(file)
	require.NoError(t, err)
	require.EqualValues(t, 3, store.Size())
	require.EqualValues(t, "user@example.com", store.UserID())
	require.EqualValues(t, ProMonthly, store.Plan())
	require.True(t, store.hasHadPro)
}

func Test_Validator(t *testing.T) {
	authority1, err := NewTestAuthority()
	require.NoError(t, err)
	authority2, err := NewTestAuthority()
	require.NoError(t, err)
	now := time.Now()
	goodLicense, err := authority1.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: now.Add(1 * time.Hour),
	})
	require.NoError(t, err)

	// license created by another authority
	badLicense, err := authority2.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: now.Add(10 * time.Minute),
	})
	require.NoError(t, err)

	store := NewStore(authority1.CreateValidator(), "")
	store.SetUserID("user@example.com")

	err = store.Add(goodLicense.Token)
	require.NoError(t, err, "A license issued by the correct authority must validate")
	require.EqualValues(t, 1, store.Size())

	err = store.Add(badLicense.Token)
	require.Error(t, err, "A license not issued by the corect authority must not validate")
	require.EqualValues(t, 1, store.Size())
	require.EqualValues(t, "user@example.com", store.UserID())
}

func Test_Multiple(t *testing.T) {
	authority, err := NewTestAuthority()
	require.NoError(t, err)
	now := time.Now()
	store := NewStore(authority.CreateValidator(), "")
	store.SetUserID("user@example.com")

	monthlyLicenseExpired, err := authority.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: now.Add(1 * time.Second),
	})
	require.NoError(t, err)
	// expire license
	time.Sleep(2 * time.Second)
	err = store.Add(monthlyLicenseExpired.Token)
	require.NoError(t, err)
	require.EqualValues(t, 1, store.Size())
	require.EqualValues(t, FreePlan, store.Plan())

	trialLicense, err := authority.CreateLicense(Claims{
		UserID:    store.UserID(),
		Product:   Pro,
		Plan:      ProTrial,
		ExpiresAt: now.Add(1 * time.Hour),
	})
	require.NoError(t, err)
	err = store.Add(trialLicense.Token)
	require.NoError(t, err)
	require.EqualValues(t, 2, store.Size())
	require.EqualValues(t, ProTrial, store.Plan())

	tempLicense, err := authority.CreateLicense(Claims{
		UserID:    store.UserID(),
		Product:   Pro,
		Plan:      ProTemp,
		ExpiresAt: now.Add(30 * time.Minute),
	})
	require.NoError(t, err)
	err = store.Add(tempLicense.Token)
	require.NoError(t, err)
	require.EqualValues(t, 3, store.Size())
	require.EqualValues(t, ProTemp, store.Plan())

	monthlyLicense, err := authority.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
	})
	require.NoError(t, err)
	err = store.Add(monthlyLicense.Token)
	require.NoError(t, err)
	require.EqualValues(t, 4, store.Size())
	require.EqualValues(t, ProMonthly, store.Plan())

	yearlyLicense, err := authority.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProYearly,
		ExpiresAt: now.Add(365 * 24 * time.Hour),
	})
	require.NoError(t, err)
	err = store.Add(yearlyLicense.Token)
	require.NoError(t, err)
	require.EqualValues(t, 5, store.Size())
	require.EqualValues(t, ProYearly, store.Plan())

	monthlyLicense2, err := authority.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: now.Add(60 * 24 * time.Hour),
	})
	require.NoError(t, err)
	err = store.Add(monthlyLicense2.Token)
	require.NoError(t, err)
	require.EqualValues(t, 6, store.Size())
	require.EqualValues(t, ProYearly, store.Plan())

	// validate that the license with the latest expiration wins, regardless of the plan
	monthlyLicense3, err := authority.CreateLicense(Claims{
		UserID:    "user@example.com",
		Product:   Pro,
		Plan:      ProMonthly,
		ExpiresAt: now.Add(730 * 24 * time.Hour),
	})
	require.NoError(t, err)
	err = store.Add(monthlyLicense3.Token)
	require.NoError(t, err)
	require.EqualValues(t, 7, store.Size())
	require.EqualValues(t, ProMonthly, store.Plan(), "The license with the latest expiration is monthlyLicense3")
}

func Test_TrialState(t *testing.T) {
	authority, err := NewTestAuthority()
	require.NoError(t, err)
	now := time.Now()
	store := NewStore(authority.CreateValidator(), "")
	store.SetUserID("user@example.com")
	require.EqualValues(t, FreePlan, store.Plan())
	require.EqualValues(t, true, store.TrialAvailable(), "The trial must be available for an empty license store")

	trialLicenseExpired, err := authority.CreateLicense(Claims{
		UserID:    store.UserID(),
		Product:   Pro,
		Plan:      ProTrial,
		ExpiresAt: now.Add(1 * time.Second),
	})
	require.NoError(t, err)
	// make sure it's expired
	time.Sleep(2 * time.Second)
	err = store.Add(trialLicenseExpired.Token)
	require.NoError(t, err)
	require.EqualValues(t, FreePlan, store.Plan())
	require.EqualValues(t, false, store.TrialAvailable(), "The trial must be unavailable with an expired trial license")

	trialLicenseValid, err := authority.CreateLicense(Claims{
		UserID:    store.UserID(),
		Product:   Pro,
		Plan:      ProTrial,
		ExpiresAt: now.Add(1 * time.Hour),
	})
	require.NoError(t, err)
	err = store.Add(trialLicenseValid.Token)
	require.NoError(t, err)
	require.EqualValues(t, ProTrial, store.Plan())
	require.EqualValues(t, false, store.TrialAvailable(), "The user must be trialing with an active trial license")

	// a store with just a single, active trial license also has to be "Trialing"
	store.ClearAll()
	store.SetUserID("user@example.com")
	err = store.Add(trialLicenseValid.Token)
	require.NoError(t, err)
	require.EqualValues(t, ProTrial, store.Plan())
	require.EqualValues(t, false, store.TrialAvailable(), "The user must be trialing with an active trial license")
}

func Test_KiteServer(t *testing.T) {
	authority, err := NewTestAuthority()
	require.NoError(t, err)
	store := NewStore(authority.CreateValidator(), "")
	store.KiteServer = true

	info := store.LicenseInfo()
	require.EqualValues(t, LicenseInfo{Product: Pro, Plan: ProServer}, info)

	_, _, plan, product := store.LicenseStatus()
	require.EqualValues(t, ProServer, plan)
	require.EqualValues(t, Pro, product)

	plan = store.Plan()
	require.EqualValues(t, ProServer, plan)

	product = store.Product()
	require.EqualValues(t, Pro, product)
}
