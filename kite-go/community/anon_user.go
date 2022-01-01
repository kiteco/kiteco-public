package community

import (
	"net/http"
	"time"
)

// AnonUser contains identification for an anonymous user
type AnonUser struct {
	InstallID string `json:"install_id" gorm:"-"`
}

// GetAnonUser will return the AnonUser object associated with an anon user. It will return
// nil if no anon user was found. Make sure to use this method only in handlers that are wrapped
// with UserValidation.WrapAllowAnon.
func GetAnonUser(r *http.Request) *AnonUser {
	if user, ok := r.Context().Value(anonUserKey).(*AnonUser); ok {
		return user
	}
	return nil
}

// MetricsID implements UserIdentifier
func (u *AnonUser) MetricsID() string {
	return u.InstallID
}

// UserID implements UserIdentifier
func (u *AnonUser) UserID() int64 {
	return 0
}

// UserIDString implements UserIdentifier
func (u *AnonUser) UserIDString() string {
	return ""
}

// AnonID implements UserIdentifier
func (u *AnonUser) AnonID() string {
	return u.InstallID
}

// GetEmail implements UserIdentifier
func (u *AnonUser) GetEmail() string {
	return ""
}

// ExistingUser implements UserIdentifier
func (u *AnonUser) ExistingUser(before time.Time) bool {
	return false
}
