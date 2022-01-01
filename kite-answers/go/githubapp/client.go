package githubapp

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/shurcooL/githubv4"
)

// Credentials encapsulates a GitHub App's credentials
type Credentials struct {
	ID         string
	PrivateKey *rsa.PrivateKey
}

// ParseCredentials creates a Credentials by parsing the given PEM private key
func ParseCredentials(appID string, keyPEM []byte) (Credentials, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return Credentials{}, errors.Errorf("could not decode PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return Credentials{}, err
	}

	return Credentials{
		ID:         appID,
		PrivateKey: key,
	}, nil
}

// NewInstallClient creates a new github.com/shurcooL/githubv4.Client for a given GitHub App installation
func NewInstallClient(creds Credentials, installID string, base http.RoundTripper) *githubv4.Client {
	transport := newInstallRoundTripper(creds, installID, base)
	client := http.Client{Transport: transport}
	return githubv4.NewClient(&client)
}
