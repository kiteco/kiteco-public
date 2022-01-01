package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/kiteco/kiteco/kite-golib/licensing"
)

func main() {
	log.SetPrefix("")
	log.SetFlags(0)
	if len(os.Args) < 2 {
		empty, _ := json.MarshalIndent(licensing.Claims{}, "\t", "  ")
		log.Fatalf(`Usage:
	licenseutil create [private.key] <claims.json >license.txt
	licenseutil validate [public.key] <license.txt >claims.json
Claims:
	%s
`, empty)
	}

	cmd := os.Args[1]
	switch cmd {
	case "create":
		cmdSign(os.Args[2:])
	case "validate":
		cmdParse(os.Args[2:])
	}
}

func cmdParse(args []string) {
	var rsaKey *rsa.PublicKey
	if len(args) > 0 {
		keyBytes, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.Fatalln(err)
		}

		rsaKey, err = x509.ParsePKCS1PublicKey(keyBytes)
		if err != nil {
			log.Fatalln(err)
		}
	}

	tokenBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalln(err)
	}

	validator := licensing.NewValidatorWithKey(rsaKey)
	lic, err := validator.Parse(string(tokenBytes))
	if err != nil {
		log.Fatalln("license is invalid", err)
	}

	log.Printf("license is valid")
	b, err := json.MarshalIndent(lic.Claims, "", "  ")
	if err != nil {
		log.Fatalln("could not marshal claims", err)
	}

	fmt.Println(string(b))
}

func cmdSign(args []string) {
	var mgr *licensing.Authority
	var err error
	if len(args) > 0 {
		keyBytes, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.Fatalln(err)
		}

		rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
		if err != nil {
			log.Fatalln(err)
		}

		mgr, err = licensing.NewAuthorityWithKey(rsaKey)
	} else {
		log.Println("no key file provided: using test authority")
		mgr, err = licensing.NewTestAuthority()
	}
	if err != nil {
		log.Fatalln(err)
	}

	var claims licensing.Claims
	if err := json.NewDecoder(os.Stdin).Decode(&claims); err != nil {
		log.Fatalln("could not decode license claims", err)
	}

	lic, err := mgr.CreateLicense(claims)
	if err != nil {
		log.Fatalln("failed to create license", err)
	}

	log.Println("created license")
	b, err := json.MarshalIndent(lic.Claims, "", "  ")
	if err != nil {
		log.Fatalln("could not marshal claims", err)
	}
	log.Println(string(b))

	fmt.Println(lic.Token)
}
