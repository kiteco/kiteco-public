package community

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/domains"
)

var (
	defaultNonceExpiration = 30 * time.Second
)

var (
	errNonceNotFound        = errors.New("nonce not found")
	errNonceExpired         = errors.New("nonce expired")
	errNonceSessionNotFound = errors.New("no session found for nonce")
)

var (
	productionTarget *url.URL
	stagingTarget    *url.URL
)

func init() {
	var err error
	productionTarget, err = url.Parse(fmt.Sprintf("https://%s", domains.PrimaryHost))
	if err != nil {
		log.Fatalln(err)
	}

	stagingTarget, err = url.Parse(fmt.Sprintf("https://%s", domains.GaStaging))
	if err != nil {
		log.Fatalln(err)
	}
}

// Nonce is used for one-time login to the web-app
type Nonce struct {
	ID        int64
	UserID    int64     `valid:"required"`
	MachineID string    `valid:"required"`
	Value     string    `valid:"required"`
	ExpiresAt time.Time `valid:"required"`
}

// NewNonce creates a nonce for the provided user id.
func NewNonce(uid int64, mid string) (*Nonce, error) {
	n := &Nonce{
		UserID:    uid,
		MachineID: mid,
		Value:     randomBytesBase64(32),
		ExpiresAt: time.Now().Add(defaultNonceExpiration),
	}
	if err := validate(n); err != nil {
		return nil, err
	}
	return n, nil
}

// Server HTTP Handlers -------------------------------------------------------

// HandleCreateNonce returns a nonce object to the authenticated user
func (s *Server) HandleCreateNonce(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)
	machineID := GetMachine(r)
	nonce, err := s.app.Users.CreateNonceForUser(user.ID, machineID)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	buf, err := json.Marshal(nonce)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

// HandleRedeemNonce validates the nonce, and redirects the user to the requested page
// with session cookies set.
func (s *Server) HandleRedeemNonce(w http.ResponseWriter, r *http.Request) {
	dest := r.URL.Query().Get("d")
	value := r.URL.Query().Get("n")
	target := r.URL.Query().Get("t")

	_, session, machine, err := s.app.Users.RedeemNonce(value)
	if err != nil {
		// TODO(tarak): Redirect to login screen
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	SetSession(session, w, r)

	// We set the machineID via a cookie as well so that it persists over the redirect.
	// This allows it to be extracted for user-specific data retrieval (many endpoints
	// require both the userID and machineID)
	setMachine(machine, w, r)

	// Some interesting footwork here: If the request came from:
	// - alpha.kite.com -> send to kite.com
	// - staging.kite.com -> send to ga-staging.kite.com
	// - otherwise, leave it be and redirect relative to current location

	var destErr error
	var newDest *url.URL
	switch target {
	case domains.Alpha:
		newDest, destErr = productionTarget.Parse(dest)
	case domains.Staging:
		newDest, destErr = stagingTarget.Parse(dest)
	}

	if destErr == nil && newDest != nil {
		http.Redirect(w, r, newDest.String(), http.StatusFound)
		return
	}

	http.Redirect(w, r, dest, http.StatusFound)
}

// UserManager Methods --------------------------------------------------------

// CreateNonceForUser will create a Nonce for the provided uid
func (u *UserManager) CreateNonceForUser(uid int64, mid string) (*Nonce, error) {
	n, err := NewNonce(uid, mid)
	if err != nil {
		return nil, err
	}

	err = u.db.Save(n).Error
	if err != nil {
		return nil, err
	}

	return n, nil
}

// RedeemNonce takes a nonce value, verifies it, marks as used, and returns
// associated *User and *Session object.
func (u *UserManager) RedeemNonce(value string) (*User, *Session, string, error) {
	var n Nonce
	if u.db.Where(Nonce{Value: value}).First(&n).RecordNotFound() {
		return nil, nil, "", errNonceNotFound
	}

	if err := u.db.Delete(&n).Error; err != nil {
		return nil, nil, "", err
	}

	if n.ExpiresAt.Before(time.Now()) {
		return nil, nil, "", errNonceExpired
	}

	var session Session
	if u.db.Where(Session{UserID: int(n.UserID)}).Last(&session).RecordNotFound() {
		return nil, nil, "", errNonceSessionNotFound
	}

	user, err := u.Get(n.UserID)
	if err != nil {
		return nil, nil, "", err
	}

	newSession, err := u.addSessionDuration(user, WebSessionExpiration, nil)
	if err != nil {
		return nil, nil, "", err
	}

	return user, newSession, n.MachineID, nil
}
