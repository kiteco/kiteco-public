package checkout

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/stripe"
)

// DefaultPlansResponse represents default plans of our app
type DefaultPlansResponse struct {
	Monthly int `json:"monthly"`
	Yearly  int `json:"yearly"`
}

// ProSubscriptionRequest represents customer, subscription and tax rate data that we needed to proceed with payments
type ProSubscriptionRequest struct {
	Customer     CustomerData     `json:"customer"`
	Subscription SubscriptionData `json:"subscription"`
	TaxRate      TaxRateData      `json:"tax_details"`
}

// ProSubscriptionResponse represents the response of accessing checkout api's
type ProSubscriptionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// SetupRoutes set up all the routes that is needed for checkout
func SetupRoutes(plans stripe.Plans, checkoutRouter *mux.Router) {
	checkoutRouter.HandleFunc("/default-plans", handleDefaultPlans(plans)).Methods(http.MethodGet)
	checkoutRouter.HandleFunc("/verify-coupon/{coupon_code}", handleVerifyCoupon).Methods(http.MethodGet)
	checkoutRouter.HandleFunc("/prices/{price_id}", handleGetPrice).Methods(http.MethodGet)
	checkoutRouter.HandleFunc("/pro-subscription", handleCreateProSubscription(plans)).Methods(http.MethodPost)
}

func handleDefaultPlans(plans stripe.Plans) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)

		proMonthlyPrice, errProMonthlyPrice := retrievePrice(plans[licensing.ProMonthly])
		if errProMonthlyPrice != nil {
			parseStripeErr(&w, errProMonthlyPrice)
			return
		}

		proYearlyPrice, errProYearlyPrice := retrievePrice(plans[licensing.ProYearly])
		if errProYearlyPrice != nil {
			parseStripeErr(&w, errProYearlyPrice)
			return
		}

		respondJSON(
			&w,
			&DefaultPlansResponse{
				Monthly: int(proMonthlyPrice.UnitAmount),
				Yearly:  int(proYearlyPrice.UnitAmount),
			},
		)
	}
}

func handleVerifyCoupon(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	vars := mux.Vars(r)

	coupon, errCoupon := retrieveCoupon(vars["coupon_code"])
	if errCoupon != nil {
		parseStripeErr(&w, errCoupon)
		return
	}

	respondJSON(&w, coupon)
}

func handleGetPrice(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	vars := mux.Vars(r)

	price, errPrice := retrievePrice(vars["price_id"])
	if errPrice != nil {
		parseStripeErr(&w, errPrice)
		return
	}

	respondJSON(&w, price)
}

func handleCreateProSubscription(plans stripe.Plans) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)

		var inputData ProSubscriptionRequest
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error occurred while parsing input data", http.StatusBadRequest)
		}
		log.Println("/pro-subscription request:", string(reqBody))

		errBodyParse := json.Unmarshal(reqBody, &inputData)
		if errBodyParse != nil {
			respondJSON(
				&w,
				&ProSubscriptionResponse{
					Success: false,
					Message: "Something went wrong.",
				},
			)
			return
		}

		// We should always receive a tax evidence ID
		if inputData.TaxRate.OctobatEvidenceID == "" {
			respondJSON(
				&w,
				&ProSubscriptionResponse{
					Success: false,
					Message: "Something went wrong.",
				},
			)
			return
		}

		customer, errCustomer := createCustomer(inputData.Customer)
		if errCustomer != nil {
			parseStripeErr(&w, errCustomer)
			return
		}

		stripeTaxRateIds, errStripeTaxRateIds := findOrCreateStripeTaxRate(inputData.TaxRate)
		if errStripeTaxRateIds != nil {
			parseStripeErr(&w, errStripeTaxRateIds)
			return
		}

		_, errSubscription := createSubscription(plans, customer, inputData.Subscription, stripeTaxRateIds, reqBody, inputData.TaxRate.OctobatEvidenceID)
		if errSubscription != nil {
			parseStripeErr(&w, errSubscription)
			return
		}

		respondJSON(
			&w,
			&ProSubscriptionResponse{
				Success: true,
				Message: "Please try again.",
			},
		)
	}
}

// findOrCreateStripeTaxRate finds or creates an appropriate tax rate
// https://docs.octobat.com/octobat/integrations-docs/stripe/tax-exclusive-integration#apply-to-subscriptions
func findOrCreateStripeTaxRate(taxDetails TaxRateData) ([]string, error) {
	// If all these fields are their 0-value, Octobat didn't return a tax rate with the evidence,
	// so we should return nil,nil because the user should be untaxed
	if taxDetails.Tax == "" &&
		taxDetails.Name == "" &&
		taxDetails.Jurisdiction == "" &&
		taxDetails.Rate == 0 {
		return nil, nil
	}

	stripeTaxRate := findStripeTaxRate(taxDetails)
	if stripeTaxRate != nil {
		return []string{stripeTaxRate.ID}, nil
	}

	newStripeTaxRate, errTaxRate := createStripeTaxRate(taxDetails)
	if errTaxRate != nil {
		return nil, errTaxRate
	}

	return []string{newStripeTaxRate.ID}, nil
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func respondJSON(w *http.ResponseWriter, response interface{}) {
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(*w, "Error occurred while parsing input data", http.StatusBadRequest)
	}

	(*w).Header().Set("Content-Type", "application/json")
	(*w).Write(jsonResponse)
}
