package community

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kiteco/kiteco/kite-golib/envutil"
)

var (
	hmacAuthKey    = []byte(envutil.GetenvDefault("COMMUNITY_HMAC_KEY", "insecure_key"))
	hmacExpiration = 24 * time.Hour
)

const (
	tokenHeaderKey     = "Kite-Token"
	tokenDataHeaderKey = "Kite-TokenData"
)

var (
	errHMACInvalid = errors.New("invalid hmac token")
	errHMACExpired = errors.New("expired hmac token")
)

type tokenData struct {
	User      User
	Session   string
	ExpiresAt time.Time
}

func hmacFromTokenData(data tokenData) ([]byte, error) {
	msg, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}
	return createHMAC(msg, hmacAuthKey), nil
}

// HmacHeadersFromUserSession is exported for testing external packages.
func HmacHeadersFromUserSession(user *User, session *Session) http.Header {
	var data tokenData
	data.User = *user
	data.Session = session.Key
	data.ExpiresAt = time.Now().Add(hmacExpiration)
	buf, err := json.Marshal(&data)
	if err != nil {
		log.Println("could not marshal token data:", err)
		return nil
	}

	token, err := hmacFromTokenData(data)
	if err != nil {
		log.Println("could not create token:", err)
		return nil
	}

	header := make(http.Header)
	header.Add(tokenDataHeaderKey, base64.StdEncoding.EncodeToString(buf))
	header.Add(tokenHeaderKey, base64.StdEncoding.EncodeToString(token))
	return header
}

func checkHMACFromRequest(r *http.Request) (*User, error) {
	buf, err := base64.StdEncoding.DecodeString(r.Header.Get(tokenDataHeaderKey))
	if err != nil {
		log.Println("could not decode token data:", err)
		return nil, err
	}

	if len(buf) == 0 {
		log.Println("empty token data")
		return nil, fmt.Errorf("empty token data")
	}

	var data tokenData
	err = json.Unmarshal(buf, &data)
	if err != nil {
		log.Println("could not unmarshal token data:", string(buf), err)
		return nil, err
	}

	if data.ExpiresAt.Before(time.Now()) {
		log.Printf("hmac token for user %d (%s) expired", data.User.ID, data.User.Email)
		return nil, errHMACExpired
	}

	token, err := base64.StdEncoding.DecodeString(r.Header.Get(tokenHeaderKey))
	if err != nil {
		log.Println("could not decode token:", err)
		return nil, err
	}

	expectedMAC, err := hmacFromTokenData(data)
	if err != nil {
		log.Println("error creating expected token:", err)
		return nil, err
	}

	if !hmac.Equal(token, expectedMAC) {
		return nil, errHMACInvalid
	}

	return &data.User, nil
}

func clearHMAC(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(tokenDataHeaderKey, "invalid")
	w.Header().Set(tokenHeaderKey, "invalid")
}

// --

func createHMAC(message, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}
