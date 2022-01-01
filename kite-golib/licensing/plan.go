package licensing

// Product enumerates the Kite products
type Product string

// Products
const (
	Pro  Product = "pro"
	Free Product = "free"
)

// GetProduct implements ProductGetter
func (p Product) GetProduct() Product {
	return p
}

// Plan enumerates the Kite plans. Plans are associated with products.
type Plan string

// Plans
const (
	ProTrial        Plan = "pro_trial"
	ProEducation    Plan = "pro_education"
	ProMonthly      Plan = "pro_monthly"
	ProYearly       Plan = "pro_yearly"
	ProIndefinite   Plan = "pro_indefinite" // For sunsetting Kite. Is considered a subscription for fallback behavior.
	ProTrialMonthly Plan = "pro_trial_monthly"
	ProTrialYearly  Plan = "pro_trial_yearly"
	ProServer       Plan = "pro_server" // This plan is only used for Kite Server, and is not associated with real licenses.
	ProTemp         Plan = "pro_temp"   // This license is used for providing a license immediately after the payment
	// We don't know yet if the user subscribed for a monthly or yearly license so we generate this one while we wait
	// for stripe invoice.payment_succeed event

	FreePlan Plan = "free"
)

// PaidPlans contains all plans, which are paid. This includes plans with a paid trial.
var PaidPlans = []Plan{ProYearly, ProMonthly, ProTrialYearly, ProTrialMonthly}

// IsValid checks if the plan is a recognized plan
func (p Plan) IsValid() bool {
	switch p {
	case ProTrial, ProEducation, ProMonthly, ProYearly, ProTrialMonthly, ProTrialYearly, ProIndefinite, ProServer, ProTemp, FreePlan:
		return true
	}
	return false
}

// Product returns the Product for the given Plan
func (p Plan) Product() Product {
	switch p {
	case ProTrial, ProEducation, ProMonthly, ProYearly, ProTrialMonthly, ProTrialYearly, ProIndefinite, ProServer, ProTemp:
		return Pro
	case FreePlan:
		return Free
	default:
		panic("invalid plan")
	}
}

// IsPaid returns true if a license with this plan corresponds to a confirmed payment.
// Notably, this ProTemp licenses, since that is for unconfirmed payments.
func (p Plan) IsPaid() bool {
	switch p {
	case ProMonthly, ProYearly, ProTrialMonthly, ProTrialYearly, ProServer:
		return true
	}
	return false
}

// IsPaidPlanWithTrial returns true if the plan of this license has a trial
func (p Plan) IsPaidPlanWithTrial() bool {
	switch p {
	case ProTrialMonthly, ProTrialYearly:
		return true
	}
	return false
}

// IsSubscriber returns true if the plan is a "subscriber" plan.
// Free & trial licenses are not considered subscriber plans.
func (p Plan) IsSubscriber() bool {
	switch p {
	case FreePlan, ProTrial:
		return false
	}
	return true
}

// Order returns the order of products: lower is more preferable to use.
func (p Plan) Order() int {
	switch p {
	case ProServer:
		return 0
	case ProMonthly, ProYearly, ProTrialMonthly, ProTrialYearly:
		return 1
	case ProEducation:
		return 2
	case ProTemp:
		return 3
	case ProTrial:
		return 4
	default:
		return 5
	}
}
