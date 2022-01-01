package octobat

// MetadataKeyTrialDuration is the key used in BeanieKiteMetadata
const MetadataKeyTrialDuration = "kite_trial_duration"

// BeanieMode defines the available modes for the client/server session setup
type BeanieMode string

// ModePayment seems to be the required value for session setup. The modes are undocumented by Octobat
const ModePayment = "payment"

// ModeSetup is undocumented by Octobat
const ModeSetup = "setup"

// ModeSubscription is undocumented by Octobat
const ModeSubscription = "subscription"

// TaxMode defines the tax mode supported by Octobat
type TaxMode string

// TaxModeInclusive defines tax-inclusive billing
const TaxModeInclusive = "inclusive"

// TaxModeExclusive defines tax-exclusive billing
const TaxModeExclusive = "exclusive"

// BeaniePrefillData defines the data, which could be bassed to the Octobat Beanie form
type BeaniePrefillData struct {
	CustomerName  string `json:"customer_name,omitempty"`
	CustomerEmail string `json:"customer_email,omitempty"`
}

// BeanieKiteMetadata defines metadata, which is used by Kite and passed on to Octobat
type BeanieKiteMetadata struct {
	KiteUserID        string `json:"kite_user_id,omitempty"`
	IPAddress         string `json:"ip_address,omitempty"`
	KiteTrialDuration int64  `json:"kite_trial_duration,omitempty"` //seconds
}

// BeanieItem is a single line item
type BeanieItem struct {
	StripePlanID string `json:"plan"`                   // from the Stripe dashboard
	Quantity     int    `json:"quantity,omitempty"`     // Optional
	ProductType  string `json:"product_type,omitempty"` // Optional
}

// SubscriptionData defines the subscription data posted to the server
type SubscriptionData struct {
	Items           []BeanieItem `json:"subscription_items"`
	TrialEnd        int64        `json:"trial_end,omitempty"`         // unix timestamp to set the end of the trial
	TrialPeriodDays int          `json:"trial_period_days,omitempty"` // alternative to TrialEnd to define the trial length in days
}

// BeanieServerlessSession is a subset of the JSON request shown at https://octobat-share.s3.eu-west-3.amazonaws.com/OctobatTmpBeanieDoc.pdf
// The naming of the JSON properties is slightly different to BeanieServerSession
type BeanieServerlessSession struct {
	ConfigurationID   string             `json:"beanie_configuration,omitempty"` // The ID of the customer you want to attach this session to. Mandatory for setup mode.
	Customer          string             `json:"customer,omitempty"`
	SuccessURL        string             `json:"successUrl,omitempty"`
	CancelURL         string             `json:"cancelUrl,omitempty"`
	TaxCalculation    TaxMode            `json:"tax_calculation,omitempty"`
	ClientReferenceID string             `json:"client_reference_id"`
	PrefillData       BeaniePrefillData  `json:"prefill_data"`
	Metadata          BeanieKiteMetadata `json:"metadata"`
	Items             []BeanieItem       `json:"items,omitempty"`
	PrimaryColor      string             `json:"primaryColor,omitempty"`
}

// BeanieServerSession is a subset of the JSON request shown at https://octobat-share.s3.eu-west-3.amazonaws.com/OctobatTmpBeanieDoc.pdf
type BeanieServerSession struct {
	Mode              BeanieMode          `json:"mode,omitempty"`
	ConfigurationID   string              `json:"beanie_configuration,omitempty"` // The ID of the customer you want to attach this session to. Mandatory for setup mode.
	Customer          string              `json:"customer,omitempty"`
	SuccessURL        string              `json:"success_url,omitempty"`
	CancelURL         string              `json:"cancel_url,omitempty"`
	TaxCalculation    TaxMode             `json:"tax_calculation,omitempty"`
	ClientReferenceID string              `json:"client_reference_id"`
	PrefillData       *BeaniePrefillData  `json:"prefill_data,omitempty"`
	Metadata          *BeanieKiteMetadata `json:"metadata,omitempty"`
	PrimaryColor      string              `json:"primary_color,omitempty"`

	// Use one of LineItems or SubscriptionData
	LineItems        []BeanieItem      `json:"line_items,omitempty"`
	SubscriptionData *SubscriptionData `json:"subscription_data,omitempty"`
}

// BeanieSessionResponse is a subset of the JSON response shown in https://octobat-share.s3.eu-west-3.amazonaws.com/OctobatTmpBeanieDoc.pdf
type BeanieSessionResponse struct {
	ID string `json:"id"`
}
