package component

import (
	"net/http"

	"github.com/kiteco/kiteco/kite-go/client/internal/conversion/listener"
	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/remotectrl"
)

// CohortManager ...
type CohortManager interface {
	ConversionCohortGetter

	FeatureEnabledWrapper

	// Cohorts ...
	Cohorts() Cohorts

	// SetSetupCompleted provides special logic for settings manager
	SetSetupCompleted(newvalue bool) error

	// RegisterOnConversionCohortChanged registers a callback to be called after the ConversionCohort changes.
	RegisterOnConversionCohortChanged(func(old, new string))

	// CohortManager can recieve RC messages
	remotectrl.Handler
}

// FeatureEnabledWrapper ...
type FeatureEnabledWrapper interface {
	// WrapFeatureEnabled checks whether the features are enabled based on the license, cohort, and the rc config all_features_pro
	WrapFeatureEnabled(http.HandlerFunc) http.HandlerFunc
}

// ConversionCohortGetter provides a method to get the conversion cohort
type ConversionCohortGetter interface {
	// ConversionCohort returns what pro-conversion experience a user should get.
	ConversionCohort() string
}

// Cohorts provides logic for various conversion experiences
type Cohorts interface {
	licensing.ProductGetter
	OnComplSelecters() []listener.OnComplSelecter
	AugmentRenderOpts(opts data.RenderOptions) (ro data.RenderOptions, lexPyDisabled bool)
}

// MockCohortManager only implements specific methods needed for current tests
type MockCohortManager struct {
	Convcohort string
}

// ConversionCohort implements ConversionCohortGetter
func (m MockCohortManager) ConversionCohort() string {
	return m.Convcohort
}

// WrapFeatureEnabled implements FeatureEnabledWrapper
func (m MockCohortManager) WrapFeatureEnabled(nochange http.HandlerFunc) http.HandlerFunc {
	return nochange
}
