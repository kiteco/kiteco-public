package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"

	"github.com/kiteco/kiteco/kite-golib/errors"
)

// PEM encoded rsa public key
const publicKey = `-----BEGIN RSA PUBLIC KEY-----
XXXXXXX
-----END RSA PUBLIC KEY-----`

// validateSignature validates the file at filePath against the provided signature
func validateSignature(filePath, publicKey string, signature []byte) error {
	keyBytes := []byte(publicKey)
	block, rest := pem.Decode(keyBytes)
	if block == nil || block.Type != "RSA PUBLIC KEY" || len(rest) != 0 {
		return errors.Errorf("error reading public key")
	}

	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return err
	}

	input, err := os.Open(filePath)
	if err != nil {
		return err
	}

	hashFunc := crypto.SHA256
	hash := hashFunc.New()
	if _, err = io.Copy(hash, input); err != nil {
		return err
	}

	if err = rsa.VerifyPSS(pub, hashFunc, hash.Sum(nil), signature, nil); err != nil {
		return errors.Errorf("error validating %s: %s", filePath, err.Error())
	}
	return nil
}
