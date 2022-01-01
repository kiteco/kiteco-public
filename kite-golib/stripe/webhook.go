package stripe

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/kiteco/kiteco/kite-golib/errors"

	"github.com/dgrijalva/jwt-go"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/webhook"
)

var stripeWebhookSecret string

// InitStripeWebhookSecret set the stripe webhook secret to use to validate events
// It must be called before any event get processed
func InitStripeWebhookSecret(secret string) {
	stripeWebhookSecret = secret
}

// WebhookCallbacks interface represents all the entrypoint we are currently for processing stripe webhooks
type WebhookCallbacks interface {
	// CustomerSubscriptionUpdated is triggered on subscription update (creation, past_due, activated)
	CustomerSubscriptionUpdated(subscription stripe.Subscription, event stripe.Event) error
	// InvoicePaymentSucceed is triggered on each payment, that's when new license get generated
	InvoicePaymentSucceed(invoice stripe.Invoice) error
	// OctobatPaymentSuccess is triggered synchronously on success after payment with octobat checkout system
	// it returns a redirection URL in non-error cases.
	OctobatPaymentSuccess(payload map[string]interface{}) (string, error)
}

// HandleStripeWebhookRequest is an entry point for validating and parsing stripe event
// It delegates the actual processing of the event to the callbacks provided in argument
func HandleStripeWebhookRequest(callbacks WebhookCallbacks) func(w http.ResponseWriter, r *http.Request) {
	if stripeWebhookSecret == "" {
		panic("StripeWebhookSecret empty, please initialize it before calling this function (using InitStripeWebhookSecret)")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		const MaxBodyBytes = int64(65536)
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
		payload, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %v\n", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Construct Event allows to validate event signature against webhook secret
		// see https://stripe.com/docs/webhooks/signatures
		evt, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"),
			stripeWebhookSecret)
		if err != nil {
			log.Printf("Failed to construct webhook event : %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = nil
		// Unmarshal the event data into an appropriate struct depending on its Type
		switch evt.Type {
		case "invoice.payment_succeeded":
			// https://stripe.com/docs/api/events/types#event_types-invoice.payment_succeeded
			var invoice stripe.Invoice
			if err = json.Unmarshal(evt.Data.Raw, &invoice); err != nil {
				err = errors.Wrapf(err, "error parsing webhook JSON")
				break
			}
			err = callbacks.InvoicePaymentSucceed(invoice)
		case "customer.subscription.updated", "customer.subscription.created", "customer.subscription.deleted":
			// https://stripe.com/docs/api/events/types#event_types-customer.subscription.updated
			// https://stripe.com/docs/api/events/types#event_types-customer.subscription.deleted
			// https://stripe.com/docs/api/events/types#event_types-customer.subscription.created
			var subscription stripe.Subscription
			if err = json.Unmarshal(evt.Data.Raw, &subscription); err != nil {
				err = errors.Wrapf(err, "error parsing webhook JSON")
				break
			}
			err = callbacks.CustomerSubscriptionUpdated(subscription, evt)
		default:
			log.Printf("Unexpected event type: %s\n", evt.Type)
		}

		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// HandleOctobatSuccess validates octobat checkout success events and delegate the processing of the
// event to the callbacks provided as argument
// It uses the octobatSecretKey to validate the JWT token passed as transaction_details URL parameter
func HandleOctobatSuccess(callbacks WebhookCallbacks) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		tok, ok := r.URL.Query()["transaction_details"]
		if !ok || len(tok) == 0 {
			log.Println("Error, transaction_details URL param missing in OctobatSuccess request")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// don't do claims validation, since it's ok if the issue-at time is slightly in the future
		// TODO(naman) if/once jwt-go 4.0 is available, upgrading will fix the issue,
		// and we can re-enable claims validation: https://github.com/dgrijalva/jwt-go/issues/246
		var parser jwt.Parser
		parser.SkipClaimsValidation = true
		token, err := parser.Parse(tok[0], func(token *jwt.Token) (interface{}, error) {
			if m, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || m.Name != jwt.SigningMethodHS256.Name {
				return nil, errors.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(octobatSecretKey), nil
		})
		if err != nil {
			log.Println("Error while parsing JWT token : ", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			fmt.Println("Claims from token : ", claims)
			dest, err := callbacks.OctobatPaymentSuccess(claims)
			if err != nil {
				log.Println("Error while processing OctobatPaymentSuccess : ", err)
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				http.Redirect(w, r, dest, 301)
			}
		} else {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
