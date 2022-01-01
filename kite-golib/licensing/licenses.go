package licensing

import (
	"crypto/rsa"
	"encoding/json"
)

// Licenses encapsulates a set of licenses, typically for a single user.
// However any license / user validation is the client's responsibility.
type Licenses struct {
	licenses []*License
	// index of the most preferable license
	bestIdx int
}

// NewLicenses ...
func NewLicenses() *Licenses {
	return &Licenses{}
}

// Add adds a license
func (l *Licenses) Add(lic *License) {
	if len(l.licenses) > 0 && lic.IsPreferableTo(l.licenses[l.bestIdx]) {
		l.bestIdx = len(l.licenses)
	}
	l.licenses = append(l.licenses, lic)
}

// Len returns the number of licenses
func (l *Licenses) Len() int {
	return len(l.licenses)
}

// Iterate returns an iterator pair:
// for lic, next := l.Iterate(); lic != nil; lic = next() { }
func (l *Licenses) Iterate() (*License, func() *License) {
	i := 0
	next := func() *License {
		if i >= len(l.licenses) {
			return nil
		}

		lic := l.licenses[i]
		i++
		return lic
	}

	return next(), next
}

// License returns the most preferable license to use.
// It will be a current (not expired) license.
func (l *Licenses) License() *License {
	if len(l.licenses) == 0 {
		return nil
	}
	lic := l.licenses[l.bestIdx]
	if lic.IsExpired() {
		return nil
	}
	return lic
}

// MarshalJSON implements json.Marshaler
func (l *Licenses) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.licenses)
}

// UnmarshalJSON implements json.Unmarshaler
func (l *Licenses) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &(l.licenses)); err != nil {
		return err
	}

	l.bestIdx = 0
	for i, lic := range l.licenses {
		if lic.IsPreferableTo(l.licenses[l.bestIdx]) {
			l.bestIdx = i
		}
	}

	return nil
}

// - helpers

// AddToken parses and adds a license from a JWT token.
// The parsed License is returned, or a parse error.
// If a key is not provided, no signature validation is done.
func (l *Licenses) AddToken(licenseToken string, key *rsa.PublicKey) (*License, error) {
	lic, err := ParseLicense(licenseToken, key)
	if err == nil {
		l.Add(lic)
	}
	return lic, err
}

// TrialAvailable returns the trial state for the current user by checking all licenses in this store
func (l *Licenses) TrialAvailable() bool {
	for _, license := range l.licenses {
		if license.Product == Pro {
			return false
		}
	}
	return true
}
