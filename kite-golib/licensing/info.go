package licensing

import (
	"time"
)

const day = 24 * time.Hour

// LicenseInfo as consumed by UI
type LicenseInfo struct {
	// Product is Kite Pro or Kite Free
	Product Product `json:"product"`

	// Plan is trial, education, temp, etc.
	Plan Plan `json:"plan,omitempty"`

	// DaysRemaining is time to plan end rounded *up*. It only applies to non-free licenses.
	DaysRemaining int `json:"days_remaining,omitempty"`

	// TrialAvailable indicates if a Kite Pro trial is available.
	TrialAvailable bool `json:"trial_available"`
}

// LicenseInfo returns info about the active plan.
// If there is more than one active plan, it uses the default precedence rules.
func (l *Licenses) LicenseInfo() LicenseInfo {
	latest := l.License()
	info := LicenseInfo{
		Product:        latest.GetProduct(),
		Plan:           latest.GetPlan(),
		TrialAvailable: l.TrialAvailable(),
	}

	if latest != nil {
		timeLeft := latest.PlanEnd.Sub(time.Now())
		// ceiling integer division
		info.DaysRemaining = int(1 + (timeLeft-1)/day)
	}

	return info
}

// LicenseInfo ...
func (s *Store) LicenseInfo() LicenseInfo {
	if s.KiteServer {
		return LicenseInfo{Product: Pro, Plan: ProServer}
	}
	info := s.l.LicenseInfo()
	info.TrialAvailable = s.TrialAvailable()
	return info
}
