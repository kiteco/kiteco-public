package account

import (
	"bytes"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/domains"
	"github.com/kiteco/kiteco/kite-golib/email"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// handleSendEmailInvites sends out emails to people invited by users
func (s *Server) handleSendEmailInvites(w http.ResponseWriter, r *http.Request) {
	lp := "account.Server.handleSendEmailInvites:"

	var request struct {
		Emails []string `json:"emails"`
		Name   string   `json:"name"`
	}

	if ed := unmarshalBody(lp, r, &request); ed.HTTPError() {
		http.Error(w, ed.Msg, ed.Code)
		return
	}

	// validate emails
	for _, e := range request.Emails {
		_, err := mail.ParseAddress(e)
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid email", e), http.StatusBadRequest)
			return
		}
	}

	user, err := s.app.Users.IsValidLogin(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	host := r.Host
	switch host {
	case domains.Alpha:
		host = domains.PrimaryHost
	case domains.Staging:
		host = domains.GaStaging
	}

	link := fmt.Sprintf("https://%s/ref", host)
	u, err := url.Parse(link)
	if err != nil {
		rollbar.Error(err, link)
	}

	for _, e := range request.Emails {
		q := u.Query()
		q.Set("source", "referral_email")
		q.Set("email", e)
		u.RawQuery = q.Encode()

		community.TrackReferralEmailSent(user.IDString(), e, u.String(), request.Name)
	}

	w.WriteHeader(http.StatusOK)
}

// emailVerification sends an email with verification link to the address to be verified.
// NOT USED but we might want to reactivate it
func (s *Server) emailVerification(v *community.EmailVerification, host string) error {
	values := url.Values{}
	values.Set("email", v.Email)
	values.Set("code", v.Code)

	switch host {
	case domains.Alpha:
		host = domains.PrimaryHost
	case domains.Staging:
		host = domains.GaStaging
	}

	link := fmt.Sprintf("https://%s/verify-email?%s", host, values.Encode())
	parsedLink, err := url.Parse(link)
	if err != nil {
		rollbar.Error(err, link)
	}

	var body bytes.Buffer
	err = s.app.Templates.Render(&body, "email-verification.html", map[string]string{
		"VerifyLink": parsedLink.String(),
	})
	if err != nil {
		return err
	}

	emailer, err := s.app.Settings.GetEmailer()
	if err != nil {
		return err
	}

	sender := s.app.Settings.GetSenderAddress()

	return emailer(sender, email.Message{
		HTML:    true,
		To:      []string{v.Email},
		Subject: "Kite: verify your email",
		Body:    body.Bytes(),
	})
}
