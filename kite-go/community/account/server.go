package account

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/domains"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/mixpanel"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/kiteco/kiteco/kite-golib/stripe"
	"github.com/quickemailverification/quickemailverification-go"
)

// Server is an http wrapper around a customer manager.
type Server struct {
	m                *manager
	app              *community.App
	slackManager     *slackManager
	discourseManager *discourseManager
	mixpanelManager  *mixpanel.JQLClient
	delightedManager *delightedManager
	gaProxy          *httputil.ReverseProxy
	quickEmailClient *quickemailverification.Client
	stripeCallbacks  *stripeCallbacks
}

// NewServer returns a pointer to a new customer management server.
func NewServer(app *community.App, slackToken, discourseSecret, mixpanelSecret, delightedSecret, quickEmailToken string, authority *licensing.Authority, plans stripe.Plans) *Server {
	slackm := newSlackManager(slackToken)
	accounts := newManager(app.DB, app.Users, authority)
	discourse := newDiscourseManager(discourseSecret)
	mixpanel := mixpanel.NewJQLClient(mixpanelSecret)
	delighted := newDelightedManager(delightedSecret)
	stripeCbk := stripeCallbacks{
		manager: accounts,
		plans:   plans,
	}
	quickEmail := quickemailverification.CreateClient(quickEmailToken)
	s := &Server{
		app:              app,
		m:                accounts,
		slackManager:     slackm,
		discourseManager: discourse,
		mixpanelManager:  mixpanel,
		delightedManager: delighted,
		gaProxy:          newGAProxy(),
		quickEmailClient: quickEmail,
		stripeCallbacks:  &stripeCbk,
	}
	return s
}

// Migrate the relevant DBs for the server
func (s *Server) Migrate() error {
	return s.m.Migrate()
}

// SetupRoutes uses the provided router and user authorization to set up the API endpoints
// for customer management and stripe webhooks.
func (s *Server) SetupRoutes(router *mux.Router) {
	// user/account creation endpoints
	router.HandleFunc("/create-web", s.handleCreateWeb).Methods("POST")
	router.HandleFunc("/login-web", s.handleLoginWeb).Methods("POST")
	router.HandleFunc("/login-desktop", s.handleLoginDesktop).Methods("POST")

	router.HandleFunc("/invite-emails", s.handleSendEmailInvites).Methods("POST")

	// invite to flywithkite slack
	router.HandleFunc("/invite-slack", s.handleSlackInvite).Methods("POST")

	// login to discourse forum
	router.HandleFunc("/forum-login", s.app.Auth.Wrap(s.handleDiscourseSSO)).Methods("POST")

	// record newsletter signup
	router.HandleFunc("/newsletter", s.handleNewsletter).Methods("POST")

	// record pycon signup
	router.HandleFunc("/pycon-signup", s.handlePyconSignup).Methods("POST")

	// verify user signup attempt
	router.HandleFunc("/verify-newsletter", s.handleUserEmailVerify).Methods("POST")

	// send mobile download email
	router.HandleFunc("/mobile-download", s.handleMobileDownload).Methods("POST")

	// google analytics request
	router.HandleFunc("/kite-google{rest:.*}", http.StripPrefix("/api/account/kite-google", s.gaProxy).ServeHTTP)

	// Webhooks
	webhooks := router.PathPrefix("/webhooks/").Subrouter()
	webhooks.HandleFunc("/activated", s.HandleActivatedUser).Methods("POST")
	webhooks.HandleFunc("/delighted-survey", s.HandleDelightedEvent).Methods("POST")

	// - licensing
	router.HandleFunc("/start-trial", s.app.Auth.WrapAllowAnon(s.handleAPIStartTrial)).Methods("POST")
	router.HandleFunc("/licenses", s.app.Auth.WrapAllowAnon(s.handleLicenses)).Methods("GET")
	router.HandleFunc("/license-info", s.app.Auth.WrapAllowAnon(s.handleLicenseInfo)).Methods("GET")
	router.HandleFunc("/subscriptions", s.app.Auth.Wrap(s.handleSubscriptions)).Methods("GET")
	router.HandleFunc("/subscriptions", s.app.Auth.Wrap(s.handleSubscriptionsDelete)).Methods("DELETE")
	router.HandleFunc("/stripe-webhook", stripe.HandleStripeWebhookRequest(s.stripeCallbacks))

	// - deprecated routes, return static response until we are sure they can be removed
	// user does not need to have an account yet to access this endpoint, so we cannot use WrapAccount
	router.HandleFunc("/details", s.handleAccountDetails).Methods("GET")
}

// SetupWebRoutes ...
func (s *Server) SetupWebRoutes(router *mux.Router) {
	// - licensing
	router.HandleFunc("/start-trial", s.app.Auth.WrapRedirect(fmt.Sprintf("https://%s/login/start-trial", domains.PrimaryHost), s.handleWebStartTrial)).Methods("GET")
	router.HandleFunc("/checkout/start", s.app.Auth.WrapRedirect(fmt.Sprintf("https://%s/login/upgrade-pro", domains.PrimaryHost), s.handleCheckout))
	router.HandleFunc("/checkout/octobat-success", stripe.HandleOctobatSuccess(s.stripeCallbacks))
}

// handleSlackInvite handles inviting an email address to the flywithkite slack channel
func (s *Server) handleSlackInvite(w http.ResponseWriter, r *http.Request) {
	lp := "account.Server.handleSlackInvite:"

	var request struct {
		Email string `json:"email"`
	}

	if ed := unmarshalBody(lp, r, &request); ed.HTTPError() {
		http.Error(w, ed.Msg, ed.Code)
		return
	}

	err := s.slackManager.Invite(request.Email)
	if err != nil {
		rollbar.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

// handleDiscourseSSO handles logging in a user to the discourse forum
func (s *Server) handleDiscourseSSO(w http.ResponseWriter, r *http.Request) {
	lp := "account.Server.handleDiscourseSSO:"

	user := community.GetUser(r)

	if user == nil {
		http.Error(w, "authorized", http.StatusUnauthorized)
		return
	}

	request := ForumLogin{}
	if ed := unmarshalBody(lp, r, &request); ed.HTTPError() {
		http.Error(w, ed.Msg, ed.Code)
		return
	}

	response, err := s.discourseManager.SingleSignOn(request, user)
	if err != nil {
		http.Error(w, "unable to log in", http.StatusInternalServerError)
		return
	}

	if err := marshalResponse(lp, w, response); err.HTTPError() {
		http.Error(w, err.Msg, err.Code)
	}
}

func (s *Server) handlePyconSignup(w http.ResponseWriter, r *http.Request) {
	lp := "account.Server.handlePyconSignup:"

	var request struct {
		Email     string `json:"email"`
		Pycon2019 bool   `json:"pycon_2019"`
		Channel   string `json:"channel"`
	}

	if ed := unmarshalBody(lp, r, &request); ed.HTTPError() {
		http.Error(w, ed.Msg, ed.Code)
		return
	}

	traits := map[string]interface{}{
		"pycon_2019": request.Pycon2019,
		"channel":    request.Channel,
	}

	if user, _ := s.app.Users.FindByEmail(request.Email); user != nil {
		community.AddTraits(user.IDString(), traits)
		w.WriteHeader(http.StatusOK)
		return
	}

	community.TrackEmailSignup(request.Email, "homepage_pycon")
	community.AddEmailSignupTraits(request.Email, traits)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleNewsletter(w http.ResponseWriter, r *http.Request) {
	lp := "account.Server.handleNewsletter:"

	var request struct {
		Email      string `json:"email"`
		Newsletter bool   `json:"newsletter"`
		Channel    string `json:"channel"`
	}

	if ed := unmarshalBody(lp, r, &request); ed.HTTPError() {
		http.Error(w, ed.Msg, ed.Code)
		return
	}

	traits := map[string]interface{}{
		"newsletter": request.Newsletter,
		"channel":    request.Channel,
	}

	if user, _ := s.app.Users.FindByEmail(request.Email); user != nil {
		community.AddTraits(user.IDString(), traits)
		w.WriteHeader(http.StatusOK)
		return
	}

	community.TrackEmailSignup(request.Email, "homepage_newsletter")
	community.AddEmailSignupTraits(request.Email, traits)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleUserEmailVerify(w http.ResponseWriter, r *http.Request) {
	lp := "account.Server.handleUserEmailVerify:"

	var request struct {
		Email string `json:"email"`
	}

	if ed := unmarshalBody(lp, r, &request); ed.HTTPError() {
		http.Error(w, ed.Msg, ed.Code)
		return
	}

	resp, err := s.quickEmailClient.Verify(request.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var response struct {
		Verified bool `json:"verified"`
	}

	// NOTE: We should treat unknowns as valid to reduce false negatives
	// http://docs.quickemailverification.com/getting-started/understanding-email-verification-result
	if resp.Result == "valid" || resp.Result == "unknown" {
		response.Verified = true
	}

	if ed := marshalResponse(lp, w, response); ed.HTTPError() {
		http.Error(w, ed.Msg, ed.Code)
	}
}

func (s *Server) handleMobileDownload(w http.ResponseWriter, r *http.Request) {
	lp := "account.Server.handleMobileDownload:"

	var request struct {
		Email   string `json:"email"`
		Channel string `json:"channel"`
	}

	if ed := unmarshalBody(lp, r, &request); ed.HTTPError() {
		http.Error(w, ed.Msg, ed.Code)
		return
	}

	traits := map[string]interface{}{
		"channel": request.Channel,
	}

	if user, _ := s.app.Users.FindByEmail(request.Email); user != nil {
		community.AddTraits(user.IDString(), traits)
		community.TrackUserMobileDownload(user.IDString(), "mobile_homepage")
		w.WriteHeader(http.StatusOK)
		return
	}

	community.TrackMobileDownload(request.Email, "mobile_homepage")
	community.AddEmailSignupTraits(request.Email, traits)
	w.WriteHeader(http.StatusOK)
}
