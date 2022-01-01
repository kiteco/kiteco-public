package account

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/octobat"
	kitestripe "github.com/kiteco/kiteco/kite-golib/stripe"
	"github.com/stripe/stripe-go/v72"
)

type stripeCallbacks struct {
	manager *manager
	plans   kitestripe.Plans
}

func (s *stripeCallbacks) CustomerSubscriptionUpdated(subscription stripe.Subscription, event stripe.Event) error {
	userID := subscription.Metadata["kite_user_id"]
	if userID == "" {
		return errors.Errorf("CustomerSubscriptionUpdated Missing userID in subscription metadata coming from stripe event")
	}

	if event.Type == "customer.subscription.deleted" {
		userCancel := event.Request == nil
		community.TrackProCancel(userID, userCancel)
		return nil
	}

	if event.Type == "customer.subscription.updated" {
		// Track Kite Pro subscriptions with trials.
		// A subscription update event is the only place where it's possible to find out
		// if it's a transition trialing -> active.
		// This webhook is possibly more inaccurate than "payment succeeded",
		// because it's unable to check if a new license has already been created.
		// Therefore all other tracking happens in "payment succeeded".
		kitePlan, ok := subscription.Plan.Metadata["kite_plan"]
		if ok && licensing.Plan(kitePlan).IsPaidPlanWithTrial() {
			paidAfterTrial := event.GetPreviousValue("status") == "trialing" && subscription.Status == stripe.SubscriptionStatusActive
			end := time.Unix(subscription.CurrentPeriodEnd, 0)
			paid := subscription.Plan.Amount
			community.TrackProPurchase(userID, paidAfterTrial, s.createEventProps(licensing.Plan(kitePlan), subscription.LatestInvoice.ID, subscription.LatestInvoice.HostedInvoiceURL, end, paid))
		}
	}

	return nil
}

func (s *stripeCallbacks) InvoicePaymentSucceed(invoice stripe.Invoice) error {
	if invoice.Lines == nil || len(invoice.Lines.Data) != 1 {
		return errors.Errorf("expected exactly one line item")
	}
	lineItem := invoice.Lines.Data[0]

	if lineItem.Plan == nil {
		return errors.New("line item with no plan")
	}

	var plan licensing.Plan
	planID, ok := lineItem.Plan.Metadata["kite_plan"]
	if !ok {
		return errors.New("line item with plan with no kite_plan metadata")
	}
	plan = licensing.Plan(planID)

	userIDStr, ok := lineItem.Metadata["kite_user_id"]
	if !ok {
		return errors.New("line item with no kite_user_id metadata")
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return errors.Wrapf(err, "line item with non-uint64 kite_user_id metadata %s", userIDStr)
	}

	if lineItem.Period == nil || lineItem.Period.End == 0 {
		return errors.New("line item with no period end")
	}

	// if the user is in cc-required trial, then issue a license, which is valid until the trial ends
	// the first non-trial payment will trigger this method again to create a license until the end
	// of the current subscription period.
	var licenseCreated bool
	var end time.Time
	if plan.IsPaidPlanWithTrial() {
		// we're using stripe's trial end data instead of now()+kite_trial_duration because the payment could've been delayed
		// invoice.Subscription.TrialEnd isn't available here, because the subscription just has an ID property
		end = time.Unix(lineItem.Period.End, 0)
		// no grace period for trials
		licenseCreated, _, err = s.checkAndCreateLicense(userID, plan, s.createProvID(invoice.ID), end, end)
	} else {
		end = time.Unix(lineItem.Period.End, 0)
		licenseCreated, _, err = s.checkAndCreateLicense(userID, plan, s.createProvID(invoice.ID), end.Add(gracePeriod), end)
	}

	// We only track event if the license has been created as it means this event is not a duplicate
	if licenseCreated {
		if plan.IsPaidPlanWithTrial() {
			trialSecondsValue, ok := lineItem.Metadata[octobat.MetadataKeyTrialDuration]
			if !ok {
				return errors.New("line item without expected %s", octobat.MetadataKeyTrialDuration)
			}

			trialSeconds, err := strconv.Atoi(trialSecondsValue)
			if err != nil {
				return err
			}

			// transition trialing->active of plans with a trial is handled in the "subscription updated" webhook
			if invoice.BillingReason == stripe.InvoiceBillingReasonSubscriptionCreate {
				props := s.createEventProps(plan, invoice.ID, invoice.HostedInvoiceURL, end, invoice.AmountPaid)
				community.TrackProPaidTrialStarted(userIDStr, time.Second*time.Duration(trialSeconds), props)
			}
		} else {
			s.trackInvoicePaid(userIDStr, invoice, end, plan)
		}
	}

	return err
}

func (s *stripeCallbacks) trackInvoicePaid(userID string, invoice stripe.Invoice, endSub time.Time, plan licensing.Plan) {
	switch invoice.BillingReason {
	case stripe.InvoiceBillingReasonSubscriptionCreate:
		community.TrackProPurchase(userID, true, s.createEventProps(plan, invoice.ID, invoice.HostedInvoiceURL, endSub, invoice.AmountPaid))
		community.TrackFirstInvoicePaid(userID, invoice.HostedInvoiceURL, invoice.InvoicePDF)
	case stripe.InvoiceBillingReasonSubscriptionCycle:
		community.TrackProPurchase(userID, false, s.createEventProps(plan, invoice.ID, invoice.HostedInvoiceURL, endSub, invoice.AmountPaid))
	}
}

func (s *stripeCallbacks) OctobatPaymentSuccess(payload map[string]interface{}) (string, error) {
	beanieData, ok := payload["beanie"].(map[string]interface{})
	if !ok {
		return "", errors.Errorf("The transaction_details token returned by octobat hasn't the right format")
	}
	userIDStr, ok := beanieData["client_reference_id"]
	if !ok {
		return "", errors.Errorf("beanie.client_reference_id missing from transaction details token (Octobat)")
	}
	userID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		return "", errors.Wrapf(err, "Error while parsing the accountID from an octobat payment success")
	}
	n := time.Now()
	n = n.AddDate(0, 0, 5)

	ref, ok := beanieData["order_id"]
	if !ok {
		return "", errors.Errorf("beanie.order_id missing from transaction details token (Octobat)")
	}

	reference := fmt.Sprintf("octobat_checkout:%v", ref)
	_, user, err := s.checkAndCreateLicense(userID, licensing.ProTemp, reference, n, n)

	return proStartedURL(user), err
}

// returns a bool if a license was created, a user, and an error.
// a license may not be created in a non-erroneous case if a license already exists with the given plan/provID.
func (s *stripeCallbacks) checkAndCreateLicense(userID int64, plan licensing.Plan, provID string, end, subscriptionEnd time.Time) (bool, *community.User, error) {
	user, err := s.manager.users.Get(userID)
	if err != nil {
		return false, nil, errors.Wrapf(err, "checkAndCreateLicense: Couldn't find an user for ID %d", userID)
	}
	licenses, err := s.manager.license.Licenses(user)
	if err != nil {
		return false, nil, err
	}
	// We first check if a license has already been created for this period
	// Webhooks can be triggered multiple time, so we prefer to make sure the event processing is idempotent

	for l, next := licenses.Iterate(); l != nil; l = next() {
		if l.Plan == plan && l.ProvID == provID {
			// license with same provenance already exists, we don't need to generate a new one
			return false, user, nil
		}
	}
	license, err := s.manager.authority.CreateLicense(licensing.Claims{
		UserID:    user.IDString(),
		Product:   licensing.Pro,
		Plan:      plan,
		ProvID:    provID,
		PlanEnd:   subscriptionEnd,
		ExpiresAt: end,
	})
	if err != nil {
		return false, nil, errors.Wrapf(err, "Error while creating a pro license")
	}

	tx := s.manager.db.Begin()
	_, err = s.manager.license.Create(user, *license)
	if err != nil {
		tx.Rollback()
		return false, nil, err
	}
	return true, user, nil
}

// InitStripe initialize stripe and octobat with the required secret keys. Use test keys for any kind of testing
func InitStripe(stripeSecret, stripeWebhookSecret, octobatSecret, octobatPublishable, beanieConfigID string) {
	if stripeSecret == "" || octobatSecret == "" || octobatPublishable == "" {
		log.Fatalf("Stripe or Octobat secrets are empty")
	}
	stripe.Key = stripeSecret
	kitestripe.InitStripeWebhookSecret(stripeWebhookSecret)
	kitestripe.InitOctobat(octobatPublishable, octobatSecret, beanieConfigID)
}

// isProSubscriber checks if the user has an active subscriber license.
// If the user does have such a license, we confirm with Stripe before returning true.
func (s *stripeCallbacks) isProSubscriber(user *community.User) (bool, error) {
	licenses, err := s.manager.Licenses(user)
	if err != nil {
		return false, err
	}

	l := licenses.License()
	if l == nil {
		return false, nil
	}

	log.Printf("plan end : %v and now %v\n", l.PlanEnd, time.Now())
	if !l.IsPlanActive() || !l.IsSubscriber() {
		return false, nil
	}
	if !l.IsPaid() {
		// non-paid subscriber, no need to check with Stripe
		return true, nil
	}

	subs, err := kitestripe.GetActiveStripeSubscriptions(user)
	if err != nil {
		return false, errors.Wrapf(err, "Error while checking if user is pro subscriber")
	}
	if len(subs) == 0 {
		return false, nil
	}

	return true, nil
}

func (s *stripeCallbacks) createProvID(invoiceID string) string {
	return fmt.Sprintf("stripe_invoice:%s", invoiceID)
}

func (s *stripeCallbacks) createEventProps(plan licensing.Plan, invoiceID, invoiceURL string, subscriptionEnd time.Time, amountPaid int64) map[string]interface{} {
	// we can't use invoice.Line, because invoice might just have an ID, e.g. when used with "subscription updated"
	return map[string]interface{}{
		"price":       fmt.Sprint(amountPaid),
		"term":        fmt.Sprint(subscriptionEnd),
		"ref":         s.createProvID(invoiceID),
		"invoice_url": invoiceURL,
		"plan":        plan,
	}
}
