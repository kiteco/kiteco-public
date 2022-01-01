package client

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

const licensePublicKey = `-----BEGIN PUBLIC KEY-----
XXXXXXX
-----END PUBLIC KEY-----`

func readPublicKey() (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(licensePublicKey))
	if block == nil {
		return nil, errors.Errorf("unable to parse license key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.Errorf("unsupported key type")
	}
	return rsaKey, nil
}

// NewLicenseValidator creates a new licensing.Validator with the baked-in key
func NewLicenseValidator() (*licensing.Validator, error) {
	publicKey, err := readPublicKey()
	if err != nil {
		return nil, err
	}
	return licensing.NewValidatorWithKey(publicKey), nil
}
