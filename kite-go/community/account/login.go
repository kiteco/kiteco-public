package account

import (
	"net/http"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
)

// --

// handleCreateWeb creates a community.User and account object in the database using
// a referral code (if present). This logic is similar to community.Server.HandleCreate,
// except referral-code aware.
func (s *Server) handleCreateWeb(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	channel := r.FormValue("channel")

	user, session, err := s.app.Users.Create(name, email, password, channel)

	if err != nil {
		webutils.ErrorResponse(w, r, err, community.UserErrorMap)
		return
	}

	community.SendUserResponse(user, session, w, r)
}

// handleLoginWeb logs a user in, similar to community.Server.HandleLogin. The logic is
// duplicated here because we want a separate /api/account/login-web, distinct from
// /api/account/login-desktop (below), and they both should be a part of account.
func (s *Server) handleLoginWeb(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	user, session, err := s.app.Users.LoginDuration(email, password, community.WebSessionExpiration)
	if err != nil {
		webutils.ErrorResponse(w, r, err, community.UserErrorMap)
		return
	}

	community.SendUserResponse(user, session, w, r)
}

// handleLoginDesktop logs a user in, similar to community.Server.HandleLogin. It also
// checks to see if the login should "activate" any pending referrals for this user. Note
// that referrals are only activated once a referred user logs in from the desktop app.
func (s *Server) handleLoginDesktop(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	user, session, err := s.app.Users.Login(email, password)
	if err != nil {
		webutils.ErrorResponse(w, r, err, community.UserErrorMap)
		return
	}

	community.SendUserResponse(user, session, w, r)
}
