package stripe

import (
	"log"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/plan"
)

// PlanID is a string
type PlanID = string

// Plans contains the information about the different plan offered and a mutex to protect concurrent access
type Plans map[licensing.Plan]PlanID

// DefaultPlanIDs lists which Stripe IDs are preferred in case of duplicate kitePlanIDs
var DefaultPlanIDs = map[string]struct{}{
	"XXXXXXX": {}, // prod pro-yearly
	"XXXXXXX": {}, // prod pro-monthly
	"XXXXXXX": {}, // test pro-yearly
	"XXXXXXX": {}, // test pro-monthly
}

// PlansFromStripe fetches the plan mapping from Stripe
func PlansFromStripe() (Plans, error) {
	if err := checkInit(); err != nil {
		return nil, err
	}

	plans := make(Plans)

	stripePlans := plan.List(&stripe.PlanListParams{
		Active: stripe.Bool(true),
	})
	for stripePlans.Next() {
		stripePlan := stripePlans.Plan()

		kitePlanID, ok := stripePlan.Metadata["kite_plan"]
		if !ok {
			continue
		}

		kitePlan := licensing.Plan(kitePlanID)
		if !kitePlan.IsValid() {
			return nil, errors.Errorf("invalid kite_plan %s", kitePlan)
		}
		if !kitePlan.IsPaid() {
			return nil, errors.Errorf("kite_plan %s is not a paid plan", kitePlan)
		}
		if _, ok := DefaultPlanIDs[stripePlan.ID]; ok {
			plans[kitePlan] = stripePlan.ID
		}
	}

	for _, kitePlan := range licensing.PaidPlans {
		if _, ok := plans[kitePlan]; !ok {
			log.Printf("no Stripe plan with kite_plan %s", kitePlan)
		}
	}

	return plans, nil
}

func checkInit() error {
	if stripe.Key == "" {
		return errors.New("Please init stripe before calling this function")
	}
	if octobatPublicKey == "" || octobatSecretKey == "" {
		return errors.New("Please make sure that both octobat public and private key are initialized before calling this function (using InitOctobat function)")
	}
	return nil
}
