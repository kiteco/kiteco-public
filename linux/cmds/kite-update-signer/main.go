package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		path, _ := os.Executable()
		log.Fatalf("Usage: %s data.file private_key_string", path)
	}

	dataFile := os.Args[1]
	keyData := os.Args[2]

	bytes := []byte(keyData)
	block, rest := pem.Decode(bytes)
	if block == nil || block.Type != "RSA PRIVATE KEY" || len(rest) != 0 {
		log.Fatalf("error reading private key: %v, %v", rest, block)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatal(err.Error())
	}

	dataInput, err := os.Open(dataFile)
	if err != nil {
		log.Fatalf(err.Error())
	}

	hashFunc := crypto.SHA256
	hash := hashFunc.New()
	_, err = io.Copy(hash, dataInput)
	if err != nil {
		log.Fatalf(err.Error())
	}

	signature, err := rsa.SignPSS(rand.Reader, privateKey, hashFunc, hash.Sum(nil), nil)
	if err != nil {
		log.Fatalf(err.Error())
	}

	fmt.Println(base64.StdEncoding.EncodeToString(signature))
}
