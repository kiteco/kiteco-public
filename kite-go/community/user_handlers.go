package community

import (
	"net/http"

	"github.com/kiteco/kiteco/kite-go/web/webutils"
)

// UserHandlers provides the API to handle user login related http requests
type UserHandlers struct {
	users *UserManager
}

// NewUserHandlers returns a pointer to a new UserHandlers object
func NewUserHandlers(users *UserManager) *UserHandlers {
	return &UserHandlers{
		users: users,
	}
}

// HandleCreate creates a new user using the app's UserManager
func (u *UserHandlers) HandleCreate(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	channel := r.FormValue("channel")

	user, session, err := u.users.Create(name, email, password, channel)

	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	SendUserResponse(user, session, w, r)
}

// HandleLogin logs a user in using the app's UserManager
func (u *UserHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	user, session, err := u.users.Login(email, password)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	SendUserResponse(user, session, w, r)
}

// HandleLogout invalidates the session
func (u *UserHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	sessionKey, err := SessionKey(r)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	err = u.users.Logout(sessionKey)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	w.WriteHeader(http.StatusOK)
}
