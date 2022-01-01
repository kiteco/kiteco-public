package account

import (
	"net/http"
	"time"
)

// STILL IN DB (Migrate operation)

// member of an organization
type member struct {
	// ID in the member table
	ID int64 `json:"-"`
	// AccountID for the member (foreign key to account in account table)
	AccountID int64 `json:"-" gorm:"column:account_id"`
	// OrganizationID for the organization that they are members of (foreign key to organization in the organization table)
	OrganizationID int64 `json:"-" gorm:"column:organization_id"`
	// Email for the member
	Email string `json:"email"`
	// PlanName that the member is subscribed to
	PlanName string `json:"plan_name"`
}

// account is the basic information associated with a user
type account struct {
	// ID is the primary key in the account table
	ID int64
	// UserID is a foreign key to the UserID in the community database
	UserID int64 `gorm:"column:user_id"`
	// StripeCustomerID is the id for the stripe customer object associated with this account
	StripeCustomerID string
	// CreatedAt tracks when the account was created
	CreatedAt time.Time
}

// subscription for an account or organization
type subscription struct {
	// ID of the subscription
	ID int64

	// StripeID for the subscription
	// This will be empty for lifetime subscriptions
	StripeID string

	// PlanID for the subscription
	PlanID string

	// Quantity of the subscription
	Quantity uint64

	// CreatedAt tracks when this subscription was created
	CreatedAt time.Time
}

// Next structs are only used in handleAccountDetails endpoint

// FeatureSet for an account
type FeatureSet struct {
	// Usages
	UsagesWebapp  bool `json:"usages_webapp"`
	UsagesEditor  bool `json:"usages_editor"`
	UsagesSidebar bool `json:"usages_sidebar"`
	// Call sigs
	CallSignaturesWebapp  bool `json:"call_signatures_webapp"`
	CallSignaturesEditor  bool `json:"call_signatures_editor"`
	CallSignaturesSidebar bool `json:"call_signatures_sidebar"`
	// Common invocations
	CommonInvocationsWebapp  bool `json:"common_invocations_webapp"`
	CommonInvocationsEditor  bool `json:"common_invocations_editor"`
	CommonInvocationsSidebar bool `json:"common_invocations_sidebar"`
	// examples
	ExamplesWebapp  bool `json:"examples_webapp"`
	ExamplesEditor  bool `json:"examples_editor"`
	ExamplesSidebar bool `json:"examples_sidebar"`
	// docs
	DocsWebapp  bool `json:"docs_webapp"`
	DocsEditor  bool `json:"docs_editor"`
	DocsSidebar bool `json:"docs_sidebar"`
	// completions
	CompletionsWebapp  bool `json:"completions_webapp"`
	CompletionsEditor  bool `json:"completions_editor"`
	CompletionsSidebar bool `json:"completions_sidebar"`
	// top members
	TopMembersWebapp  bool `json:"top_members_webapp"`
	TopMembersEditor  bool `json:"top_members_editor"`
	TopMembersSidebar bool `json:"top_members_sidebar"`
	// search
	SearchWebapp  bool `json:"search_webapp"`
	SearchEditor  bool `json:"search_editor"`
	SearchSidebar bool `json:"search_sidebar"`
	// outline view
	OutlineViewWebapp  bool `json:"outline_view_webapp"`
	OutlineViewEditor  bool `json:"outline_view_editor"`
	OutlineViewSidebar bool `json:"outline_view_sidebar"`
	// links
	LinksWebapp  bool `json:"links_webapp"`
	LinksEditor  bool `json:"links_editor"`
	LinksSidebar bool `json:"links_sidebar"`
}

// PlanResponse for an account, used in client.internal.kite.client.handlePlan
// to relay the user's current plan information to the client.
type PlanResponse struct {
	// Name of the plan for semantic display, values are "unknown", "pro", "community"
	Name string `json:"name"`
	// Status of the plan, e.g "trialing", "active", etc
	Status string `json:"status"`
	// ActiveSubscription is a top level indicator of whether the user
	// has any active plan that gives them access to pro features,
	// values are "unknown", "pro", "community"
	ActiveSubscription string `json:"active_subscription"`
	// Features that the user has access to via the plan
	Features FeatureSet `json:"features"`
	// TrialDaysRemaining for the user
	TrialDaysRemaining uint64 `json:"trial_days_remaining"`
	// StartedKiteProTrial is true if the user has already started their kite pro trial_days_remaining
	StartedKiteProTrial bool `json:"started_kite_pro_trial"`
	// ReferralsCredited is the number of referrals that have been credited for the user
	ReferralsCredited uint64 `json:"referrals_credited"`
	// MaxReferralCredits is the maximum number of referrals that can be credited to the user
	MaxReferralCredits uint64 `json:"max_referral_credits"`
	// ReferralDaysCredited is the number of referral days credited the user
	ReferralDaysCredited uint64 `json:"referral_days_credited"`
}

type planResponse struct {
	PlanName      string   `json:"plan_name"`
	PaymentPeriod string   `json:"payment_period"`
	Price         string   `json:"price"`
	ChargeDate    string   `json:"charge_date"`
	BillingNote   string   `json:"billing_note"`
	NumLicenses   uint64   `json:"num_licenses"`
	Cancelled     bool     `json:"cancelled"`
	Members       []member `json:"members"`
}

// card associated with an account
type card struct {
	// Last4 digits of the credit card number
	Last4 string `json:"last4"`
	// Brand of the credit card, e.g Visa
	Brand string `json:"brand"`
}

// handleAccountDetails handles showing the details for an account
func (s *Server) handleAccountDetails(w http.ResponseWriter, r *http.Request) {

	var response struct {
		CurrentPlan  *planResponse `json:"current_plan"`
		Card         *card         `json:"card"`
		ReferralCode string        `json:"referral_code"`
	}

	response.CurrentPlan = &planResponse{}

	if ed := marshalResponse("account.Server.handleDetails:", w, response); ed.HTTPError() {
		http.Error(w, ed.Msg, ed.Code)
	}
	// 200 returned automatically if no error
}
