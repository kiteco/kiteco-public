package community

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
)

func cryptoBytes(byteCount uint32) []byte {
	key := make([]byte, byteCount)
	_, err := rand.Read(key)
	if err != nil {
		log.Fatal(err)
	}
	return key
}

// Generates n random bytes and base64-encodes them.  Used for things
// like session ids and email verification codes.
func randomBytesBase64(n uint32) string {
	return base64.StdEncoding.EncodeToString(cryptoBytes(n))
}

// Generates n random bytes and hex-encodes them.  Used for things
// like session ids and email verification codes.
func randomBytesHex(n uint32) string {
	return fmt.Sprintf("%x", cryptoBytes(n))
}
