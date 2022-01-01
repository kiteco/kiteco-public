package mockserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/community/account"
	"github.com/kiteco/kiteco/kite-golib/gziphttp"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

const (
	//must be same value as defined the backend server
	sessionKey = "kite-session"

	tokenHeaderKey     = "Kite-Token"
	tokenDataHeaderKey = "Kite-TokenData"
)

// NewBackend returns a new instance of a mocked backend server. It currently supports only a few requests:
//  POST /api/account/login-desktop
//  GET  /api/account/logout
//  GET  /api/account/authenticated
//  GET  /api/account/user
//  GET  /api/account/plan
//  POST /http/events
//
// Dummy requests:
//  GET  /api/ping
//  GET  /api/pingCookie
//
// The server counts the requests and provides variables to retrieve these numbers.
// TODO(joachim): make sure that the mock server is safe for concurrent use
func NewBackend(validUserCredentials map[string]string) (*MockBackendServer, error) {
	router := mux.NewRouter()

	httpServer := httptest.NewServer(router)
	u, err := url.Parse(httpServer.URL)
	if err != nil {
		return nil, err
	}

	authority, err := licensing.NewTestAuthority()
	if err != nil {
		return nil, err
	}

	server := &MockBackendServer{
		Server:                httpServer,
		URL:                   u,
		router:                router,
		validUsersCredentials: validUserCredentials,
		authority:             authority,
		validUsers:            map[string]community.User{},
		proUsers:              map[string]bool{},
		licenses:              map[string][]*licensing.License{},
		loggedInUser:          nil,
		requestCounts:         map[string]int64{},
	}

	server.setupAccountRoutes(router)
	return server, nil
}

// MockBackendServer provides a simple implementation to be used in unit tests
type MockBackendServer struct {
	Server *httptest.Server
	//URL is the servers http urls
	URL *url.URL

	//guards backend properties
	lock                  sync.Mutex
	validUsersCredentials map[string]string               //email -> password
	proUsers              map[string]bool                 //email -> isPro
	licenses              map[string][]*licensing.License //email -> []*License
	validUsers            map[string]community.User       //email -> User, allows to define the returned values for the login request
	loggedInUser          *community.User
	requestCounts         map[string]int64

	router    *mux.Router
	authority *licensing.Authority
}

// Authority returns the authority used by this mock backend to sign new licenses
func (s *MockBackendServer) Authority() *licensing.Authority {
	return s.authority
}

// AddValidUser adds a new user to the list of users which are accepted by the backend
func (s *MockBackendServer) AddValidUser(user community.User, password string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.validUsers[user.Email] = user
	s.validUsersCredentials[user.Email] = password
}

// SetUserPlan updates the plan staus of a given user
func (s *MockBackendServer) SetUserPlan(email string, isPro bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.proUsers[email] = isPro
}

// AddLicense adds a new license
func (s *MockBackendServer) AddLicense(license *licensing.License) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.licenses[license.UserID] = append(s.licenses[license.UserID], license)
}

// RemoveLicense removes a license
func (s *MockBackendServer) RemoveLicense(license *licensing.License) {
	s.lock.Lock()
	defer s.lock.Unlock()

	userLicenses := s.licenses[license.UserID]
	for i, lic := range userLicenses {
		if lic == license {
			userLicenses = append(userLicenses[0:i], userLicenses[i+1:]...)
			break
		}
	}
	s.licenses[license.UserID] = userLicenses
}

// NotFound replies to the request with an HTTP 404 not found error.
func (s *MockBackendServer) notFound(w http.ResponseWriter, r *http.Request) {
	s.IncrementRequestCount(fmt.Sprintf("not-found: %s %s", r.Method, r.URL.Path))

	log.Printf("Backend request: %v\n", r.URL)
	http.Error(w, "404 page not found", http.StatusNotFound)
}

func (s *MockBackendServer) setupAccountRoutes(router *mux.Router) {
	s.AddPrefixRequestHandler("/api/account/login-desktop", []string{"POST"}, s.handleLoginDesktop)
	s.AddPrefixRequestHandler("/api/account/create-web", []string{"POST"}, s.handleCreateWeb)
	s.AddPrefixRequestHandler("/api/account/createPasswordless", []string{"POST"}, s.handleCreatePasswordless)
	s.AddPrefixRequestHandler("/api/account/logout", []string{"GET"}, s.handleLogoutDesktop)
	s.AddPrefixRequestHandler("/api/account/authenticated", []string{"GET"}, s.handleAuthenticated)
	s.AddPrefixRequestHandler("/api/account/user", []string{"GET"}, s.handleGetUser)
	s.AddPrefixRequestHandler("/api/account/licenses", []string{"GET"}, s.handleGetLicenses)
	s.AddPrefixRequestHandler("/http/events", []string{"POST"}, gziphttp.Wrap(s.handleEditorEvent))

	s.AddPrefixRequestHandler("/api/pingCookie", []string{"GET"}, s.handlePingCookie)
	s.AddPrefixRequestHandler("/api/ping", []string{"GET"}, s.handlePing)

	//log all unimplemented requests
	router.NotFoundHandler = http.HandlerFunc(s.notFound)
}

func (s *MockBackendServer) handleAuthenticated(w http.ResponseWriter, r *http.Request) {
	s.IncrementRequestCount("/api/account/authenticated")
	http.SetCookie(w, &http.Cookie{
		Name:  "kite-session",
		Value: "mock",
		Path:  "/",
	})
}

func (s *MockBackendServer) handleGetUser(w http.ResponseWriter, r *http.Request) {
	s.IncrementRequestCount(r.URL.Path)

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.loggedInUser == nil {
		http.Error(w, "No user logged in", http.StatusUnauthorized)
		return
	}

	// provide custom user data, if available
	user := s.loggedInUser
	if u, ok := s.validUsers[s.loggedInUser.Email]; ok {
		user = &u
	}
	json.NewEncoder(w).Encode(user)
}

func (s *MockBackendServer) handleGetLicenses(w http.ResponseWriter, r *http.Request) {
	s.IncrementRequestCount("licenses")

	s.lock.Lock()
	defer s.lock.Unlock()

	if s.loggedInUser == nil {
		http.Error(w, "No user logged in", http.StatusUnauthorized)
		return
	}

	licenses := s.licenses[s.loggedInUser.IDString()]
	var tokens []string
	for _, license := range licenses {
		tokens = append(tokens, license.Token)
	}

	json.NewEncoder(w).Encode(account.LicensesResponse{
		LicenseTokens: tokens,
	})
}

func (s *MockBackendServer) handleLogoutDesktop(w http.ResponseWriter, r *http.Request) {
	s.IncrementRequestCount("logout")

	s.lock.Lock()
	defer s.lock.Unlock()

	s.loggedInUser = nil
}

func (s *MockBackendServer) handleCreateWeb(w http.ResponseWriter, r *http.Request) {
	s.IncrementRequestCount("create")
	s.handleNewUser(true, false, w, r)
}

func (s *MockBackendServer) handleCreatePasswordless(w http.ResponseWriter, r *http.Request) {
	s.IncrementRequestCount("create-passwordless")
	s.handleNewUser(false, true, w, r)
}

func (s *MockBackendServer) handleNewUser(requirePassword, requireChannel bool, w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	channel := r.FormValue("channel")

	s.lock.Lock()
	defer s.lock.Unlock()

	// create struct for validation
	user := &community.User{
		Email: email,
	}

	// validate email
	if _, err := govalidator.ValidateStruct(user); err != nil {
		http.Error(w, fmt.Sprintf("invalid email %s", email), http.StatusBadRequest)
		return
	}

	if requirePassword {
		// validate password
		if len(password) < 6 {
			http.Error(w, "password is too short", http.StatusBadRequest)
			return
		}
		if len(password) > 55 {
			http.Error(w, "password is too long", http.StatusBadRequest)
			return
		}
	}

	if requireChannel {
		if channel != "test-channel" {
			http.Error(w, "expected channel: test-channel", http.StatusBadRequest)
			return
		}
	}

	// add to validUsers
	s.validUsersCredentials[email] = password
	user = &community.User{
		Email:         email,
		Name:          email,
		PasswordSalt:  []byte(password),
		EmailVerified: true,
		ID:            7,
	}
	s.validUsers[email] = *user
	s.loggedInUser = user
	// marshal user into response
	// marshal user into response
	buf, err := json.Marshal(s.loggedInUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Add(tokenHeaderKey, "token-key-value")
	w.Header().Add(tokenDataHeaderKey, "token-data-value")

	// kited will only authenticate if we return a cookie
	http.SetCookie(w, &http.Cookie{
		Name:    sessionKey,
		Value:   "session-key",
		Path:    "/",
		Expires: time.Now().Add(time.Hour),
	})

	w.Write(buf)
}

func (s *MockBackendServer) handleLoginDesktop(w http.ResponseWriter, r *http.Request) {
	s.IncrementRequestCount("login")

	email := r.FormValue("email")
	password := r.FormValue("password")

	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.validUsersCredentials[email]; !ok {
		http.Error(w, fmt.Sprintf("invalid email %s", email), http.StatusBadRequest)
		return
	}

	if s.validUsersCredentials[email] != password {
		http.Error(w, fmt.Sprintf("invalid password for %s", email), http.StatusBadRequest)
		return
	}

	// update current user
	if u, ok := s.validUsers[email]; ok {
		//predefined value
		s.loggedInUser = &u
	} else {
		//fallback value
		s.loggedInUser = &community.User{Email: email, Name: email, PasswordSalt: []byte(password), EmailVerified: true, ID: 42}
	}

	// marshal user into response
	buf, err := json.Marshal(s.loggedInUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Add(tokenHeaderKey, "token-key-value")
	w.Header().Add(tokenDataHeaderKey, "token-data-value")

	//kited will only authenticate if we return a cookie
	http.SetCookie(w, &http.Cookie{
		Name:    sessionKey,
		Value:   "session-key",
		Path:    "/",
		Expires: time.Now().Add(time.Hour),
	})

	w.Write(buf)
}

// handles /api/ping, adds a cookie named kite-ping-cookie
func (s *MockBackendServer) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pingback"))

	s.IncrementRequestCount("ping")
}

// handles /api/pingCookie, adds a cookie named kite-ping-cookie
func (s *MockBackendServer) handlePingCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "kite-ping-cookie", Value: "kite-ping-cookie-value"})
	w.Write([]byte("pingback"))

	s.IncrementRequestCount("pingCookie")
}

// AddPrefixRequestHandler allows the caller to customize the backend request handling
func (s *MockBackendServer) AddPrefixRequestHandler(path string, methods []string, handlerFunc http.HandlerFunc) {
	s.lock.Lock()
	defer s.lock.Unlock()

	//override an existing node, if possible. We use the path prefix as a route's name to simplify the lookup
	route := s.router.Get(path)
	if route == nil {
		route = s.router.NewRoute().PathPrefix(path).Name(path)
	}

	route.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// don't do keep-alive connections for test-cases
		w.Header().Add("Connection", "Close")
		handlerFunc(w, r)
	}).Methods(methods...)
}

// Close frees up resources taken by this test server
func (s *MockBackendServer) Close() {
	s.Server.Close()
}

// IncrementRequestCount increments the value of the counter identified by name
func (s *MockBackendServer) IncrementRequestCount(name string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.requestCounts[name] = s.requestCounts[name] + 1
}

// GetRequestCount returns the current value of the counter identified by name
func (s *MockBackendServer) GetRequestCount(name string) int64 {
	s.lock.Lock()
	defer s.lock.Unlock()

	count := s.requestCounts[name]
	return count
}

// ResetRequestCount returns the current value of the counter identified by name
func (s *MockBackendServer) ResetRequestCount() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.requestCounts = make(map[string]int64)
}

// PingRequestCount returns how many requests to the ping endpoint were received
func (s *MockBackendServer) PingRequestCount() int64 {
	return s.GetRequestCount("ping")
}

// PingCookieRequestCount returns how many requests to the pingCookie endpoint were received
func (s *MockBackendServer) PingCookieRequestCount() int64 {
	return s.GetRequestCount("pingCookie")
}

// CreateAccountRequestCount returns how many requests to the create endpoint were received
func (s *MockBackendServer) CreateAccountRequestCount() int64 {
	return s.GetRequestCount("create")
}

// CreatePasswordlessAccountRequestCount returns how many requests to the create endpoint were received
func (s *MockBackendServer) CreatePasswordlessAccountRequestCount() int64 {
	return s.GetRequestCount("create-passwordless")
}

// LoginRequestCount returns how many requests to the login endpoint were received
func (s *MockBackendServer) LoginRequestCount() int64 {
	return s.GetRequestCount("login")
}

// LogoutRequestCount returns how many requests to the logout endpoint were received
func (s *MockBackendServer) LogoutRequestCount() int64 {
	return s.GetRequestCount("logout")
}

// PlanRequestCount returns how many requests to the plan endpoint were received
func (s *MockBackendServer) PlanRequestCount() int64 {
	return s.GetRequestCount("plan")
}

// CountDebugString returns a debug string which lists how many times requests were called
func (s *MockBackendServer) CountDebugString() string {
	s.lock.Lock()
	defer s.lock.Unlock()

	lines := []string{}
	for k, v := range s.requestCounts {
		lines = append(lines, fmt.Sprintf("%s: %d", k, v))
	}
	sort.Strings(lines)

	return strings.Join(lines, "\n")
}
