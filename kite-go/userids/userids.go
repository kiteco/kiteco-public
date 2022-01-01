package userids

import (
	"fmt"
	"sync"
)

const localMachineID = "kite_local"

// IDs provides high-level access to a user's ids
type IDs interface {
	// MetricsID selects the best ID which is suitable as identifier in metrics
	MetricsID() string

	// ForgetfulMetricsID is like MetricsID, but ignores cached identities for logged-out users
	ForgetfulMetricsID() string

	// UserID returns the numeric id of the current user.
	// It's 0 if the user is not logged in.
	// If the user is logged in then the user id is greater than 0 and unique.
	UserID() int64

	// InstallID returns an id which identifies the installation.
	InstallID() string

	// MachineID returns a unique ID for the current machine
	MachineID() string

	// Email returns the email of the current user, or an empty string
	Email() string

	// String returns a string representation which is useful for debugging
	String() string
}

// NewUserIDs returns a new UserIDs struct which contains all the values
func NewUserIDs(installID, machineID string) *UserIDs {
	return &UserIDs{
		installID: installID,
		machineID: machineID,
	}
}

// UserIDs wraps several ids which identify a user or a user's machine.
// It implements interface IDs. Use the interface in places where only read access is needed.
// Store a pointer to the this struct in places which can update the wrapped IDs
// userID and email can be modified using methods but installID and machineID are static
type UserIDs struct {
	// if !loggedIn, then userID/email refer to cached values
	loggedIn  bool
	userID    int64
	email     string
	installID string
	machineID string
	local     bool
	mu        sync.RWMutex
}

// SetLocal enables using values specific to kite local
func (u *UserIDs) SetLocal(val bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.local = val
}

// UserID returns the stored user id, this may be empty
func (u *UserIDs) UserID() int64 {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.userID
}

// IDs returns the stored user id, machine id and install id (userID might be 0)
func (u *UserIDs) IDs() (int64, string, string) {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.userID, u.machineID, u.installID
}

// SetUser updates the user id and email
func (u *UserIDs) SetUser(id int64, email string, login bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.userID = id
	u.email = email
	u.loggedIn = login
}

// Logout marks the user as logged out
func (u *UserIDs) Logout() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.loggedIn = false
}

// Email returns the email, this may be empty
func (u *UserIDs) Email() string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.email
}

// InstallID returns the installation id, this may be empty
func (u *UserIDs) InstallID() string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.installID
}

// MachineID returns the machine id, this may be empty
func (u *UserIDs) MachineID() string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	if u.local {
		return localMachineID
	}
	return u.machineID
}

// String returns the string representation
func (u *UserIDs) String() string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return fmt.Sprintf("User{id:%d email:%s installID:%s machineID:%s}", u.userID, u.email, u.installID, u.machineID)
}

// MetricsID selects the best ID which is suitable as identifier in metrics
func (u *UserIDs) MetricsID() string {
	u.mu.RLock()
	defer u.mu.RUnlock()

	if u.userID > 0 {
		return fmt.Sprintf("%d", u.userID)
	}
	return u.installID
}

// ForgetfulMetricsID is like MetricsID, but ignores cached identities for logged-out users
func (u *UserIDs) ForgetfulMetricsID() string {
	u.mu.RLock()
	defer u.mu.RUnlock()

	// this behavior is duplicated in kite-go/client/internal/auth/remote.go
	if u.userID > 0 && u.loggedIn {
		return fmt.Sprintf("%d", u.userID)
	}
	return u.installID
}
