package handlers

import (
	"net/http"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

var (
	communityStatusMap = webutils.StatusCodeMap{
		community.ErrCodeInvalidSession: http.StatusUnauthorized,
	}
)

// --

// UserHandlers provides the API to handle user login related http requests
type UserHandlers struct {
	users     *community.UserManager
	templates *templateset.Set
}

// NewUserHandlers returns a pointer to a new UserHandlers object
func NewUserHandlers(users *community.UserManager, templates *templateset.Set) *UserHandlers {
	return &UserHandlers{
		users:     users,
		templates: templates,
	}
}

// HandleLogin handles user login request.
func (u *UserHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	payload := map[string]string{"Alert": ""}

	switch r.Method {
	case "POST":
		email := r.FormValue("email")
		password := r.FormValue("password")
		_, session, err := u.users.Login(email, password)
		if err != nil {
			payload["Alert"] = err.Error()
		} else {
			community.SetSession(session, w, r)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
	}

	err := u.templates.Render(w, "login.html", payload)
	if err != nil {
		webutils.ErrorResponse(w, r, err, communityStatusMap)
		return
	}
}

// HandleLogout handles user log out request.
func (u *UserHandlers) HandleLogout(w http.ResponseWriter, r *http.Request) {
	sessionKey, err := community.SessionKey(r)
	if err != nil {
		webutils.ErrorResponse(w, r, err, communityStatusMap)
		return
	}

	err = u.users.Logout(sessionKey)
	if err != nil {
		webutils.ErrorResponse(w, r, err, communityStatusMap)
		return
	}

	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
}
