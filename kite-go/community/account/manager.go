package account

import (
	"fmt"
	"math"
	"time"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-go/community"
)

var (
	errTrialAlreadyConsumed = errors.New("trial already started")
)

// DB returns the db used for storing and managing accounts.
func DB(driver, uri string) (gorm.DB, error) {
	db, err := gorm.Open(driver, uri)
	if err != nil {
		return gorm.DB{}, err
	}
	db.SingularTable(true)
	db.LogMode(true)
	return db, err
}

// manager of accounts and plans
type manager struct {
	db        gorm.DB
	license   *licenseManager
	users     *community.UserManager
	authority *licensing.Authority
}

func newManager(db gorm.DB, users *community.UserManager, authority *licensing.Authority) *manager {
	license := newLicenseManager(db)
	return &manager{
		db:        db,
		license:   license,
		users:     users,
		authority: authority,
	}
}

// Migrate the relevant tables
func (m *manager) Migrate() error {
	if err := m.db.AutoMigrate(&account{}, &member{}, &subscription{}).Error; err != nil {
		return fmt.Errorf("error migrating tables in account db: %v", err)
	}
	// Add member account_id index
	if err := m.db.Model(&member{}).AddIndex("member_account_id_idx", "account_id").Error; err != nil {
		return fmt.Errorf("error adding index: %v", err)
	}

	if err := m.license.Migrate(); err != nil {
		return err
	}

	return nil
}

// adjustExpiration modifies the expiration so that it expires at 12:00:01 am in the local time of the client
// Either an updates expiration or the original expiration is returned.
// The original expiration is always returned if clientTimeZoneOffset is zero.
// clientTimeZoneOffset is the time zone offset in seconds east of UTC
func (m *manager) adjustExpiration(expiration time.Time, clientTimeZoneOffset int) time.Time {
	// reject 0 and too large offsets, 14h seems to be the maximum
	if clientTimeZoneOffset == 0 || math.Abs(float64(clientTimeZoneOffset)) > 14*60*60 {
		return expiration
	}

	clientZone := time.FixedZone("client", clientTimeZoneOffset)

	// when the trial ends between 12:00am to 7pm (inclusive) local client time,
	// then expire 12:00am (first second of day) of the trial expiry date
	expirationClient := expiration.In(clientZone)
	year, month, day := expirationClient.Date()
	expiration = time.Date(year, month, day, 0, 0, 1, 0, clientZone).Local()

	// when trial ends after 7pm local client time,
	// then expire 12:00am (first second of day) of the day AFTER trial expiry date
	if expirationClient.After(time.Date(year, month, day, 19, 0, 0, 0, clientZone)) {
		expiration = expiration.AddDate(0, 0, 1)
	}

	return expiration
}
