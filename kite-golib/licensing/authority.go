package licensing

import (
	"crypto/rand"
	"crypto/rsa"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

// Authority creates licenses. It needs a private key to sign the new licenses.
type Authority struct {
	privateKey *rsa.PrivateKey
}

// NewAuthorityFromPEMString returns a new license manager which uses the PEM key provided in the string arg
func NewAuthorityFromPEMString(keyContent string) (*Authority, error) {
	if keyContent == "" {
		return nil, errors.New("Please provide the RSA key to use to sign the licenses (env var LICENSE_RSA_KEY)")
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(keyContent))
	if err != nil {
		return nil, err
	}
	return NewAuthorityWithKey(privateKey)
}

// NewAuthorityWithKey returns a new license manager, which uses an existing private key
func NewAuthorityWithKey(key *rsa.PrivateKey) (*Authority, error) {
	return &Authority{
		privateKey: key,
	}, nil
}

// NewTestAuthority is a convenience method to create a new manager with a randomly generated rsa key
// this is only suitable for testing and must not be used in production code
func NewTestAuthority() (*Authority, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}

	return NewAuthorityWithKey(rsaKey)
}

// CreateValidator is a convenience method to create a matching validator
// Use NewValidator() if the private key is unavailable.
func (m *Authority) CreateValidator() *Validator {
	return NewValidatorWithKey(&m.privateKey.PublicKey)
}

// CreateLicense creates a new license and signs it with the configured private key
// It returns the license, the jwt string, and an error.
func (m *Authority) CreateLicense(claims Claims) (*License, error) {
	// for date and time values see "NumericDate" in https://tools.ietf.org/html/rfc7519#section-2
	if claims.IssuedAt.IsZero() {
		claims.IssuedAt = time.Now()
	}

	claims.Version = "1.0"
	if claims.ExpiresAt.IsZero() {
		claims.ExpiresAt = claims.PlanEnd
	}
	if claims.PlanEnd.IsZero() {
		claims.PlanEnd = claims.ExpiresAt
	}

	if err := claims.Valid(); err != nil {
		return nil, err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	tokenString, err := token.SignedString(m.privateKey)
	if err != nil {
		return nil, err
	}

	return &License{Claims: claims, Token: tokenString}, nil
}
