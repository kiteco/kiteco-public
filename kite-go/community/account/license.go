package account

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/stripe/stripe-go/v72/sub"

	"github.com/jinzhu/gorm"

	"github.com/kiteco/kiteco/kite-golib/domains"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	kitestripe "github.com/kiteco/kiteco/kite-golib/stripe"
)

const (
	defaultTrialDays = 28
	maxTrialDays     = 90 // ~3 months
	gracePeriod      = 30 * 24 * time.Hour
)

var (
	defaultTrialDuration = time.Hour * 24 * defaultTrialDays
	maxTrialDuration     = time.Hour * 24 * maxTrialDays
	kiteProLaunchDate    = time.Date(2020, 5, 8, 0, 0, 0, 0, time.UTC)
)

func eduDuration(t time.Time) time.Time {
	return t.AddDate(0, 6, 0)
}

var (
	proInfoURL         = fmt.Sprintf("https://%s/pro", domains.WWW)
	trialConsumedURL   = fmt.Sprintf("https://%s/pro?source=trial_expired", domains.WWW)
	settingsURL        = fmt.Sprintf("https://%s/settings", domains.WWW)
	proStartedURL      = addEmailToURL(fmt.Sprintf("https://%s/pro/confirmation", domains.WWW))
	proTrialStartedURL = addEmailToURL(fmt.Sprintf("https://%s/pro/trial", domains.WWW))
	proEduStartedURL   = addEmailToURL(fmt.Sprintf("https://%s/pro/student", domains.WWW))
)

func addEmailToURL(urlStr string) func(identifier community.UserIdentifier) string {
	return func(u community.UserIdentifier) string {
		if u == nil {
			return urlStr
		}
		url, err := url.Parse(urlStr)
		if err != nil {
			panic(err)
		}
		q := url.Query()
		q["email"] = []string{u.GetEmail()}
		url.RawQuery = q.Encode()
		return url.String()
	}
}

// license created for users
// An user may have multiple licenses,
// but a single license can only be assigned to a one particular user
// because the license data contains the user ID
type license struct {
	// ID in the license table
	ID int64 `json:"-"`
	// ID in the user table
	UserID sql.NullInt64 `json:"-" gorm:"column:user_id"`
	// ID for anon users
	InstallID sql.NullString `json:"-" gorm:"column:install_id"`
	// Token is the license code, which is send to clients
	// assuming max size of 2k of the token, the typical size seems to be between 300-500 chars
	Token string `json:"token" gorm:"column:token" sql:"type:text;not null;"`
}

func (l license) License() *licensing.License {
	lic, err := licensing.ParseLicense(l.Token, nil)
	if err != nil {
		rollbar.Error(errors.New("error parsing license from DB"), err.Error())
		return nil
	}
	return lic
}

// licenseManager manages the license data in the db
type licenseManager struct {
	db gorm.DB
}

func newLicenseManager(db gorm.DB) *licenseManager {
	return &licenseManager{
		db: db,
	}
}

func (m *licenseManager) Migrate() error {
	if err := m.db.AutoMigrate(&license{}).Error; err != nil {
		return errors.Wrapf(err, "error migrating tables in licenses db")
	}
	if err := m.db.Model(&license{}).AddIndex("user_id_idx", "user_id").Error; err != nil {
		return errors.Wrapf(err, "error adding index")
	}
	if err := m.db.Model(&license{}).AddIndex("install_id_idx", "install_id").Error; err != nil {
		return errors.Wrapf(err, "error adding index")
	}
	if err := m.db.Exec("ALTER TABLE license ALTER COLUMN user_id DROP NOT NULL;").Error; err != nil {
		return errors.Wrapf(err, "error removing null from column user_id")
	}
	m.db.Exec("ALTER TABLE license ADD CONSTRAINT license_uid_iid_CK CHECK ((user_id IS NULL) <> (install_id IS NULL));")

	return nil
}

// Licenses returns all licenses, which are assigned to the given user and their anonymous identifier
// Both expired and active licenses are returned. It's the callers responsibility to filter the result.
// Licenses are returned in decreasing order of "preference."
func (m *licenseManager) Licenses(ids community.UserIdentifier) (*licensing.Licenses, error) {
	var licenses []license
	if err := m.db.Where(
		&license{
			UserID: sql.NullInt64{
				Int64: ids.UserID(),
				Valid: ids.UserID() != 0,
			},
		},
	).Or(
		&license{
			InstallID: sql.NullString{
				String: ids.AnonID(),
				Valid:  ids.AnonID() != "",
			},
		},
	).Find(&licenses).Error; err != nil {
		err = errors.Wrapf(err, "account.licenseManager.Licenses: acct %s: error getting license objects", ids.MetricsID())
		rollbar.Error(errors.New("error looking up licenses from DB"), err.Error())
		return nil, err
	}

	l := licensing.NewLicenses()
	for _, lic := range licenses {
		l.AddToken(lic.Token, nil)
	}
	return l, nil
}

// Create stores a new license in the licenses table
// The license properties are not validated by this method
func (m *licenseManager) Create(ids community.UserIdentifier, kiteLicense licensing.License) (*license, error) {
	if ids.AnonID() == "" && ids.UserID() == 0 {
		return nil, errors.Errorf("Cannot create license. One of AnonID and UserID must be non-zero.")
	}

	result := &license{
		UserID: sql.NullInt64{
			Int64: ids.UserID(),
			Valid: ids.UserID() != 0,
		},
		InstallID: sql.NullString{
			String: ids.AnonID(),
			// Prefer the authenticated user if both are given.
			Valid: ids.AnonID() != "" && ids.UserID() == 0,
		},
		Token: kiteLicense.Token,
	}

	if err := m.db.Create(result).Error; err != nil {
		return nil, errors.Wrapf(err, "account.licenseManager.Create: metrics_ID %s: error creating license object: %v", ids.MetricsID(), err)
	}
	return result, nil
}

//--- extensions to the account manager server

// Licenses returns all licenses of a user account
func (m *manager) Licenses(ids community.UserIdentifier) (*licensing.Licenses, error) {
	licenses, err := m.license.Licenses(ids)

	// no active license, but eligible for educational license:
	// automatically create a new educational license and send refreshed licenses
	if err == nil && licenses.License() == nil && m.users.IsStudent(ids) {
		err = m.GiveEducationalLicense(ids, "license-request", "")
		if err == nil {
			return m.license.Licenses(ids)
		}
	}

	return licenses, err
}

// CreateLicense stores a new license of a user account
func (m *manager) CreateLicense(ids community.UserIdentifier, kiteLicense licensing.License, reference string) (*license, error) {
	return m.license.Create(ids, kiteLicense)
}

// StartTrial starts a new trial for the given user if there's no trial yet
// clientTimeZoneOffset is the offset in seconds east of UTC
func (m *manager) StartTrial(user community.UserIdentifier, trialDuration time.Duration, startType, ctaSrc string, loggedIn bool, clientTimeZoneOffset int) error {
	lp := fmt.Sprintf("account.manager.StartTrial: metrics_ID %s:", user.MetricsID())
	licenses, err := m.Licenses(user)
	if err != nil {
		return errors.Wrapf(err, "%s error fetching licenses", lp)
	}

	currentLicense := licenses.License()
	switch {
	case currentLicense != nil && !currentLicense.IsExpired() && currentLicense.GetPlan() == licensing.ProTrial:
		// Trial already started
		return nil
	case !licenses.TrialAvailable():
		// no active trial, and trial consumed
		return errTrialAlreadyConsumed
	default:
		return m.createTrialLicense(user, trialDuration, licenses, startType, ctaSrc, loggedIn, clientTimeZoneOffset)
	}
}

// TODO use for trial-extension endpoint
func (m *manager) createExtensionTrial(ids community.UserIdentifier, extendFor time.Duration, licenses *licensing.Licenses, registerAsync bool) error {
	lp := fmt.Sprintf("account.manager.createExtensionLicense: metrics_id %s:", ids.MetricsID())

	curLicense := licenses.License()
	if curLicense == nil {
		return errors.Errorf("%s could not find license to extend", lp)
	}

	expiration := m.adjustExpiration(curLicense.ExpiresAt.Add(extendFor), 0)
	claims := licensing.Claims{
		UserID:    ids.UserIDString(),
		InstallID: ids.AnonID(),
		Product:   licensing.Pro,
		Plan:      licensing.ProTrial,
		IssuedAt:  time.Now(),
		ExpiresAt: expiration,
		PlanEnd:   expiration,
	}

	track := func(newLicense licensing.License) {
		community.TrackTrialExtension(ids.MetricsID(), curLicense.ExpiresAt, newLicense.ExpiresAt, "")
	}

	return m.createAndStoreLicense(ids, claims, licenses, track, registerAsync, lp)
}

// clientTimeZoneOffset is the offset in seconds east of UTC. Use 0 if the time zone is undefined.
func (m *manager) createTrialLicense(ids community.UserIdentifier, trialDur time.Duration, licenses *licensing.Licenses, startType, ctaSrc string, loggedIn bool, clientTimeZoneOffset int) error {
	lp := fmt.Sprintf("account.manager.createTrialLicense: metrics_id %s:", ids.MetricsID())

	now := time.Now()
	expiration := m.adjustExpiration(now.Add(trialDur), clientTimeZoneOffset)

	claims := licensing.Claims{
		UserID:    ids.UserIDString(),
		InstallID: ids.AnonID(),
		Product:   licensing.Pro,
		Plan:      licensing.ProTrial,
		IssuedAt:  now,
		ExpiresAt: expiration,
		PlanEnd:   expiration,
	}

	track := func(newLicense licensing.License) {
		community.TrackStartTrial(ids.MetricsID(), newLicense, startType, ctaSrc, loggedIn)
	}

	return m.createAndStoreLicense(ids, claims, licenses, track, false, lp)
}

// createAndStoreLicense creates a license from the claims and calls the passed track function with it
// This is to allow callers to customize their tracking since register may be called async
func (m *manager) createAndStoreLicense(ids community.UserIdentifier, claims licensing.Claims, licenses *licensing.Licenses, track func(licensing.License), registerAsync bool, lp string) error {
	license, err := m.authority.CreateLicense(claims)
	if err != nil {
		return errors.Wrapf(err, "%s error creating trial license", lp)
	}
	licenses.Add(license)

	licenseToDBAndTrack := func() error {
		_, err = m.license.Create(ids, *license)
		if err != nil {
			return errors.Wrapf(err, "%s error registering trial license", lp)
		}
		track(*license)
		return nil
	}

	// Don't block on DB, since DB latency can be high depending on the region
	if registerAsync {
		go func() {
			err := licenseToDBAndTrack()
			if err != nil {
				log.Println(lp, "error asynchronously registering license:", err)
			}
		}()
	}

	return licenseToDBAndTrack()
}

// GiveEducationalLicense creates a new educational license for the given user
// The license is created even if there's already an existing license.
// The caller is responsible to verify this beforehand.
func (m *manager) GiveEducationalLicense(user community.UserIdentifier, origin, ctaSrc string) error {
	now := time.Now()
	claims := licensing.Claims{
		UserID:   user.UserIDString(),
		Product:  licensing.Pro,
		Plan:     licensing.ProEducation,
		IssuedAt: now,
		PlanEnd:  eduDuration(now),
	}
	claims.ExpiresAt = claims.PlanEnd.Add(gracePeriod)

	license, err := m.authority.CreateLicense(claims)
	if err != nil {
		log.Println("Error generating educational license: ", err)
		return err
	}

	_, err = m.license.Create(user, *license)
	if err != nil {
		log.Println("Error storing educational license in database: ", err)
		return err
	}

	community.TrackProEducationalStarted(user.UserIDString(), origin, ctaSrc)
	return nil
}

func (m *manager) GiveEducationalLicenseAndRedirect(user community.UserIdentifier, origin string, ctaSrc string, w http.ResponseWriter, r *http.Request) {
	licenses, err := m.Licenses(user)
	if err != nil {
		log.Println("Error while fetching user licenses license:", err)
		http.Error(w, "Error while generating educational license, please contact support to get your free license", http.StatusInternalServerError)
		return
	}

	l := licenses.License()
	if l.IsPlanActive() && l.IsSubscriber() {
		if l.GetPlan() == licensing.ProEducation {
			http.Redirect(w, r, proEduStartedURL(user), http.StatusFound)
		} else {
			http.Redirect(w, r, proStartedURL(user), http.StatusFound)
		}
		return
	}

	err = m.GiveEducationalLicense(user, origin, ctaSrc)
	if err != nil {
		http.Error(w, "Error while generating educational license, please contact support to get your free license", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, proEduStartedURL(user), http.StatusFound)
}

//-- HTTP request handlers

// LicensesResponse is the object returned when the client query the list of licenses for an account
type LicensesResponse struct {
	LicenseTokens []string `json:"licenses"`
}

// handleLicenses is the request handler for /api/account/licenses
// it returns a list of all license tokens for the current user.
// it must be wrapped with authenticating request handlers
func (s *Server) handleLicenses(w http.ResponseWriter, r *http.Request) {
	lp := "account.Server.handleLicenses:"

	var user community.UserIdentifier
	u := community.GetUser(r)
	anon := community.GetAnonUser(r)

	switch {
	case u != nil && anon != nil:
		u.AnonUser = *anon
		user = u
	case u != nil:
		user = u
	case anon != nil:
		user = anon
	default:
		http.Error(w, lp+"failed to get valid user", http.StatusUnauthorized)
		return
	}

	licenses, err := s.m.Licenses(user)
	if err != nil {
		http.Error(w, "failed to retrieve licenses", http.StatusInternalServerError)
		return
	}

	resp := LicensesResponse{}
	for lic, next := licenses.Iterate(); lic != nil; lic = next() {
		resp.LicenseTokens = append(resp.LicenseTokens, lic.Token)
	}

	if err := marshalResponse(lp, w, &resp); err.HTTPError() {
		http.Error(w, err.Msg, err.Code)
	}

	// 200 returned automatically if no error
}

func (s *Server) handleLicenseInfo(w http.ResponseWriter, r *http.Request) {
	var user community.UserIdentifier
	u := community.GetUser(r)
	anon := community.GetAnonUser(r)

	switch {
	case u != nil && anon != nil:
		u.AnonUser = *anon
		user = u
	case u != nil:
		user = u
	case anon != nil:
		user = anon
	default:
		http.Error(w, "failed to get valid user", http.StatusUnauthorized)
		return
	}

	licenses, err := s.m.Licenses(user)
	if err != nil {
		http.Error(w, "failed to retrieve licenses", http.StatusInternalServerError)
		return
	}

	buf, err := json.Marshal(licenses.LicenseInfo())
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf) // 200 set automatically
}

// Generic start-trial endpoint for starting any trials (anon and non-anon)
// handleWebTrialStart is kept around for older versions of the client
func (s *Server) handleAPIStartTrial(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ctaSrc := q.Get("cta-source")
	startType := q.Get("start-type")

	var loggedIn bool
	var user community.UserIdentifier
	if u := community.GetUser(r); u != nil {
		user = u
		loggedIn = true
	} else if anon := community.GetAnonUser(r); anon != nil {
		user = anon
	} else {
		http.Error(w, "failed to get valid user", http.StatusUnauthorized)
		return
	}

	_, trialDuration := trialDurationFromRequest(r)
	err := s.m.StartTrial(user, trialDuration, startType, ctaSrc, loggedIn, 0)
	switch err {
	case nil:
	case errTrialAlreadyConsumed:
		http.Error(w, fmt.Sprintf("Trial already consumed for metrics_id: %s", user.MetricsID()), http.StatusUnauthorized)
	default:
		http.Error(w, fmt.Sprintf("Unable to start trial for metrics_id: %s", user.MetricsID()), http.StatusInternalServerError)
	}
}

func (s *Server) handleWebStartTrial(w http.ResponseWriter, r *http.Request) {
	user := community.GetUser(r)

	ctaSrc := r.URL.Query().Get("cta-source")

	if s.app.Users.IsStudent(user) {
		s.m.GiveEducationalLicenseAndRedirect(user, "trial", ctaSrc, w, r)
		return
	}

	clientTimeZoneOffset := 0
	if offset, err := strconv.Atoi(r.URL.Query().Get("timezoneOffset")); err == nil {
		clientTimeZoneOffset = offset
	}

	_, trialDuration := trialDurationFromRequest(r)
	// Use zero value "" for startType opt-in, since trials at pro launch all required opt-in.
	err := s.m.StartTrial(user, trialDuration, "", ctaSrc, true, clientTimeZoneOffset)
	switch err {
	case nil:
		// trial successfully started
		http.Redirect(w, r, proTrialStartedURL(user), http.StatusFound)
		return
	case errTrialAlreadyConsumed:
		// trial already started: forward user to subscribe
		http.Redirect(w, r, trialConsumedURL, http.StatusTemporaryRedirect)
		return
	default:
		// other failure
		// TODO KitePro (naman) where to redirect for this? we need an error page of sorts
		log.Println("failed to start trial", err)
		http.Redirect(w, r, fmt.Sprintf("https://%s/404", domains.PrimaryHost), http.StatusTemporaryRedirect)
		return
	}
}

// handleCheckout supports both checkout without and with a trial.
// the presence of query parameter trial-duration defines which checkout mode is used
func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	user, err := s.m.users.IsValidLogin(r)
	if err != nil {
		log.Printf("Error while fetching the user in handleCheckout: %s", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if s.m.users.IsStudent(user) {
		s.m.GiveEducationalLicenseAndRedirect(user, "checkout", "", w, r)
		return
	}

	if isPro, err := s.stripeCallbacks.isProSubscriber(user); err != nil {
		log.Printf("Error while fetching licenses info for user %s: %s", user.IDString(), err)
		http.Error(w, "Error while checking user status", http.StatusInternalServerError)
		return
	} else if isPro {
		// The user has a valid license we redirect to account settings
		http.Redirect(w, r, settingsURL, http.StatusFound)
		return
	}

	requestedTrial, trialDuration := trialDurationFromRequest(r)

	stripePlanID, err := s.findPlanID(r, requestedTrial)
	if err != nil {
		log.Println("Invalid plan:", err)
		http.Error(w, "invalid plan", http.StatusBadRequest)
		return
	}

	var data []byte
	if requestedTrial {
		data, err = kitestripe.CreateSubscriptionCheckoutOctobatWithTrial(trialDuration, user.Name, user.Email, user.IDString(), r.RemoteAddr, stripePlanID, r.Host, proInfoURL)
	} else {
		data, err = kitestripe.CreateSubscriptionCheckoutOctobat(user.Name, user.Email, user.IDString(), r.RemoteAddr, stripePlanID, r.Host, proInfoURL)
	}

	if err != nil {
		log.Println("Error while getting checkout payload : ", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(data)
	if err != nil {
		log.Println("Error while writing checkout response: ", err)
	}
}

func (s *Server) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	user := community.GetUser(r)
	if user == nil {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	isProSubscriber, err := s.stripeCallbacks.isProSubscriber(user)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	marshalResponse("handleSubscriptions", w, map[string]interface{}{
		"isProSubscriber": isProSubscriber,
	})
}

func (s *Server) handleSubscriptionsDelete(w http.ResponseWriter, r *http.Request) {
	user := community.GetUser(r)
	if user == nil {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}
	subscriptions, _ := kitestripe.GetActiveStripeSubscriptions(user)

	for _, s := range subscriptions {
		_, err := sub.Cancel(s.ID, nil)
		if err != nil {
			log.Println("Error: handleCancelPlan error while canceling subscription:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func (s *Server) findPlanID(r *http.Request, withTrial bool) (kitestripe.PlanID, error) {
	var plan licensing.Plan

	q := r.URL.Query()
	if s, p := q["subscription"], q["plan"]; len(s)+len(p) != 1 {
		return "", errors.New(fmt.Sprintf("expected exactly one value for `plan` query parameter, got %d", len(s)+len(p)))
	} else if len(s) > 0 {
		plan = licensing.Plan(s[0])
	} else {
		plan = licensing.Plan(p[0])
	}

	switch string(plan) {
	case "monthly":
		switch withTrial {
		case true:
			plan = licensing.ProTrialMonthly
		case false:
			plan = licensing.ProMonthly
		}

	case "yearly":
		switch withTrial {
		case true:
			plan = licensing.ProTrialYearly
		case false:
			plan = licensing.ProYearly
		}
	}

	stripePlanID, ok := s.stripeCallbacks.plans[plan]
	if !ok {
		return "", errors.New(fmt.Sprintf("invalid subscription (plan) %s", plan))
	}

	return stripePlanID, nil
}

func trialDurationFromRequest(r *http.Request) (bool, time.Duration) {
	durationValue := r.URL.Query().Get("trial-duration")
	trialDuration, err := time.ParseDuration(durationValue)
	if err != nil {
		trialDuration = defaultTrialDuration
	}
	hasTrial := durationValue != "" && err == nil

	if trialDuration > maxTrialDuration {
		trialDuration = maxTrialDuration
	}
	return hasTrial, trialDuration
}
