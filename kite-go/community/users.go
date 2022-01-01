package community

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-go/community/student"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"golang.org/x/crypto/bcrypt"
)

const (
	passwordCost      = 10
	minPasswordLength = 6

	// maxPasswordLength is based on a limitation of bycrpt.
	// https://www.usenix.org/legacy/events/usenix99/provos/provos_html/node4.html
	maxPasswordLength = 55

	defaultSessionExpiration       = time.Hour * 24 * 30 * 12 * 10 // 10 years
	defaultVerificationExpiration  = time.Hour * 24 * 30           // 30 days
	defaultPasswordResetExpiration = time.Hour

	// WebSessionExpiration is exported for the web login endpoint
	WebSessionExpiration = time.Hour * 24 * 7 // 7 days
)

// UserErrorMap maps errors returned by the UserManager to HTTP response codes
var UserErrorMap = webutils.StatusCodeMap{
	// Create errors
	ErrCodeUserExists:        http.StatusConflict,
	ErrCodePasswordLong:      http.StatusBadRequest,
	ErrCodePasswordShort:     http.StatusBadRequest,
	ErrCodeInvalidUser:       http.StatusBadRequest,
	ErrCodeInvalidInviteCode: http.StatusBadRequest,
	ErrCodeUsedInviteCode:    http.StatusConflict,

	// Login errors
	ErrCodeUserNotFound:     http.StatusUnauthorized,
	ErrCodeWrongPassword:    http.StatusUnauthorized,
	ErrCodeUserPasswordless: http.StatusUnauthorized,

	// ValidateSession errors
	ErrCodeInvalidSession: http.StatusUnauthorized,

	// Email verification errors
	ErrCodeNoEmailVerification: http.StatusBadRequest,

	ErrCodePersonnelNotApproved: http.StatusUnauthorized,
}

// ErrUserNotFound indicates a user was not found.
var ErrUserNotFound = errors.New("user was not found")

// UserManager takes a gorm DB instance and maps user operations to
// database operations via the models below.
type UserManager struct {
	db             gorm.DB
	studentsDomain *student.DomainLists
}

// UserIdentifier exposes methods used to identify a user and start a trial.
// One of AnonID or (UserID, UserIDString) should return zero values,
// and MetricsID should return the non-zero value.
type UserIdentifier interface {
	AnonID() string
	UserID() int64
	UserIDString() string
	MetricsID() string
	ExistingUser(before time.Time) bool
	GetEmail() string
}

// NewUserManager creates a new user manager using the provided gorm.DB.
func NewUserManager(db gorm.DB, studentsDomain *student.DomainLists) *UserManager {
	return &UserManager{db: db, studentsDomain: studentsDomain}
}

// Migrate will auto-migrate relevant tables in the db.
func (u *UserManager) Migrate() error {
	err := u.db.AutoMigrate(&User{}, &Session{}, &PasswordReset{}, &Nonce{}).Error
	if err != nil {
		return errors.Wrapf(err, "error creating tables in db")
	}
	// Add session key index
	if err := u.db.Model(&Session{}).AddIndex("session_key_idx", "key").Error; err != nil {
		return errors.Wrapf(err, "error adding index")
	}
	// Add user indexes
	if err := u.db.Model(&User{}).AddIndex("user_email_idx", "email").Error; err != nil {
		return errors.Errorf("error adding index: %v", err)
	}
	return nil
}

func stdEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// CreatePasswordless creates a user with just an email. Returns ErrUserExists
// if the email has already been registered.
func (u *UserManager) CreatePasswordless(email, channel string, ignoreChannel bool) (*User, *Session, error) {
	// TODO(Daniel): Maybe we should set this on the frontend, but since the
	// kite-installer is the only user of this, we can set this manually
	// for now
	if channel == "" {
		channel = "autocomplete-python"
	}

	// TODO(Daniel): The Copilot doesn't set the channel and instead passes
	// this value, so we don't mistakenly assume this user came from
	// autocomplete-python
	if ignoreChannel {
		channel = ""
	}

	email = stdEmail(email)
	user := &User{Email: email}

	// Have to validate before checking whether the user exists because when the email is
	// empty Gorm cannot distinguish between a filter for email = "" and the absence of a
	// filter, so Gorm will return all users in the "user already exists" check below.
	err := validate(user)
	if err != nil {
		return nil, nil, webutils.ErrorCodef(ErrCodeInvalidUser, err.Error())
	}

	// Check if the user exists, if so, return ErrUserExists
	if !u.db.Where(User{Email: email}).First(user).RecordNotFound() {
		err = webutils.ErrorCodef(ErrCodeUserExists, "user with email %s already exists", email)
		return nil, nil, err
	}

	// Create a new user struct
	user, err = NewPasswordlessUser(email)
	if err != nil {
		return nil, nil, err
	}

	// create user AND set invite code as redeemed within a transaction
	tx := u.db.Begin()

	// Add user to the database
	if err = tx.Create(user).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	// Create a new session id for the user
	session, err := u.addSession(user, tx)
	if err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	tx.Commit()

	TrackNewUser(user, channel)

	return user, session, nil
}

// Create generates a user with provided username, email, and password. Returns
// ErrUserExists if the email has already been registered.
func (u *UserManager) Create(name, email, password, channel string) (*User, *Session, error) {
	email = stdEmail(email)

	// Create this for validation, but these values will get overwritten by NewUser()
	user := &User{
		Name:  name,
		Email: email,
	}

	// Have to validate before checking whether the user exists because when the email is
	// empty Gorm cannot distinguish between a filter for email = "" and the absence of a
	// filter, so Gorm will return all users in the "user already exists" check below.
	err := validate(user)
	if err != nil {
		return nil, nil, webutils.ErrorCodef(ErrCodeInvalidUser, err.Error())
	}

	// Check if the user exists, if so, return ErrUserExists
	if !u.db.Where(User{Email: email}).First(user).RecordNotFound() {
		err = webutils.ErrorCodef(ErrCodeUserExists, "user with email %s already exists", email)
		return nil, nil, err
	}

	// Create a new user struct
	user, err = NewUser(name, email, password)
	if err != nil {
		return nil, nil, err
	}

	// create user AND set invite code as redeemed within a transaction
	tx := u.db.Begin()

	// Add user to the database
	if err = tx.Create(user).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	// Create a new session id for the user
	session, err := u.addSession(user, tx)
	if err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	tx.Commit()

	TrackNewUser(user, channel)

	return user, session, nil
}

// Get gets a user by ID
func (u *UserManager) Get(id int64) (*User, error) {
	var user User
	err := u.db.First(&user, id).Error
	return &user, err
}

// TxGetByID gets a user by ID
func (u *UserManager) TxGetByID(tx *gorm.DB, id int64) (*User, error) {
	var user User
	err := tx.First(&user, id).Error
	return &user, err
}

// FindByEmail gets a user by email. If the user was not found, an ErrUserNotFound will be returned.
// Lower-level errors may also be returned.
func (u *UserManager) FindByEmail(email string) (*User, error) {
	email = stdEmail(email)
	if !govalidator.IsEmail(email) {
		return nil, webutils.ErrorCodef(ErrCodeInvalidUser, "invalid email")
	}

	var user User
	op := u.db.Where(User{Email: email}).First(&user)
	if op.RecordNotFound() {
		return nil, ErrUserNotFound
	}
	if op.Error != nil {
		return nil, op.Error
	}
	return &user, nil
}

// Login takes an email and password, validates the password, and creates a new session of default duration.
func (u *UserManager) Login(email, password string) (*User, *Session, error) {
	return u.LoginDuration(email, password, defaultSessionExpiration)
}

// LoginDuration takes an email, password, and duration, validates the password, and creates a new session.
func (u *UserManager) LoginDuration(email, password string, expiry time.Duration) (*User, *Session, error) {
	email = stdEmail(email)
	var user User
	op := u.db.Where(User{Email: email}).First(&user)
	if op.RecordNotFound() {
		err := webutils.ErrorCodef(ErrCodeUserNotFound, "could not find user with email %s", email)
		return nil, nil, err
	} else if op.Error != nil {
		return nil, nil, op.Error
	}

	// Passwordless users must set passwords first
	if len(user.HashedPassword) == 0 {
		err := webutils.ErrorCodef(ErrCodeUserPasswordless, "user has not set password")
		return nil, nil, err
	}

	// Check password
	if !comparePassword(&user, []byte(password)) {
		err := webutils.ErrorCodef(ErrCodeWrongPassword, "incorrect password")
		return nil, nil, err
	}

	session, err := u.addSessionDuration(&user, expiry, nil)
	if err != nil {
		return nil, nil, err
	}

	return &user, session, nil
}

// Authenticate takes a session id and authenticates the associated user
func (u *UserManager) Authenticate(key string) (*User, *Session, error) {
	// We need to check for empty strings here because GORM doesn't handle
	// this as expected
	if len(key) == 0 {
		return nil, nil, webutils.ErrorCodef(ErrCodeInvalidSession, "could not find session key")
	}

	// Find the Session
	var session Session
	if u.db.Where(Session{Key: key}).First(&session).RecordNotFound() {
		return nil, nil, webutils.ErrorCodef(ErrCodeInvalidSession, "could not find session key")
	}

	// Check if session has expired
	if session.ExpiresAt.Before(time.Now()) {
		return nil, nil, webutils.ErrorCodef(ErrCodeInvalidSession, "session expired")
	}

	// Check for valid user ID
	var user User
	if u.db.Where(User{ID: int64(session.UserID)}).First(&user).RecordNotFound() {
		return nil, nil, webutils.ErrorCodef(ErrCodeUserNotFound, "user does not exist")
	}

	return &user, &session, nil
}

// Logout takes a session id and sets LoggedOutAt, effectively invalidating the session.
func (u *UserManager) Logout(key string) error {
	// Find the Session
	var session Session
	if u.db.Where(Session{Key: key}).First(&session).RecordNotFound() {
		return webutils.ErrorCodef(ErrCodeInvalidSession, "could not find session key")
	}

	// Check if session has expired
	if session.ExpiresAt.Before(time.Now()) {
		return webutils.ErrorCodef(ErrCodeInvalidSession, "session expired")
	}

	// Set LoggedOutAt
	session.LoggedOutAt = time.Now()
	if err := u.db.Save(&session).Error; err != nil {
		return webutils.ErrorCodef(ErrCodeLogoutFailed, "could not log out")
	}

	return nil
}

// ValidateSession takes a session key and returns the user it is associated with. Returns
// ErrInvalidSession if the session key is invalid or expired.
func (u *UserManager) ValidateSession(key string) (*User, *Session, error) {
	// Find the Session
	var session Session
	if u.db.Where(Session{Key: key}).First(&session).RecordNotFound() {
		return nil, nil, webutils.ErrorCodef(ErrCodeInvalidSession, "could not find session key")
	}

	// Check if session has logged out
	var noTime time.Time
	if session.LoggedOutAt.After(noTime) && session.LoggedOutAt.Before(time.Now()) {
		return nil, nil, webutils.ErrorCodef(ErrCodeInvalidSession, "session logged out")
	}

	// Check if session has expired
	if session.ExpiresAt.Before(time.Now()) {
		return nil, nil, webutils.ErrorCodef(ErrCodeInvalidSession, "session expired")
	}

	// Find the user
	var user User
	if u.db.Model(&session).Related(&user).RecordNotFound() {
		return nil, nil, webutils.ErrorCodef(ErrCodeInvalidSession, "no user with matching session")
	}

	return &user, &session, nil
}

// List returns an array of User objects
func (u *UserManager) List() ([]*User, error) {
	var users []*User
	if err := u.db.Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

// UpdatePassword is used offline to make one-off password changes.
func (u *UserManager) UpdatePassword(email, newPassword string) error {
	var user User
	if u.db.Where(User{Email: email}).First(&user).RecordNotFound() {
		err := webutils.ErrorCodef(ErrCodeUserNotFound, "could not find user with email %s", email)
		return err
	}

	err := validatePassword(newPassword)
	if err != nil {
		return err
	}

	salt := cryptoBytes(8)
	fp, err := hashPassword(salt, []byte(newPassword))
	if err != nil {
		return err
	}

	user.HashedPassword = fp
	user.PasswordSalt = salt
	if err = validate(&user); err != nil {
		return webutils.ErrorCodef(ErrCodeInvalidUser, err.Error())
	}

	if err := u.db.Save(&user).Error; err != nil {
		return errors.Errorf("error saving user: %v", err)
	}
	return nil
}

// IsValidLogin returns the user object associated with a
// request, or nil of the user is not logged in.
// TODO(juan): better name?
func (u *UserManager) IsValidLogin(r *http.Request) (*User, error) {
	sk, err := SessionKey(r)
	if err != nil {
		return nil, err
	}

	user, _, err := u.ValidateSession(sk)
	if user != nil && err == nil {
		return user, nil
	}
	return nil, err
}

func (u *UserManager) addSession(user *User, tx *gorm.DB) (*Session, error) {
	return u.insertSession(user, NewSession(defaultSessionExpiration), tx)
}

func (u *UserManager) addSessionDuration(user *User, dur time.Duration, tx *gorm.DB) (*Session, error) {
	return u.insertSession(user, NewSession(dur), tx)
}

func (u *UserManager) insertSession(user *User, session *Session, tx *gorm.DB) (*Session, error) {
	db := &u.db
	if tx != nil {
		db = tx
	}
	if err := db.Model(user).Association("Sessions").Append([]Session{*session}).Error; err != nil {
		return nil, err
	}
	return session, nil
}

// Delete deletes a user
func (u *UserManager) Delete(user *User) error {
	tx := u.db.Begin()
	// Delete user from database
	if err := tx.Delete(user).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// SoftDelete soft-deletes a user by modifying their email address
func (u *UserManager) SoftDelete(user *User) error {
	if strings.HasSuffix(user.Email, "+deactivated") || strings.HasSuffix(user.Email, "-deactivated") {
		return nil
	}

	user.Email = fmt.Sprintf("%s-%s-deactivated", user.Email, randomBytesHex(4))
	log.Println("updating email to", user.Email)

	tx := u.db.Begin()
	if err := tx.Save(user).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	TrackDeleteUser(strconv.FormatInt(user.ID, 10))

	return nil
}

// DeleteSessions remove Sessions associated with the provided uid
func (u *UserManager) DeleteSessions(uid int64) error {
	tx := u.db.Begin()
	if err := tx.Where(Session{UserID: int(uid)}).Delete(Session{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil

}

// -------------------------------

// User holds all the database account information for an individual user.
type User struct {
	ID       int64    `json:"id"` // Primary key, by GORM covention
	AnonUser AnonUser `gorm:"-"`  // For returning associated anon licenses

	Name  string `json:"name"`
	Email string `valid:"email,required" json:"email"`
	Bio   string `json:"bio"`

	EmailVerified bool `json:"email_verified"`
	IsInternal    bool `json:"is_internal"` // true for employees, contractors, and test accounts

	CreatedAt time.Time // Stores creation time, by GORM convention
	UpdatedAt time.Time `json:"-"` // Stores update time, by GORM convention

	Sessions []Session `json:"-"` // User can have multiple sessions

	HashedPassword []byte `json:"-"`
	PasswordSalt   []byte `json:"-"`

	Unsubscribed bool `json:"unsubscribed" sql:"default:false"` // For e-mail communications
}

// MetricsID implements UserIdentifier
func (u *User) MetricsID() string {
	return u.IDString()
}

// UserID implements UserIdentifier
func (u *User) UserID() int64 {
	return u.ID
}

// UserIDString implements UserIdentifier
func (u *User) UserIDString() string {
	return u.IDString()
}

// AnonID implements UserIdentifier
func (u *User) AnonID() string {
	return u.AnonUser.AnonID()
}

// GetEmail implements UserIdentifier
func (u *User) GetEmail() string {
	return u.Email
}

// ExistingUser implements UserIdentifier
func (u *User) ExistingUser(date time.Time) bool {
	return u.CreatedAt.Before(date)
}

// IDString returns the user id as a string
func (u *User) IDString() string {
	return strconv.FormatInt(u.ID, 10)
}

// IsStudent test if the user email address match a school email domain
func (u *UserManager) IsStudent(user UserIdentifier) bool {
	if u.studentsDomain == nil {
		log.Println("WARN IsStudent used without initializing domain lists before, return false")
		return false
	}
	if user == nil {
		return false
	}

	return u.studentsDomain.IsStudent(user.GetEmail())
}

// NewPasswordlessUser creates a user with just an email. Returns an error if
// applicable.
func NewPasswordlessUser(email string) (*User, error) {
	user := User{Email: email}
	if err := validate(&user); err != nil {
		return nil, webutils.ErrorCodef(ErrCodeInvalidUser, err.Error())
	}
	return &user, nil
}

// NewUser creates a user with the given parameters and returns an error if applicable.
func NewUser(name, email, password string) (*User, error) {
	err := validatePassword(password)
	if err != nil {
		return nil, err
	}

	// Make password
	salt := cryptoBytes(8)
	fp, err := hashPassword(salt, []byte(password))
	if err != nil {
		return nil, err
	}
	user := User{
		Name:           name,
		Email:          email,
		HashedPassword: fp,
		PasswordSalt:   salt,
	}
	if err = validate(&user); err != nil {
		return nil, webutils.ErrorCodef(ErrCodeInvalidUser, err.Error())
	}

	return &user, nil
}

func hashPassword(salt, password []byte) ([]byte, error) {
	input := append(salt, password...)
	fp, err := bcrypt.GenerateFromPassword(input, passwordCost)
	if err != nil {
		return nil, err
	}
	return fp, nil
}

func comparePassword(user *User, passwd []byte) bool {
	input := make([]byte, len(user.PasswordSalt))
	copy(input, user.PasswordSalt)
	input = append(input, passwd...)

	err := bcrypt.CompareHashAndPassword(user.HashedPassword, input)
	return err == nil
}

// --

// Session holds the information for a logged-in user session.
type Session struct {
	ID          int
	UserID      int
	Key         string
	ExpiresAt   time.Time
	LoggedOutAt time.Time
}

// NewSession returns a pointer to a newly initialized Session.
func NewSession(dur time.Duration) *Session {
	return &Session{
		Key:       randomBytesBase64(32),
		ExpiresAt: time.Now().Add(dur),
	}
}

// --

// This should only be called when a user has clicked an email verification link,
// and should be immediately followed by a call to EmailVerifier.Remove()!
func (u *UserManager) recordVerifiedEmail(email string) error {
	var user User
	op := u.db.Where(User{Email: email}).First(&user)
	if op.Error != nil {
		return op.Error
	}
	user.EmailVerified = true
	if strings.HasSuffix(user.Email, "@kite.com") {
		user.IsInternal = true
	}
	op = u.db.Save(&user)
	if op.Error != nil {
		return op.Error
	}
	return nil
}

// --

// PasswordReset records a user's password reset request, esp the token that is used to
// verify that the user actually has control of their email account.
type PasswordReset struct {
	ID         int64
	UserID     int64
	Token      string
	Expiration time.Time
}

// NewPasswordReset generates a new PasswordReset, properly setting Token and Expiration.
func NewPasswordReset(userID int64) *PasswordReset {
	return &PasswordReset{
		UserID: userID,
		Token:  randomBytesBase64(32),
		// Truncate time to the second, because that's what will happen in the sql
		// database anyways.  This makes things a little easier to test.
		Expiration: time.Now().Add(defaultPasswordResetExpiration).Truncate(time.Second),
	}
}

// CreatePasswordReset creates a new PasswordReset object for the given user
func (u *UserManager) CreatePasswordReset(user *User) (*PasswordReset, error) {
	reset := NewPasswordReset(user.ID)
	err := u.db.Save(&reset).Error
	if err != nil {
		return nil, err
	}
	return reset, nil
}

var (
	// ErrInvalidReset means the were supplied an invalid reset token
	ErrInvalidReset = errors.New("invalid reset token")
	// ErrExpiredReset means we were supplied an expired reset token
	ErrExpiredReset = errors.New("expired reset token")
	// ErrNoUserForReset means we could not find a user with ID as specified in the reset token
	ErrNoUserForReset = errors.New("no user for reset token")
)

// PasswordResetUser is a helper that returns the User and PasswordReset associated with the
// provided reset token.
func (u *UserManager) PasswordResetUser(resetToken string) (*User, *PasswordReset, error) {
	var reset PasswordReset
	if u.db.Where(PasswordReset{Token: resetToken}).First(&reset).RecordNotFound() {
		return nil, nil, ErrInvalidReset
	}
	if reset.Expiration.Before(time.Now()) {
		u.db.Delete(&reset)
		return nil, nil, ErrExpiredReset
	}
	var user User
	if u.db.Where(User{ID: reset.UserID}).First(&user).RecordNotFound() {
		return nil, &reset, ErrNoUserForReset
	}
	return &user, &reset, nil
}

// PerformPasswordReset actually resets the user password, provided resetToken is valid.
func (u *UserManager) PerformPasswordReset(resetToken string, newPassword string) (*User, error) {
	user, reset, err := u.PasswordResetUser(resetToken)
	if err != nil {
		return nil, err
	}

	err = validatePassword(newPassword)
	if err != nil {
		return nil, err
	}

	// Make new password
	salt := cryptoBytes(8)
	hashedPassword, err := hashPassword(salt, []byte(newPassword))
	if err != nil {
		return nil, err
	}

	// Save user with new password
	user.HashedPassword = hashedPassword
	user.PasswordSalt = salt
	err = validate(user)
	if err != nil {
		return nil, err
	}
	err = u.db.Save(user).Error
	if err != nil {
		return nil, err
	}

	// Delete password reset entry so you can't re-use the reset link
	err = u.db.Delete(reset).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Unsubscribe sets unsubscribed to true for a given email
func (u *UserManager) Unsubscribe(email string) (*User, error) {
	user, err := u.FindByEmail(email)
	if err != nil {
		return nil, err
	}
	if user.Unsubscribed {
		return user, nil
	}
	user.Unsubscribed = true
	if err = u.db.Save(&user).Error; err != nil {
		user.Unsubscribed = false
		return user, webutils.ErrorCodef(ErrDBError, "error unsubscribing: %v", err)
	}
	return user, nil
}

// Subscribe sets unsubscribed to false for a given email
func (u *UserManager) Subscribe(email string) (*User, error) {
	user, err := u.FindByEmail(email)
	if err != nil {
		return nil, err
	}
	if !user.Unsubscribed {
		return user, nil
	}
	// gorm doesn't work well when setting columns to empty values
	// see: https://github.com/jinzhu/gorm/issues/202
	if err = u.db.Exec(`UPDATE "user" SET "unsubscribed"='false' WHERE "id"=?`, user.ID).Error; err != nil {
		return user, webutils.ErrorCodef(ErrDBError, "error subscribing: %v", err)
	}
	user.Unsubscribed = false
	return user, nil
}

// ListUnsubscribed returns all users that have been unsubscribed
func (u *UserManager) ListUnsubscribed() ([]*User, error) {
	var users []*User
	if err := u.db.Where("unsubscribed = ?", true).Find(&users).Error; err != nil {
		return users, webutils.ErrorCodef(ErrDBError, "error listing unsubscribed users: %v", err)
	}
	return users, nil
}

var (
	// ErrEmailInvalid indicates that the email is not a valid email
	ErrEmailInvalid = errors.New("invalid email address")
	// ErrEmailUsed indicates that the email is already in use
	ErrEmailUsed = errors.New("email address already in use")
)

// CheckEmail returns nil if both of the following conditions are true:
//  - 1) is valid email
//  - 2) email is not already in use
func (u *UserManager) CheckEmail(email string) error {
	user := User{
		Email: email,
	}

	if validate(&user) != nil {
		return ErrEmailInvalid
	}

	_, err := u.FindByEmail(email)
	if err == nil {
		return ErrEmailUsed
	}
	if err == ErrUserNotFound {
		return nil
	}
	return err
}

// --

func validate(v interface{}) error {
	_, err := govalidator.ValidateStruct(v)
	if err != nil {
		return err
	}
	return nil
}

func validatePassword(pw string) error {
	switch {
	case len(pw) > maxPasswordLength:
		return webutils.ErrorCodef(ErrCodePasswordLong, "password is too long")
	case len(pw) < minPasswordLength:
		return webutils.ErrorCodef(ErrCodePasswordShort, "password is too short")
	}
	return nil
}

// TODO(juan): merge with above,
// left here for now since unclear where we rely
// on the extra info in the error above
func checkPassword(pw string) error {
	switch {
	case len(pw) > maxPasswordLength:
		return errors.New("password is too long")
	case len(pw) < minPasswordLength:
		return errors.New("password is too short")
	}
	return nil
}
