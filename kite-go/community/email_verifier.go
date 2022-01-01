package community

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
)

// EmailVerifier encapsulates the logic for verifying email addresses.
type EmailVerifier interface {
	Create(addr string) (*EmailVerification, error)
	Lookup(email, code string) (*EmailVerification, error)
	Remove(v *EmailVerification) error
	Migrate() error
}

// emailVerificationManager is the default emailVerifier implementation.
type emailVerificationManager struct {
	db gorm.DB
}

// NewEmailVerificationManager constructs a new EmailVerifier.
func newEmailVerificationManager(db gorm.DB) EmailVerifier {
	return &emailVerificationManager{
		db: db,
	}
}

func (m *emailVerificationManager) Migrate() error {
	return m.db.AutoMigrate(&EmailVerification{}).Error
}

// EmailVerification describes an email address that needs to be verified, and stores
// the code necessary to verify it.
type EmailVerification struct {
	ID         int
	Email      string `valid:"email,required"`
	Code       string
	Expiration time.Time
}

var (
	// ErrVerificationInvalid means the provided verification code does not match
	// anything in our table
	ErrVerificationInvalid = fmt.Errorf("could not find email verification with that code")
	// ErrVerificationExpired means the user tries to verify an email but the
	// code has expired
	ErrVerificationExpired = fmt.Errorf("that email verification code has expired")
)

// Create constructs a new EmailVerification, saves it to database, and returns it.
func (m *emailVerificationManager) Create(addr string) (*EmailVerification, error) {
	verification := &EmailVerification{
		Email: addr,
		Code:  randomBytesBase64(32),
		// Truncate time to the second, because that's what will happen in the sql
		// database anyways.  This makes things a little easier to test.
		Expiration: time.Now().Add(defaultVerificationExpiration).Truncate(time.Second),
	}

	err := m.db.Save(&verification).Error
	if err != nil {
		return nil, err
	}

	return verification, nil
}

// Lookup finds the EmailVerification for the given email and code.
func (m *emailVerificationManager) Lookup(email, code string) (*EmailVerification, error) {
	var verification EmailVerification
	op := m.db.Where(EmailVerification{Email: email, Code: code}).First(&verification)

	if op.RecordNotFound() {
		return nil, ErrVerificationInvalid
	} else if op.Error != nil {
		return nil, op.Error
	}

	if verification.Expiration.Before(time.Now()) {
		return nil, ErrVerificationExpired
	}

	return &verification, nil
}

// Remove deletes the EmailVerification from the database.
func (m *emailVerificationManager) Remove(v *EmailVerification) error {
	op := m.db.Delete(v)
	if op.Error != nil {
		return op.Error
	}
	return nil
}
