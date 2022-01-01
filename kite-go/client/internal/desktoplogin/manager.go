package desktoplogin

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/clienttelemetry"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/domains"
)

const (
	nonceEndpoint = "/account/login-nonce"
	loginEndpoint = "/account/desktop-login"
)

var (
	productionTarget *url.URL
	stagingTarget    *url.URL
)

func init() {
	var err error
	productionTarget, err = url.Parse("https://" + domains.WWW)
	if err != nil {
		log.Fatalln(err)
	}

	stagingTarget, err = url.Parse("https://" + domains.GaStaging)
	if err != nil {
		log.Fatalln(err)
	}
}

// Manager wraps desktop login methods/handlers
type Manager struct {
	proxy    component.AuthClient
	userIds  userids.IDs
	settings component.SettingsManager
}

// Name implements component Core
func (m *Manager) Name() string {
	return "desktop-login"
}

// NewManager creates a new manager
func NewManager() *Manager {
	return &Manager{}
}

// Initialize implements component.Initializer
func (m *Manager) Initialize(opts component.InitializerOptions) {
	m.proxy = opts.AuthClient
	m.userIds = opts.UserIDs
	m.settings = opts.Settings
}

// RegisterHandlers implements components Handler
func (m *Manager) RegisterHandlers(mux *mux.Router) {
	mux.HandleFunc("/clientapi/desktoplogin", m.handleDesktopLogin)
	mux.HandleFunc("/clientapi/desktoplogin/start-trial", m.handleStartTrial)
}

// HandleDesktopLogin generates the redirect to the backend with a nonce value, forwarding
// along the requested url location.
func (m *Manager) handleDesktopLogin(w http.ResponseWriter, r *http.Request) {
	usedCounterDistribution.Add(1)

	dest := r.URL.Query().Get("d")
	if dest == "" {
		log.Println("error in desktop login: requires query param 'd'")
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	target := m.desktopLogin(r.Context(), dest, false)
	if target == "" {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (m *Manager) handleStartTrial(w http.ResponseWriter, r *http.Request) {
	dest := "/web/account/start-trial"

	queries := url.Values{}
	if src := r.URL.Query().Get("cta-source"); src != "" {
		queries.Add("cta-source", src)
		clienttelemetry.Event("cta_clicked", map[string]interface{}{
			"cta_source": src,
		})
		// see full list of sources in sidebar/src/store/license.tsx
		// TODO KitePro check if the source is valid?
	}
	if trialDur, err := m.settings.GetDuration(settings.TrialDuration); err == nil {
		queries.Add("trial-duration", trialDur.String())
	}

	_, timeZoneOffset := time.Now().Zone()
	queries.Add("timezoneOffset", strconv.Itoa(timeZoneOffset))

	dest = dest + "?" + queries.Encode()

	target := m.desktopLogin(r.Context(), dest, true)
	if target == "" {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, target, http.StatusFound)
}

func (m *Manager) desktopLogin(ctx context.Context, dest string, hitAPINode bool) string {
	// target is the url for the backend host
	baseURL := m.proxy.Target()
	if baseURL == nil {
		log.Println("error desktop login: no backend target")
		return ""
	}

	if !hitAPINode {
		switch baseURL.Host {
		case domains.Alpha:
			baseURL = productionTarget
		case domains.Staging:
			baseURL = stagingTarget
		}
	}

	destURL, err := baseURL.Parse(dest)
	if err != nil {
		log.Printf("error parsing final destination %s: %s", dest, err)
		return ""
	}

	// detect known WP routes, and go to staging WP Engine instead of staging webapp
	if baseURL == stagingTarget {
		switch destURL.EscapedPath() {
		case "/plans", "/pro", "/pro/student", "/pro/confirmation", "/pro/trial":
			destURL.Host = "XXXXXXX"
		}
	}

	nonce, err := m.generateNonce(ctx)
	if err != nil {
		log.Println("error desktop login: generating nonce:", err)

		// If we can't generate the nonce, perform the redirect, bypassing
		// the desktop-login flow so that the user will be at the right page, but
		// will have to login first.

		return destURL.String()
	}

	redirect, err := m.proxy.Parse(loginEndpoint)
	if err != nil {
		log.Println("error parsing provided destination:", err, dest)
		return ""
	}

	values := make(url.Values)
	values.Add("n", nonce)
	values.Add("d", destURL.String())

	return fmt.Sprintf("%s?%s", redirect.String(), values.Encode())
}

// SendToDesktopLogin writes a HTTP redirect into the HTTP response which redirect to the given destination on the Kite server
func (m *Manager) SendToDesktopLogin(w http.ResponseWriter, r *http.Request, destination string) {
	// update request url query parameters with destination and localtoken
	// and then forward request to desktop login which will then redirect to www.kite.com/trial
	params := make(url.Values)
	params.Add("d", destination)

	r.URL.RawQuery = params.Encode()

	m.handleDesktopLogin(w, r)
}

// --

type nonceData struct {
	Value string
}

func (m *Manager) generateNonce(ctx context.Context) (string, error) {
	// If we're not logged in, don't attempt to generate the nonce
	if _, err := m.proxy.GetUser(); err != nil {
		return "", fmt.Errorf("not logged in, bypassing nonce")
	}

	resp, err := m.proxy.Get(ctx, nonceEndpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("got status code %d when requesting nonce", resp.StatusCode)
	}

	respBuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data nonceData
	err = json.Unmarshal(respBuf, &data)
	if err != nil {
		return "", err
	}

	if data.Value == "" {
		return "", fmt.Errorf("got empty nonce value")
	}

	return data.Value, nil
}
