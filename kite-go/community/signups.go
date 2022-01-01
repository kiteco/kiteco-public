package community

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/jinzhu/gorm"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
)

// Error codes that can be returned by the Identifiers manager.
const (
	ErrSignupNotExist  = 1
	ErrDBError         = 2
	ErrInvalidRequest  = 3
	ErrIncorrectSecret = 4
	ErrEmailSending    = 5
)

var (
	errAlreadyInvited = errors.New("user already invited")

	errorMap = webutils.StatusCodeMap{
		ErrSignupNotExist:  http.StatusNotFound,
		ErrDBError:         http.StatusInternalServerError,
		ErrInvalidRequest:  http.StatusBadRequest,
		ErrIncorrectSecret: http.StatusUnauthorized,
		ErrEmailSending:    http.StatusInternalServerError,
	}

	defaultBcc = "invite-bcc@kite.com"
)

// SignupManager takes a gorm DB instance and maps signup related operations
// to database operations via the models below.
type SignupManager struct {
	db         gorm.DB
	inviteLock sync.Mutex
}

// Signup contains information about a signup to be invited to use Kite.
type Signup struct {
	ID        int64     `json:"-"`
	Email     string    `json:"email,omitempty" valid:"email"`
	ClientIP  string    `json:"client_ip,omitempty" valid:"ip"`
	Metadata  string    `json:"metadata,omitempty" sql:"type:jsonb; default:'{}'; not null" valid:"json"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Secret    string    `json:"secret" sql:"-"`

	InviteCode        string    `json:"invite_code,omitempty"` // empty if not yet invited
	InvitedTimestamp  time.Time `json:"invited_timestamp,omitempty"`
	Redeemed          int64     `json:"redeemed,omitempty"`           // user ID, or empty if not yet redeemed
	RedeemedTimestamp time.Time `json:"redeemed_timestamp,omitempty"` // empty if not yet redeemed

	Downloads []*Download `json:"-"` // timestamps when Kite was downloaded

	Unsubscribed bool `json:"unsubscribed" sql:"default:false"` // For e-mail communications
}

// Download contains a timestamp when a user downloaded Kite.
type Download struct {
	SignupID  int64
	Timestamp time.Time
}

// NewSignupManager creates a new manager to manage signups and invite codes.
func NewSignupManager(db gorm.DB) *SignupManager {
	return &SignupManager{db: db}
}

// Migrate will auto-migrate relevant tables in the db.
func (s *SignupManager) Migrate() error {
	err := s.db.AutoMigrate(&Signup{}, &Download{}).Error
	if err != nil {
		return fmt.Errorf("error creating tables in db: %s", err)
	}
	return nil
}

// CreateOrUpdateSignup creates a new sign up or updates an existing one for a given email.
func (s *SignupManager) CreateOrUpdateSignup(email, metadata, clientIP string) (*Signup, error) {
	email = stdEmail(email)
	var signup Signup
	if s.db.Where(Signup{Email: email}).First(&signup).RecordNotFound() {
		sup := Signup{
			Email:     email,
			Timestamp: time.Now(),
			Metadata:  metadata,
			ClientIP:  clientIP,
		}
		if err := validateSignup(&sup); err != nil {
			return nil, webutils.ErrorCodef(ErrInvalidRequest, "signup did not validate: %v", err)
		}
		if err := s.db.Create(&sup).Error; err != nil {
			return nil, webutils.ErrorCodef(ErrDBError, "error creating new signup: %v", err)
		}
		signup = sup
	} else {
		signup.Metadata = metadata
		if err := validateSignup(&signup); err != nil {
			return nil, webutils.ErrorCodef(ErrInvalidRequest, "signup did not validate: %v", err)
		}
		if err := s.db.Save(&signup).Error; err != nil {
			return nil, webutils.ErrorCodef(ErrDBError, "error saving new metadata for existing signup: %v", err)
		}
	}
	return &signup, nil
}

// Get retrieves a sign up based on email.
func (s *SignupManager) Get(email string) (*Signup, error) {
	email = stdEmail(email)
	var signup Signup
	if s.db.Where(Signup{Email: email}).First(&signup).RecordNotFound() {
		return nil, webutils.ErrorCodef(ErrSignupNotExist, "record not found for email %s", email)
	}
	return &signup, nil
}

// All retrieves all the signups.
func (s *SignupManager) All() ([]*Signup, error) {
	var signups []*Signup
	if err := s.db.Find(&signups).Error; err != nil {
		return nil, webutils.ErrorCodef(ErrDBError, "error retrieving all signups")
	}
	return signups, nil
}

// Invite generates a new invite code and sends it to the given email address.
func (s *SignupManager) Invite(email, host string) (string, error) {
	email = stdEmail(email)
	var signup Signup
	var found *Signup
	if s.db.Where(Signup{Email: email}).First(&signup).RecordNotFound() {
		// lazily create if it's a user who hasn't signed up but we want to invite
		// eg. an influencer
		sup, err := s.CreateOrUpdateSignup(email, "", "")
		if err != nil {
			return "", webutils.ErrorCodef(ErrDBError, "error lazily creating new signup when inviting %s: %v", email, err)
		}
		found = sup
	} else {
		if len(signup.InviteCode) > 0 {
			return "", errAlreadyInvited
		}
		found = &signup
	}

	inviteCode := newInviteCode()
	var usedCode Signup
	for !s.db.Where(Signup{InviteCode: inviteCode}).First(&usedCode).RecordNotFound() {
		inviteCode = newInviteCode()
	}

	found.InviteCode = inviteCode
	found.InvitedTimestamp = time.Now()

	if err := s.db.Save(found).Error; err != nil {
		return "", webutils.ErrorCodef(ErrDBError, "error saving invite code in signup: %v", err)
	}
	return inviteCode, nil
}

// Download captures the current timestamp and saves it in the signup object.
func (s *SignupManager) Download(signup *Signup) error {
	download := &Download{signup.ID, time.Now()}
	if err := s.db.Create(download).Error; err != nil {
		return webutils.ErrorCodef(ErrDBError, "error saving new download in signup: %v", err)
	}
	return nil
}

// Validate checks the db to see if the provided invite code exists.
func (s *SignupManager) Validate(inviteCode string) (*Signup, error) {
	var signup Signup
	if s.db.Where(Signup{InviteCode: inviteCode}).First(&signup).RecordNotFound() {
		return nil, webutils.ErrorCodef(ErrSignupNotExist, "record not found for invite code %s", inviteCode)
	}
	return &signup, nil
}

// Unsubscribe sets unsubscribed to true for a given email
func (s *SignupManager) Unsubscribe(email string) (*Signup, error) {
	signup, err := s.Get(email)
	if err != nil {
		return nil, err
	}
	if signup.Unsubscribed {
		return signup, nil
	}
	signup.Unsubscribed = true
	if err = s.db.Save(signup).Error; err != nil {
		signup.Unsubscribed = false
		return signup, webutils.ErrorCodef(ErrDBError, "error unsubscribing: %v", err)
	}
	return signup, nil
}

// Subscribe sets unsubscribed to false for a given email
func (s *SignupManager) Subscribe(email string) (*Signup, error) {
	signup, err := s.Get(email)
	if err != nil {
		return nil, err
	}
	if !signup.Unsubscribed {
		return signup, nil
	}
	// gorm doesn't work well when setting columns to empty values
	// see: https://github.com/jinzhu/gorm/issues/202
	if err = s.db.Exec(`UPDATE "signup" SET "unsubscribed"='false' WHERE "id"=?`, signup.ID).Error; err != nil {
		return signup, webutils.ErrorCodef(ErrDBError, "error subscribing: %v", err)
	}
	signup.Unsubscribed = false
	return signup, nil
}

// ListUnsubscribed returns all signups that have been unsubscribed
func (s *SignupManager) ListUnsubscribed() ([]*Signup, error) {
	var signups []*Signup
	if err := s.db.Where("unsubscribed = ?", true).Find(&signups).Error; err != nil {
		return signups, webutils.ErrorCodef(ErrDBError, "error listing unsubscribed signups: %v", err)
	}
	return signups, nil
}

// --

const (
	alphabet = "abcdefghjklmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codeLen  = 7
)

func newInviteCode() string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, codeLen)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}

func validateSignup(signup *Signup) error {
	_, err := govalidator.ValidateStruct(signup)
	if err != nil {
		return err
	}
	return nil
}
