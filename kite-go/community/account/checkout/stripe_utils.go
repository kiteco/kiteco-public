package checkout

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	kiteStripe "github.com/kiteco/kiteco/kite-golib/stripe"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/coupon"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/price"
	"github.com/stripe/stripe-go/v72/sub"
	"github.com/stripe/stripe-go/v72/taxrate"
)

// CustomerData represents client data which is needed to create stripe Customer
type CustomerData struct {
	Email           string         `json:"email"`
	Name            string         `json:"name"`
	PaymentMethodID string         `json:"payment_method_id"`
	Coupon          string         `json:"coupon_code"`
	Address         stripe.Address `json:"address"`
}

// SubscriptionData represents client data which is needed to create stripe Subscription
type SubscriptionData struct {
	UserID               int    `json:"user_id"`
	Coupon               string `json:"coupon_code"`
	BillingCycle         string `json:"billing_cycle"`
	ConfigAnnualPriceID  string `json:"annual_plan_id"`
	ConfigMonthlyPriceID string `json:"monthly_plan_id"`
	ConfigTrialDays      int64  `json:"trial_days"`
}

// createCustomer creates new Customer, attaches PaymentMethod and Coupon if provided
func createCustomer(data CustomerData) (*stripe.Customer, error) {
	// we need to keep this here for compatibility as the webapp is deployed asynchronously to the backend.
	if data.Coupon != "" {
		return nil, errors.Errorf("may not apply coupon to Stripe Customer")
	}
	customerParams := &stripe.CustomerParams{
		Email:         stripe.String(data.Email),
		Description:   stripe.String(data.Name),
		PaymentMethod: stripe.String(data.PaymentMethodID),
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(data.PaymentMethodID),
		},
		Address: &stripe.AddressParams{
			Line1:      stripe.String(""),
			Country:    &data.Address.Country,
			PostalCode: &data.Address.PostalCode,
		},
	}

	return customer.New(customerParams)
}

// createSubscription creates new subscription with the help of provided customer, payment plan and taxes
func createSubscription(
	plans kiteStripe.Plans,
	customer *stripe.Customer,
	subscriptionData SubscriptionData,
	taxRates []string,
	originatingRequest []byte,
	OctobatTaxEvidenceID string,
) (*stripe.Subscription, error) {
	var stripePlanID kiteStripe.PlanID

	if subscriptionData.BillingCycle == "annual" {
		if subscriptionData.ConfigAnnualPriceID != "" {
			stripePlanID = subscriptionData.ConfigAnnualPriceID
		} else {
			stripePlanID = plans[licensing.ProYearly]
		}
	} else {
		if subscriptionData.ConfigMonthlyPriceID != "" {
			stripePlanID = subscriptionData.ConfigMonthlyPriceID
		} else {
			stripePlanID = plans[licensing.ProMonthly]
		}
	}

	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customer.ID),
		Coupon:   stripe.String(subscriptionData.Coupon),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Plan:     stripe.String(stripePlanID),
				TaxRates: stripe.StringSlice(taxRates),
			},
		},
	}

	if subscriptionData.ConfigTrialDays != 0 {
		trialDays := subscriptionData.ConfigTrialDays

		if trialDays > 30 {
			trialDays = 30
		}

		params.TrialPeriodDays = stripe.Int64(trialDays)
	}

	// Stripe has a 500char limit on strings
	truncatedRequest := string(originatingRequest)
	if len(truncatedRequest) > 500 {
		truncatedRequest = truncatedRequest[:500]
	}
	params.AddMetadata("kite_pro_subscription_request", truncatedRequest)

	params.AddMetadata("kite_user_id", fmt.Sprint(subscriptionData.UserID))
	params.Items[0].AddMetadata("kite_user_id", fmt.Sprint(subscriptionData.UserID))

	// Add Octobat Tax Evidence ID (even if customer is untaxed)
	params.Items[0].AddMetadata("octobat:tax_evidence", OctobatTaxEvidenceID)

	params.AddExpand("latest_invoice.payment_intent")

	return sub.New(params)
}

// retrieveCoupon retrieves Coupon from Stripe API
func retrieveCoupon(code string) (*stripe.Coupon, error) {
	return coupon.Get(code, nil)
}

// retrievePrice retrieves Price from Stripe API
func retrievePrice(priceID string) (*stripe.Price, error) {
	return price.Get(priceID, nil)
}

// TaxRateData represents client data needed for the creation of stripe TaxRate
type TaxRateData struct {
	OctobatEvidenceID string `json:"id"`
	Tax               string `json:"tax"`
	Name              string `json:"name"`
	// Rate is a tax percentage in [0,100]
	Rate         float64 `json:"rate"`
	Jurisdiction string  `json:"jurisdiction"`
}

// createStripeTaxRate creates a Stripe TaxRate with the help of provided Octobat data
func createStripeTaxRate(taxParams TaxRateData) (*stripe.TaxRate, error) {
	params := &stripe.TaxRateParams{
		DisplayName:  stripe.String(taxParams.Tax),
		Description:  stripe.String(fmt.Sprintf("%s - %s", taxParams.Name, taxParams.Tax)),
		Jurisdiction: stripe.String(taxParams.Jurisdiction),
		Percentage:   stripe.Float64(taxParams.Rate),
		Inclusive:    stripe.Bool(false),
		Params: stripe.Params{
			Metadata: map[string]string{
				"version": "2",
			},
		},
	}

	return taxrate.New(params)
}

// findStripeTaxRate finds an appropriate Stripe TaxRate object for the given tax information.
// Returns nil if there is no appropriate object.
func findStripeTaxRate(taxDetails TaxRateData) *stripe.TaxRate {
	// Note that stripe-go performs auto-pagination.
	// https://stripe.com/docs/api/pagination/auto
	taxList := taxrate.List(&stripe.TaxRateListParams{
		Active:    stripe.Bool(true),
		Inclusive: stripe.Bool(false),
	})

	for taxList.Next() {
		taxRate := taxList.TaxRate()

		if taxRate.Percentage == taxDetails.Rate &&
			taxRate.Jurisdiction == taxDetails.Jurisdiction {
			return taxRate
		}
	}

	return nil
}

// parseStripeErr parses stripe error in to json and sends it to client
func parseStripeErr(w *http.ResponseWriter, err error) {
	var response *ProSubscriptionResponse
	stripeErr, ok := err.(*stripe.Error)
	if !ok {
		response = &ProSubscriptionResponse{
			Success: false,
			Message: "Please try again.",
		}
	} else {
		response = &ProSubscriptionResponse{
			Success: stripeErr.HTTPStatusCode == http.StatusOK,
			Message: stripeErr.Msg,
		}
	}

	jsonResponse, _ := json.Marshal(response)

	http.Error((*w), string(jsonResponse), stripeErr.HTTPStatusCode)
}
