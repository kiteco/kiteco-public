package stripe

import (
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
)

// RetrieveStripeCustomer fetch the list of customer whose email match the one from kiteUser
func RetrieveStripeCustomer(kiteUser *community.User) []stripe.Customer {
	var result []stripe.Customer
	var params stripe.CustomerListParams
	params.Filters.AddFilter("limit", "", "20")
	params.Email = stripe.String(kiteUser.Email)
	params.AddExpand("data.subscriptions")
	i := customer.List(&params)
	for i.Next() {
		c := i.Customer()
		if c != nil {
			result = append(result, *c)
		}
	}
	return result
}

// GetActiveStripeSubscriptions retrieve all subscription associated to a customer with the same email address
// than kiteuser
func GetActiveStripeSubscriptions(kiteUser *community.User) ([]stripe.Subscription, error) {
	var result []stripe.Subscription
	customers := RetrieveStripeCustomer(kiteUser)
	if len(customers) == 0 {
		return nil, errors.Errorf("No stripe customer matching kiteUser email : %s", kiteUser.Email)
	}
	for _, c := range customers {
		for _, s := range c.Subscriptions.Data {
			switch s.Status {
			case stripe.SubscriptionStatusActive, stripe.SubscriptionStatusUnpaid, stripe.SubscriptionStatusPastDue:
				result = append(result, *s)
			}
		}
	}
	return result, nil
}
