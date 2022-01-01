package community

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/domains"
	"github.com/kiteco/kiteco/kite-golib/email"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

type contextKey string

func (c contextKey) String() string {
	return fmt.Sprintf("community:%s", string(c))
}

const (
	sessionKey       = "XXXXXXX"
	machineCookieKey = "XXXXXXX"

	userKey     = contextKey("XXXXXXX")
	anonUserKey = contextKey("XXXXXXX")
	machineKey  = contextKey("XXXXXXX")

	// MachineHeader is inserted by the client into all backend requests
	MachineHeader = "X-Kite-Machine"

	// BypassInviteCode to skip invite code checking when creating new users, where necessary.
	BypassInviteCode = "XXXXXXX"

	defaultSecret = "XXXXXXX"
)

// Server is a http wrapper around the community application
type Server struct {
	app *App
}

// NewServer creates a new server with the provided app.
func NewServer(app *App) *Server {
	return &Server{
		app: app,
	}
}

// SetupRoutes prepares handlers for the basic user server API, in the given Router.
func (s *Server) SetupRoutes(mux *mux.Router) {
	mux.PathPrefix("/community/static/").Handler(http.StripPrefix("/community", http.FileServer(s.app.fileSystem)))

	acct := mux.PathPrefix("/api/account/").Subrouter()
	acct.HandleFunc("/login", s.HandleLogin).Methods("POST")
	acct.HandleFunc("/create", s.HandleCreate).Methods("POST")
	acct.HandleFunc("/createPasswordless", s.HandleCreatePasswordless).Methods("POST")

	acct.HandleFunc("/authenticate", s.HandleAuthenticate).Methods("POST")
	acct.Handle("/user", s.app.Auth.Wrap(s.HandleUser)).Methods("GET")
	acct.Handle("/authenticated", s.app.Auth.Wrap(s.HandleAuthenticated)).Methods("GET")
	acct.Handle("/logout", s.app.Auth.Wrap(s.HandleLogout)).Methods("GET")
	acct.Handle("/resendVerification", s.app.Auth.Wrap(s.HandleResendVerification)).Methods("POST")
	acct.HandleFunc("/check-email", s.HandleCheckEmail).Methods("POST")
	acct.HandleFunc("/check-password", s.HandleCheckPassword).Methods("POST")
	acct.HandleFunc("/reset-password/request", s.HandlePasswordResetRequest).Methods("POST")
	acct.HandleFunc("/reset-password/perform", s.HandlePasswordResetPerform).Methods("POST")
	acct.HandleFunc("/verify-email", s.HandleVerifyEmail).Methods("POST")

	mux.Handle("/account/login-nonce", s.app.Auth.Wrap(s.HandleCreateNonce)).Methods("GET")
	mux.HandleFunc("/account/desktop-login", s.HandleRedeemNonce).Methods("GET")

	// Used by front end to signup and invite person interested in trying out Kite
	mux.HandleFunc("/api/signups", s.HandleCreateSignup).Methods("POST")
	mux.HandleFunc("/api/signups", s.HandleAll).Methods("GET")
	mux.HandleFunc("/api/invite", s.HandleInvite).Methods("POST")

	// Remote settings update
	mux.HandleFunc("/api/remote-settings", handleRemoteSettings).Methods("GET")

	// E-mail communications
	mux.HandleFunc("/unsubscribe", s.HandleUnsubscribe).Methods("POST")
	mux.HandleFunc("/subscribe", s.HandleSubscribe).Methods("POST")
	mux.HandleFunc("/listUnsubscribed", s.HandleListUnsubscribed).Methods("GET")
}

///////////////////////////////////////////////////////////////////////////////
//
//  User

// HandleCreate creates a new user using the app's UserManager
func (s *Server) HandleCreate(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	user, session, err := s.app.Users.Create(name, email, password, "")

	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	SendUserResponse(user, session, w, r)
}

// HandleCreatePasswordless creates a new passwordless user using the
// app's UserManager
func (s *Server) HandleCreatePasswordless(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	channel := r.FormValue("channel")
	ignoreChannel := r.FormValue("ignore_channel")

	user, session, err := s.app.Users.CreatePasswordless(email, channel, ignoreChannel != "")
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	SendUserResponse(user, session, w, r)
}

// HandleLogin logs a user in using the app's UserManager
func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	user, session, err := s.app.Users.Login(email, password)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	SendUserResponse(user, session, w, r)
}

// HandleAuthenticate authenticates a user with a session key.
// This functionality is implemented to support first-time users who
// create accounts via the in-editor install flow and subsequently
// don't have a password set.
func (s *Server) HandleAuthenticate(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")

	user, session, err := s.app.Users.Authenticate(key)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	SendUserResponse(user, session, w, r)
}

// HandleUser returns the user object.
func (s *Server) HandleUser(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)
	SendUserResponse(user, nil, w, r)
}

// HandleAuthenticated serves the string "authenticated" to logged-in users.
// This is used to determine that (1) the user is logged in, and (2) we are
// connected to Kite's servers (rather than, eg, a wifi paywall).
func (s *Server) HandleAuthenticated(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("authenticated"))
}

// HandleLogout invalidates the session
func (s *Server) HandleLogout(w http.ResponseWriter, r *http.Request) {
	sessionKey, err := SessionKey(r)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	err = s.app.Users.Logout(sessionKey)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	clearHMAC(w, r)
	ClearSession(w, r)
	w.WriteHeader(http.StatusOK)
}

func logEmailAndIP(r *http.Request, email string, reqType string) {
	remoteIP := remoteIP(r)
	log.Printf("%s => email: %s, host: %s, remoteIP: %s", reqType, email, r.Host, remoteIP)
}

// HandlePasswordResetRequest accepts an email to send out a password reset email
func (s *Server) HandlePasswordResetRequest(w http.ResponseWriter, r *http.Request) {
	emailFormVal := r.FormValue("email")
	logEmailAndIP(r, emailFormVal, "reset-password-request")
	email := stdEmail(emailFormVal)

	user, err := s.app.Users.FindByEmail(email)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	reset, err := s.app.Users.CreatePasswordReset(user)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	err = s.EmailPasswordReset(user, reset, r.Host)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandlePasswordResetPerform accepts a new password with a required password
// reset token to reset a user's password
func (s *Server) HandlePasswordResetPerform(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	token := r.FormValue("token")
	email := r.FormValue("email")
	logEmailAndIP(r, email, "reset-password-perform")

	user, _, err := s.app.Users.PasswordResetUser(token)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	if user.Email != email {
		http.Error(w, "sorry, we don't recognize that email", http.StatusBadRequest)
		return
	}

	_, err = s.app.Users.PerformPasswordReset(token, password)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}

	// Since we know this user has access to this email address, if the address
	// is not verified go ahead and verify it:
	if !user.EmailVerified {
		verifyErr := s.app.Users.recordVerifiedEmail(user.Email)
		if verifyErr != nil {
			rollbar.Error(fmt.Errorf("failed to verify email upon successful password reset: %s", verifyErr))
		}
	}

}

// SendUserResponse handles serializing the user and adding session cookies if needed.
func SendUserResponse(user *User, session *Session, w http.ResponseWriter, r *http.Request) {
	SetSession(session, w, r)
	if user != nil && session != nil {
		headers := HmacHeadersFromUserSession(user, session)
		for h, vals := range headers {
			w.Header().Set(h, vals[0])
		}
	}

	buf, err := json.Marshal(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

// HandleResendVerification resends the verification email
func (s *Server) HandleResendVerification(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)
	verification, err := s.app.EmailVerifier.Create(user.Email)
	if err != nil {
		return
	}
	err = s.emailVerification(verification, r.Host)
	if err != nil {
		webutils.ErrorResponse(w, r, err, UserErrorMap)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandleVerifyEmail allows users to verify emails
func (s *Server) HandleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	addr := r.FormValue("email")
	code := r.FormValue("code")
	user, err := s.app.Users.FindByEmail(addr)

	if err != nil {
		http.Error(w, "sorry, we don't recognize that email", http.StatusNotFound)
		return
	}

	verification, err := s.app.EmailVerifier.Lookup(addr, code)
	if err == ErrVerificationInvalid {
		if user.EmailVerified {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "that email verification code is invalid", http.StatusNotFound)
		return
	} else if err == ErrVerificationExpired {
		w.WriteHeader(http.StatusBadRequest)
		renewed, renewErr := s.app.EmailVerifier.Create(addr)
		if renewErr != nil {
			http.Error(w, "that email verification code has expired", http.StatusBadRequest)
			return
		}
		renewErr = s.emailVerification(renewed, r.Host)
		if renewErr != nil {
			rollbar.Error(renewErr, renewed)
			http.Error(w, "that email verification code has expired", http.StatusBadRequest)
			return
		}
		http.Error(w, "please check your email for a new verification link.", http.StatusBadRequest)
		return
	} else if err != nil {
		http.Error(w, "there was an error while attempting to verify your email", http.StatusInternalServerError)
		return
	}

	// Passwordless users must verify and set their password at the same time
	// if this endpoint sends back anything in the body of a successful response,
	// assume body is the token to create a new password
	if len(user.HashedPassword) == 0 {
		reset, resetErr := s.app.Users.CreatePasswordReset(user)
		if resetErr != nil {
			http.Error(w, "there was an error while attempting to verify your email", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(reset.Token))
		return
	}

	err = s.app.Users.recordVerifiedEmail(addr)
	if err != nil {
		http.Error(w, "there was an error while attempting to verify your email", http.StatusInternalServerError)
		return
	}

	err = s.app.EmailVerifier.Remove(verification)
	if err != nil {
		// User impact is negligible here, so log this error to Rollbar and carry on
		rollbar.Error(err)
	}
}

// ---

// UserValidation is middleware that will check the requests cookies for session information,
// and lookup the coresponding user. The user is stored into the request object via context.
// If no matching user is found, the middleware will return early with 401 Unauthorized.
type UserValidation struct {
	users *UserManager
}

// NewUserValidation creates a new UserValidation object using the provided App.
func NewUserValidation(users *UserManager) *UserValidation {
	return &UserValidation{
		users: users,
	}
}

// WrapNoBlockFunc operates on an http.HandlerFunc
func (u *UserValidation) WrapNoBlockFunc(next http.HandlerFunc) http.Handler {
	return u.WrapNoBlock(next)
}

// WrapNoBlock wraps an endpoint with auth checks, and will set the user object if
// a valid session is found, but will not block requests with a StatusUnauthorized.
func (u *UserValidation) WrapNoBlock(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check for machine information
		mid := getMachine(r)

		// Check HMAC to bypass database hit. Fall back on session lookup
		if user, err := checkHMACFromRequest(r); err == nil {
			r = r.WithContext(context.WithValue(r.Context(), userKey, user))
			r = r.WithContext(context.WithValue(r.Context(), machineKey, mid))
			next.ServeHTTP(w, r)
			return
		}

		// Note: some requests legitimately will not have the session set. This is OK.
		// Worst-case, upstream will handle this.
		sessionKey, err := SessionKey(r)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Try to validate the session. If it doesn't validate, don't do anything here.
		user, session, err := u.users.ValidateSession(sessionKey)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Generate hmac headers
		log.Printf("user %d (%s) authenticated for %s via db", user.ID, user.Email, r.URL.Path)
		headers := HmacHeadersFromUserSession(user, session)
		for key, vals := range headers {
			w.Header().Set(key, vals[0])
		}

		r = r.WithContext(context.WithValue(r.Context(), userKey, user))
		r = r.WithContext(context.WithValue(r.Context(), machineKey, mid))

		next.ServeHTTP(w, r)
	}
}

// Wrap will wrap the provided HandlerFunc with user authentication, using provided app.
func (u *UserValidation) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return u.WrapNoBlock(func(w http.ResponseWriter, r *http.Request) {
		if GetUser(r) == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WrapAllowAnon allows both authenticated users and anon users to access next.
func (u *UserValidation) WrapAllowAnon(next http.HandlerFunc) http.HandlerFunc {
	return u.WrapNoBlock(func(w http.ResponseWriter, r *http.Request) {
		iid := r.URL.Query().Get("install-id")
		if iid != "" {
			r = r.WithContext(context.WithValue(r.Context(), anonUserKey, &AnonUser{InstallID: iid}))
		}

		if GetUser(r) == nil && GetAnonUser(r) == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WrapRedirect redirects to dst if the user is not authenticated.
// The redirect target, dst, defaults to https://kite.com/login (if passed "").
//
// A redirect=<URL> query parameter is added to dst if one doesn't already exist.
// If dst's host is "kite.com" or "www.kite.com", and the request is coming to staging,
// we dynamically redirect to ga-staging.kite.com instead.
func (u *UserValidation) WrapRedirect(dst string, next http.HandlerFunc) http.HandlerFunc {
	if dst == "" {
		dst = fmt.Sprintf("https://%s/login", domains.PrimaryHost)
	}

	return u.WrapNoBlock(func(w http.ResponseWriter, r *http.Request) {
		dst := dst
		if GetUser(r) != nil {
			next.ServeHTTP(w, r)
			return
		}

		if dstURL, _ := url.Parse(dst); dstURL != nil {
			switch dstURL.Host {
			case "":
				dstURL.Host = domains.PrimaryHost
				fallthrough
			case domains.PrimaryHost, domains.WWW:
				switch r.Host {
				case domains.Staging:
					dstURL.Host = domains.GaStaging
				case "localhost:9090":
					dstURL.Scheme = "http"
					dstURL.Host = "localhost:3000"
				}
			}

			srcURL := r.URL
			srcURL.Host = r.Host
			if srcURL.Host == "" {
				srcURL.Host = domains.Alpha
			}

			var updateQuery bool
			vals := dstURL.Query()
			if email, ok := r.URL.Query()["email"]; ok {
				// We forward email parameter if its present to the login form
				vals["email"] = email
				updateQuery = true
			}
			if _, exists := vals["redirect"]; !exists {
				vals["redirect"] = []string{srcURL.String()}
				updateQuery = true
			}
			if updateQuery {
				dstURL.RawQuery = vals.Encode()
			}

			dst = dstURL.String()
		}

		http.Redirect(w, r, dst, http.StatusFound)
		return
	})
}

// HandleCreateSignup takes a POST request and signs up the email provided in the body.
// Subsequent requests with the same email are allowed; the metadata provided in
// the body is simply updated.
func (s *Server) HandleCreateSignup(w http.ResponseWriter, r *http.Request) {
	data, err := readSignup(r.Body)
	if err != nil {
		webutils.ErrorResponse(w, r, err, errorMap)
		return
	}

	_, err = s.app.Signups.CreateOrUpdateSignup(data.Email, data.Metadata, deduceClientIP(r))
	if err != nil {
		webutils.ErrorResponse(w, r, err, errorMap)
		return
	}
}

type inviteData struct {
	Emails []string `json:"emails"`
	Secret string   `json:"secret"`
}

// HandleInvite takes a POST request and invites a user.
// It generates a new unique 7-digit invite code, and emails it to the user.
func (s *Server) HandleInvite(w http.ResponseWriter, r *http.Request) {
	data, err := readInviteData(r.Body)
	if err != nil {
		webutils.ErrorResponse(w, r, err, errorMap)
		return
	}

	if err := validateSecretKey(data.Secret); err != nil {
		webutils.ErrorResponse(w, r, err, errorMap)
		return
	}

	var invites []map[string]string

	for _, email := range data.Emails {
		// generate invitation code
		code, err := s.app.Signups.Invite(email, r.Host)
		if err == errAlreadyInvited {
			continue
		}
		if err != nil {
			log.Printf("error inviting %s: %v\n", email, err)
		}
		invites = append(invites, map[string]string{
			"email": email,
			"code":  code,
		})
	}

	js, err := json.Marshal(invites)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// HandleAll retrieves all the signups.
func (s *Server) HandleAll(w http.ResponseWriter, r *http.Request) {
	secret := r.URL.Query().Get("secret")

	if err := validateSecretKey(secret); err != nil {
		webutils.ErrorResponse(w, r, err, errorMap)
		return
	}

	signups, err := s.app.Signups.All()
	if err != nil {
		webutils.ErrorResponse(w, r, err, errorMap)
		return
	}

	for _, signup := range signups {
		signup.Secret = ""
	}

	buf, err := json.Marshal(signups)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

// subscriptionResult describes the result of a unsubscribe/subscribe call
type subscriptionResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// HandleUnsubscribe unsubscribes the signup/user with the given email
func (s *Server) HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	email := stdEmail(r.FormValue("email"))
	if email == "" {
		webutils.ErrorResponse(w, r, webutils.ErrorCodef(ErrInvalidRequest, "valid email not specified in URL"), errorMap)
		return
	}

	signup, err := s.app.Signups.Unsubscribe(email)
	var signupError string
	signupSuccess := signup != nil && err == nil
	if !signupSuccess {
		if signup == nil {
			signupError = "email not found"
		} else {
			signupError = err.Error()
		}
	}
	signupRes := subscriptionResult{
		Success: signupSuccess,
		Error:   signupError,
	}

	user, err := s.app.Users.Unsubscribe(email)
	var userError string
	userSuccess := user != nil && err == nil
	if !userSuccess {
		if user == nil {
			userError = "email not found"
		} else {
			userError = err.Error()
		}
	}
	userRes := subscriptionResult{
		Success: userSuccess,
		Error:   userError,
	}

	resp := map[string]subscriptionResult{
		"signup": signupRes,
		"user":   userRes,
	}
	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// HandleSubscribe subscribes the signup/user with the given email
func (s *Server) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	email := stdEmail(r.FormValue("email"))
	if email == "" {
		webutils.ErrorResponse(w, r, webutils.ErrorCodef(ErrInvalidRequest, "valid email not specified in URL"), errorMap)
		return
	}

	signup, err := s.app.Signups.Subscribe(email)
	var signupError string
	signupSuccess := signup != nil && err == nil
	if !signupSuccess {
		if signup == nil {
			signupError = "email not found"
		} else {
			signupError = err.Error()
		}
	}
	signupRes := subscriptionResult{
		Success: signupSuccess,
		Error:   signupError,
	}

	user, err := s.app.Users.Subscribe(email)
	var userError string
	userSuccess := user != nil && err == nil
	if !userSuccess {
		if user == nil {
			userError = "email not found"
		} else {
			userError = err.Error()
		}
	}
	userRes := subscriptionResult{
		Success: userSuccess,
		Error:   userError,
	}

	resp := map[string]subscriptionResult{
		"signup": signupRes,
		"user":   userRes,
	}
	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// HandleListUnsubscribed returns the list of unsubscribed emails
func (s *Server) HandleListUnsubscribed(w http.ResponseWriter, r *http.Request) {
	secret := r.URL.Query().Get("secret")
	if err := validateSecretKey(secret); err != nil {
		webutils.ErrorResponse(w, r, err, errorMap)
		return
	}

	var signupEmails []string
	signups, err := s.app.Signups.ListUnsubscribed()
	if err != nil {
		webutils.ErrorResponse(w, r, err, errorMap)
		return
	}
	for _, s := range signups {
		signupEmails = append(signupEmails, s.Email)
	}

	var userEmails []string
	users, err := s.app.Users.ListUnsubscribed()
	if err != nil {
		webutils.ErrorResponse(w, r, err, errorMap)
		return
	}
	for _, u := range users {
		userEmails = append(userEmails, u.Email)
	}

	resp := map[string][]string{
		"signups": signupEmails,
		"users":   userEmails,
	}
	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// HandleCheckEmail returns a 200 if the provided email address is unused and valid, and 403 otherwise along with an error message.
func (s *Server) HandleCheckEmail(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Email string `json:"email"`
	}

	type response struct {
		FailReason    string `json:"fail_reason"`
		EmailInvalid  bool   `json:"email_invalid"`
		AccountExists bool   `json:"account_exists"`
		HasPassword   bool   `json:"has_password"`
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("community.Server.HandleCheckEmail: error reading request body: %v", err)
		http.Error(w, "unable to read request", http.StatusInternalServerError)
		return
	}

	var req request
	if err := json.Unmarshal(buf, &req); err != nil {
		log.Printf("community.Server.HandleCheckEmail: error unmarshaling json request: %v", err)
		http.Error(w, "malformed request", http.StatusBadRequest)
		return
	}

	if err := s.app.Users.CheckEmail(req.Email); err != nil {
		var resp response
		if err == ErrEmailInvalid {
			resp = response{
				FailReason:   err.Error(),
				EmailInvalid: true,
			}
		} else {
			user, _ := s.app.Users.FindByEmail(req.Email)
			resp = response{
				FailReason:    err.Error(),
				AccountExists: true,
				HasPassword:   len(user.HashedPassword) > 0,
			}
		}

		buf, errjson := json.Marshal(resp)
		if errjson != nil {
			log.Printf("community.Server.HandleCheckEmail: error marshalling json response: %v", errjson)
			http.Error(w, "unable to marshal response", http.StatusInternalServerError)
			return
		}

		// must set header information before setting error code
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write(buf)
	}
	// 200 automatically returned
}

// HandleCheckPassword returns a 200 if the provided password is valid, and 403 otherwise, with json explaining the reason
func (s *Server) HandleCheckPassword(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Password string `json:"password"`
	}

	type response struct {
		FailReason string `json:"fail_reason"`
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("community.Server.HandleCheckPassword: error reading request body: %v", err)
		http.Error(w, "unable to read request", http.StatusInternalServerError)
		return
	}

	var req request
	if err := json.Unmarshal(buf, &req); err != nil {
		log.Printf("community.Server.HandleCheckPassword: error unmarshaling json request: %v", err)
		http.Error(w, "malformed request", http.StatusBadRequest)
		return
	}

	if err := checkPassword(req.Password); err != nil {
		buf, merr := json.Marshal(response{FailReason: err.Error()})
		if merr != nil {
			log.Printf("community.Server.HandleCheckPassword: error marshaling response: %v", err)
			http.Error(w, "unable to marshal response", http.StatusInternalServerError)
			return
		}

		// must set content type before calling write header
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write(buf)
	}
	// 200 automatically returned
}

// emailVerification sends an email with verification link to the address to be verified.
func (s *Server) emailVerification(v *EmailVerification, host string) error {
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

// EmailPasswordReset emails a password reset email to a given user and password reset
func (s *Server) EmailPasswordReset(user *User, reset *PasswordReset, host string) error {
	values := url.Values{}
	values.Set("email", user.Email)
	values.Set("token", reset.Token)

	switch host {
	case domains.Alpha:
		host = domains.PrimaryHost
	case domains.Staging:
		host = domains.GaStaging
	}

	link := fmt.Sprintf("https://%s/reset-password?%s", host, values.Encode())
	parsedLink, err := url.Parse(link)
	if err != nil {
		rollbar.Error(err, link)
		return err
	}

	var body bytes.Buffer
	err = s.app.Templates.Render(&body, "reset-password-email.html", map[string]string{
		"ResetLink": parsedLink.String(),
		"Name":      user.Name,
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
		To:      []string{user.Email},
		Subject: "Reset your Kite password",
		Body:    body.Bytes(),
	})
}

func readSignup(r io.Reader) (*Signup, error) {
	var body Signup
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, webutils.ErrorCodef(ErrInvalidRequest, "error reading body of request: %v", err)
	}
	err = json.Unmarshal(data, &body)
	if err != nil {
		return nil, webutils.ErrorCodef(ErrInvalidRequest, "error unmarshaling body of request: %v", err)
	}
	return &body, nil
}

func readInviteData(r io.Reader) (*inviteData, error) {
	var body inviteData
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, webutils.ErrorCodef(ErrInvalidRequest, "error reading body of request: %v", err)
	}
	err = json.Unmarshal(data, &body)
	if err != nil {
		return nil, webutils.ErrorCodef(ErrInvalidRequest, "error unmarshaling body of request: %v", err)
	}
	return &body, nil
}

func validateSecretKey(secret string) error {
	if secret != defaultSecret {
		return webutils.ErrorCodef(ErrIncorrectSecret, "incorrect secret key specified")
	}
	return nil
}

// Deduce the client's IP address from the incoming http request. May return empty string
// if none of the headers or RemoteAddr have a valid ip.
func deduceClientIP(r *http.Request) string {
	headers := []string{"X-Forwarded-For",
		"HTTP_X_FORWARDED_FOR",
		"Proxy-Client-IP",
		"WL-Proxy-Client-IP",
		"HTTP_X_FORWARDED",
		"HTTP_X_CLUSTER_CLIENT_IP",
		"HTTP_CLIENT_IP",
		"HTTP_FORWARDED_FOR",
		"HTTP_FORWARDED",
		"HTTP_VIA",
		"REMOTE_ADDR",
	}
	for _, header := range headers {
		if origin := r.Header.Get(header); origin != "" {
			// if it's a list of ips, the first one is the origin IP so that's what we want
			if ip := strings.Split(origin, ", ")[0]; ip != "" && net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	// r.RemoteAddr may contain a proxy's IP instead of the original IP, so we check it last
	if net.ParseIP(r.RemoteAddr) != nil {
		return r.RemoteAddr
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return ""
}

// --

// SetSession sets a cookie in the HTTP response corresponding to the provided session
func SetSession(session *Session, w http.ResponseWriter, r *http.Request) {
	if session != nil {
		cookie := &http.Cookie{
			Name:    sessionKey,
			Value:   session.Key,
			Path:    "/",
			Expires: session.ExpiresAt,
		}
		http.SetCookie(w, cookie)
	}
}

// SessionKey returns the session key stored in the request.
func SessionKey(r *http.Request) (string, error) {
	cookie, err := r.Cookie(sessionKey)
	if err != nil {
		return "", err
	}
	return cookie.Value, err
}

// ClearSession will set the session cookie and machine cookie's MaxAge to -1 to clear it from the client's browser
func ClearSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionKey)
	if err == nil {
		cookie.MaxAge = -1
		http.SetCookie(w, cookie)
	}

	cookie, err = r.Cookie(machineCookieKey)
	if err == nil {
		cookie.MaxAge = -1
		http.SetCookie(w, cookie)
	}
}

// --

// setMachine sets a cookie in the HTTP response corresponding to the provided machineID
func setMachine(machineID string, w http.ResponseWriter, r *http.Request) {
	if machineID != "" {
		cookie := &http.Cookie{
			Name:    machineCookieKey,
			Value:   machineID,
			Path:    "/",
			Expires: time.Now().Add(defaultSessionExpiration),
		}
		http.SetCookie(w, cookie)
	}
}

// getMachine checks the headers and cookie for the machineID.
func getMachine(r *http.Request) string {
	mid := getMachineHeader(r)
	if mid == "" {
		mid = getMachineCookie(r)
	}
	return mid
}

// getMachineCookie retieves the machine ID cookie if set
func getMachineCookie(r *http.Request) string {
	cookie, err := r.Cookie(machineCookieKey)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// getMachineHeader retieves the machine ID from the machine header
func getMachineHeader(r *http.Request) string {
	return r.Header.Get(MachineHeader)
}

// --

// GetUser will return the User object associated with the logged in user. It will return
// nil if no user was found. Make sure to use this method only in handlers that are wrapped
// with UserValidation.Wrap.
func GetUser(r *http.Request) *User {
	if user, ok := r.Context().Value(userKey).(*User); ok {
		return user
	}
	return nil
}

// GetMachine will return the machine ID associated with the request. It will be filled in
// in methods that are wrapped with UserValidation.Wrap
func GetMachine(r *http.Request) string {
	if mid, ok := r.Context().Value(machineKey).(string); ok {
		return mid
	}

	return ""
}

func remoteIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	remoteIP := r.Header.Get("X-Forwarded-For")

	// Intermediate proxies should be prepending their IP in this field,
	// meaning the client IP is the first entry.
	parts := strings.Split(remoteIP, ",")
	if len(parts) > 0 {
		ipStr := strings.TrimSpace(parts[0])
		if ip := net.ParseIP(parts[0]); ip != nil {
			return ipStr
		}
		if host, _, err := net.SplitHostPort(parts[0]); err == nil {
			if ip := net.ParseIP(host); ip != nil {
				return host
			}
		}
	}

	log.Println("unable to extract IP from X-Forwarded-For:", remoteIP)

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}

	return host
}

func isDevIP(r *http.Request) bool {
	remoteIP := remoteIP(r)

	switch {
	// kite-dev VM
	case strings.HasPrefix(r.Host, "192.168.30.10"):
		return true
	// localhost
	case strings.HasPrefix(r.Host, "127.0.0.1"):
		return true
	// VPN
	case strings.HasPrefix(remoteIP, "10.86.0.13"):
		return true
	default:
		return false
	}
}
