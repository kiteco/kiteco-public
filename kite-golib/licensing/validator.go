package licensing

import (
	"crypto/rand"
	"crypto/rsa"
)

// Validator validates licenses, it needs a public key.
type Validator struct {
	publicKey *rsa.PublicKey
}

// NewValidatorWithKey returns a new validator, which uses an existing public key
func NewValidatorWithKey(publicKey *rsa.PublicKey) *Validator {
	return &Validator{
		publicKey: publicKey,
	}
}

// newTestValidator creates a new validator with a randomly generated rsa key
func newTestValidator() (*Validator, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	return NewValidatorWithKey(&rsaKey.PublicKey), nil
}

// Parse parses a JWT token string into a License.
// If the JWT token is invalid, an error is returned.
// If the Validator is nil, no validation is performed, and the License is simply parsed.
func (v *Validator) Parse(tokenString string) (*License, error) {
	var publicKey *rsa.PublicKey
	if v != nil {
		publicKey = v.publicKey
	}

	lic, err := ParseLicense(tokenString, publicKey)
	if err != nil {
		return nil, err
	}

	return lic, nil
}
