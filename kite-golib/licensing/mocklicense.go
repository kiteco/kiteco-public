package licensing

import "time"

// MockLicense implements various licensing interfaces for testing
type MockLicense struct {
	Product    Product
	Plan       Plan
	TrialAvail bool
}

// GetProduct implements ProductGetter
func (l *MockLicense) GetProduct() Product {
	return l.Product
}

// LicenseStatus implements StatusGetter
func (l *MockLicense) LicenseStatus() (time.Time, time.Time, Plan, Product) {
	return time.Time{}, time.Time{}, l.Plan, l.Product
}

// TrialAvailable implements TrialAvailableGetter
func (l *MockLicense) TrialAvailable() bool {
	return l.TrialAvail
}
