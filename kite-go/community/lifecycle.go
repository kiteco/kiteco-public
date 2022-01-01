package community

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/customerio/go-customerio"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/mixpanel"
	"github.com/kiteco/kiteco/kite-golib/telemetry"
)

var (
	mpTracker  *mixpanel.Metrics
	cioTracker *customerio.CustomerIO
)

func init() {
	XXXXXXX
}

// SetToken sets the segment token to use for lifecycle events
func SetToken(mp, cioID, cio string) {
	mpTracker = nil
	cioTracker = nil
	if mp != "" {
		mpTracker = mixpanel.NewMetrics(mp)
	}
	if cio != "" {
		cioTracker = customerio.NewCustomerIO(cioID, cio)
	}
}

// IdentifyUser associates a user with the given traits. It auto-populates
// the user's email, first and full name, and account creation time if these
// traits are available. It also sets the user's acquisition channel if a
// non-empty channel is supplied.
func IdentifyUser(user *User, channel string) {
	traits := map[string]interface{}{
		"email":   user.Email,
		"created": user.CreatedAt.Unix(),
	}

	if user.Name != "" {
		traits["name"] = user.Name
		traits["first_name"] = firstName(user.Name)
	}

	if channel != "" {
		traits["channel"] = channel
	}

	updateUser(user.IDString(), traits)
}

// IdentifyEmailSignup identifies a new user who submitted his email. It
// auto-populates the user's email and signup time. It also marks the user
// as not having an actual account and sets the user's acquisition channel
// if a non-empty channel is supplied.
func IdentifyEmailSignup(email, channel string) {
	email = stdEmail(email)

	traits := map[string]interface{}{
		"email":   email,
		"created": time.Now().Unix(),
	}

	if channel != "" {
		traits["channel"] = channel
	}

	pseudoUpdateUser(email, []string{"no_account"}, traits)
}

// TrackNewUser is called when a user creates a new account.
func TrackNewUser(user *User, channel string) {
	IdentifyUser(user, channel)
	data := make(map[string]interface{})
	if channel != "" {
		data["channel"] = channel
	}
	track(user.IDString(), "account_created", data)
	anonymizeTrack(user.IDString(), "anonymous_account_created", data)
}

// TrackEmailSignup is called when a user submits his email without actually
// creating an account.
func TrackEmailSignup(email, channel string) {
	email = stdEmail(email)
	IdentifyEmailSignup(email, channel)
	data := make(map[string]interface{})
	if channel != "" {
		data["channel"] = channel
	}
	pseudoTrack(email, "email_submitted", nil, data)
}

// TrackMobileDownload is called when a user submits his email on a mobile device
// to get an email to download Kite on their computer
func TrackMobileDownload(email, channel string) {
	email = stdEmail(email)
	IdentifyEmailSignup(email, channel)
	data := make(map[string]interface{})
	if channel != "" {
		data["channel"] = channel
	}
	pseudoTrack(email, "mobile_email_download_requested", nil, data)
}

// TrackUserMobileDownload is called when a user buys a Kite Pro subscription.
func TrackUserMobileDownload(userID string, channel string) {
	data := make(map[string]interface{})
	if channel != "" {
		data["channel"] = channel
	}
	track(userID, "mobile_email_download_requested", data)
}

// AddTraits adds new traits or updates existing traits for a user
func AddTraits(id string, traits map[string]interface{}) {
	updateUser(id, traits)
}

// AddEmailSignupTraits adds new traits or updates existing traits for a user
// who only submitted his email
func AddEmailSignupTraits(email string, traits map[string]interface{}) {
	email = stdEmail(email)
	pseudoUpdateUser(email, []string{"no_account"}, traits)
}

// TrackDeleteUser is called when a user is deleted
func TrackDeleteUser(userID string) {
	updateUser(userID, map[string]interface{}{
		"deactivated": true,
	})
}

// TrackStartTrial is called when a user strats his Kite Pro trial.
func TrackStartTrial(userID string, l licensing.License, startType, ctaSrc string, loggedIn bool) {
	props := map[string]interface{}{
		"cta_source":   ctaSrc,
		"logged_in":    loggedIn,
		"start-type":   startType,
		"trial_until":  l.ExpiresAt,
		"trial_length": l.ExpiresAt.Sub(l.IssuedAt).Round(time.Second).Seconds(),
	}
	track(userID, "pro_trial_started", props)
}

// TrackTrialExtension is called when a user's trial is extended.
func TrackTrialExtension(userID string, prevEnd, newEnd time.Time, reason string) {
	track(userID, "pro_trial_extended", map[string]interface{}{
		"previous_trial_until": prevEnd.Unix(),
		"new_trial_until":      newEnd.Unix(),
		"reason":               reason,
	})
}

// TrackProPurchase is called when a user buys a Kite Pro subscription.
func TrackProPurchase(userID string, initialPurchase bool, props map[string]interface{}) {
	if initialPurchase {
		props["pro_start"] = fmt.Sprint(time.Now())
	}
	updateUser(userID, props)
	event := "renewal"
	if initialPurchase {
		event = "purchase"
	}
	track(userID, event, props)
}

// TrackProPaidTrialStarted is called when a user buys a Kite Pro subscription.
func TrackProPaidTrialStarted(userID string, trialDuration time.Duration, props map[string]interface{}) {
	updateUser(userID, props)

	props["trial_length"] = trialDuration.Round(time.Second).Seconds()
	track(userID, "pro_paid_trial_started", props)
}

// TrackFirstInvoicePaid sends an event to customerIO and mixpanel when the first invoice of a subscription is paid by the user
// We want to send a mail after this invoice with a link to the invoice attached
func TrackFirstInvoicePaid(userID string, hostedInvoiceURL string, invoicePDF string) {
	props := map[string]interface{}{
		"first_invoice_url": hostedInvoiceURL,
		"first_invoice_pdf": invoicePDF,
	}
	if cioTracker != nil {
		err := cioTracker.Identify(userID, props)
		if err != nil {
			log.Println("WARN Error while updating user status on customerIO after the first invoice payment")
		}
		err = cioTracker.Track(userID, "first_payment", props)
		if err != nil {
			log.Println("WARN Error while sending event to customerIO after the first invoice payment")
		}
	}
}

// TrackProCancel is called when a user cancels his Kite Pro subscription.
func TrackProCancel(userID string, userCancel bool) {
	cancelReason := "unpaid"
	if userCancel {
		cancelReason = "request"
	}
	updateUser(userID, map[string]interface{}{
		"cancel_reason": cancelReason,
	})
	track(userID, "pro_canceled", map[string]interface{}{
		"user_cancel": userCancel,
	})
}

// TrackProEducationalStarted is called when an educational license is provided to an user
// Origin allows to differenciate if the user click on checkout or start_trial before getting an educational license
func TrackProEducationalStarted(userID string, origin string, ctaSrc string) {
	updateUser(userID, map[string]interface{}{
		"pro_educational_started": time.Now(),
	})
	track(userID, "pro_educational_started", map[string]interface{}{
		"origin":     origin,
		"cta_source": ctaSrc,
	})
}

// TrackReferralEmailSent sends a referral email to referreeEmail with userID's referralLink
func TrackReferralEmailSent(userID string, referreeEmail, referralLink, referrerName string) {
	track(userID, "referral_sent", map[string]interface{}{
		"type":          "email",
		"referee_email": referreeEmail,
		"referral_link": referralLink,
		"referrer_name": referrerName,
	})
}

// TrackNPS is called when a user submits an NPS survey
func TrackNPS(userID string, score int, comment string) {
	comment = strings.Trim(comment, " ")
	track(userID, "nps_score_submitted", map[string]interface{}{
		"survey_source": "delighted",
		"score":         score,
		"comment":       comment,
		"no_comment":    len(comment) == 0,
	})
}

// --

func updateUser(userID string, traits map[string]interface{}) {
	var errs errors.Errors
	if mpTracker != nil {
		errs = errors.Append(errs, mpTracker.Identify(userID, traits))
	}
	if cioTracker != nil {
		errs = errors.Append(errs, cioTracker.Identify(userID, traits))
	}
	if errs != nil {
		log.Println(errs)
	}
}

func pseudoUpdateUser(id string, flags []string, traits map[string]interface{}) {
	if len(flags) > 0 && traits == nil {
		traits = make(map[string]interface{})
	}
	for _, flag := range flags {
		traits[flag] = true
	}

	var errs errors.Errors
	if mpTracker != nil {
		errs = errors.Append(errs, mpTracker.Identify(id, traits))
	}
	if cioTracker != nil {
		errs = errors.Append(errs, cioTracker.Identify(id, traits))
	}
	if errs != nil {
		log.Println(errs)
	}
}

func track(userID string, event string, props map[string]interface{}) {
	props = telemetry.AugmentProps(props)
	props["source"] = "community_server"
	props["user_id"] = userID

	var errs errors.Errors
	if mpTracker != nil {
		errs = errors.Append(errs, mpTracker.Track(userID, event, props))
	}
	if cioTracker != nil {
		errs = errors.Append(errs, cioTracker.Track(userID, event, props))
	}
	if errs != nil {
		log.Println(errs)
	}
}

func pseudoTrack(id string, event string, flags []string, props map[string]interface{}) {
	props = telemetry.AugmentProps(props)
	props["source"] = "community_server"
	props["user_id"] = id
	for _, flag := range flags {
		props[flag] = true
	}

	var errs errors.Errors
	if mpTracker != nil {
		errs = errors.Append(errs, mpTracker.Track(id, event, props))
	}
	if cioTracker != nil {
		errs = errors.Append(errs, cioTracker.Track(id, event, props))
	}
	if errs != nil {
		log.Println(errs)
	}
}

func anonymizeTrack(userID string, event string, props map[string]interface{}) {
	h := sha256.New()
	h.Write([]byte(userID))
	anonymousID := base64.StdEncoding.EncodeToString((h.Sum(nil)))
	trackAnonymous(anonymousID, event, props)
}

func trackAnonymous(id string, event string, props map[string]interface{}) {
	props = telemetry.AugmentProps(props)
	props["source"] = "community_server"
	props["anonymousID"] = id

	var errs errors.Errors
	if mpTracker != nil {
		errs = errors.Append(errs, mpTracker.Track(id, event, props))
	}
	if cioTracker != nil {
		errs = errors.Append(errs, cioTracker.Track(id, event, props))
	}
	if errs != nil {
		log.Println(errs)
	}
}

func firstName(name string) string {
	if name == "" {
		return ""
	}
	return strings.Title(strings.Split(strings.TrimLeft(name, " "), " ")[0])
}
