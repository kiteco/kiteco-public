package conversion

// Conversion cohorts possible
const (
	Autostart      = "autostart"
	OptIn          = "opt-in"
	QuietAutostart = "quiet-autostart"
	UsagePaywall   = "usage-paywall"
	// Allows resetting during testing
	NoCohort = ""
)

// CohortSet ...
var CohortSet = map[string]struct{}{
	Autostart:      {},
	OptIn:          {},
	QuietAutostart: {},
	UsagePaywall:   {},
	NoCohort:       {},
}

// NotificationIDs
var (
	AutostartTrial          = "autostart_trial"
	CompletionsCTAPostTrial = "completions_cta_post_trial"
	ProLaunch               = "pro_launch"
	UsagePaywallExhausted   = "usage_paywall_exhausted"
)

// NotifIDSet ...
var NotifIDSet = map[string]struct{}{
	AutostartTrial:          {},
	CompletionsCTAPostTrial: {},
	ProLaunch:               {},
	UsagePaywallExhausted:   {},
}
