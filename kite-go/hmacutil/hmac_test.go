package hmacutil

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/stretchr/testify/assert"
)

func TestHMACHeaders(t *testing.T) {
	user := &community.User{
		ID:    1,
		Email: "test@kite.com",
	}
	session := &community.Session{
		Key: "testing123",
	}

	// Create and check headers from user and session object
	headers := HeadersFromUserSession(user, session)
	assert.NotEmpty(t, headers.Get(tokenHeaderKey), "got empty X-Kite-Token")
	assert.NotEmpty(t, headers.Get(tokenDataHeaderKey), "got empty X-Kite-TokenData")

	// See if we can decode the token data
	buf, err := base64.StdEncoding.DecodeString(headers.Get(tokenDataHeaderKey))
	assert.NoError(t, err, "error base64 decoding token data")

	var data tokenData
	err = json.Unmarshal(buf, &data)
	assert.NoError(t, err, "error unmarshalling token data")

	// Make sure token data matches data from user and session object, with correct expiration
	assert.Equal(t, *user, data.User)
	assert.Equal(t, session.Key, data.Session)
	assert.WithinDuration(t, time.Now().Add(hmacExpiration), data.ExpiresAt, time.Second)

	// Decode token, and check to see if it matches expected token based on tokenData object
	token, err := base64.StdEncoding.DecodeString(headers.Get(tokenHeaderKey))
	assert.NoError(t, err, "error base64 decoding token")

	expected, err := hmacFromTokenData(data)
	assert.NoError(t, err, "error computing hmac from token data")
	assert.True(t, hmac.Equal(expected, token), "hmac did not match")
}

func TestHMACFromRequest(t *testing.T) {
	user := &community.User{
		ID:    1,
		Email: "test@kite.com",
	}
	session := &community.Session{
		Key: "testing123",
	}

	// Create headers from user / session object to build a request
	headers := HeadersFromUserSession(user, session)
	assert.NotEmpty(t, headers.Get(tokenHeaderKey), "got empty X-Kite-Token")
	assert.NotEmpty(t, headers.Get(tokenDataHeaderKey), "got empty X-Kite-TokenData")

	req, err := http.NewRequest("GET", "http://localhost", nil)
	assert.NoError(t, err, "error building request")

	for k, v := range headers {
		req.Header.Set(k, v[0])
	}

	// Make sure we validate this request
	_, err = CheckRequest(req)
	assert.NoError(t, err)

	// Set the token to something invalid
	req.Header.Set(tokenHeaderKey, base64.StdEncoding.EncodeToString([]byte("bad_token")))

	// Make sure request is no longer valid
	_, err = CheckRequest(req)
	assert.Equal(t, ErrInvalid, err)
}

func TestHMACExpiration(t *testing.T) {
	user := &community.User{
		ID:    1,
		Email: "test@kite.com",
	}
	session := &community.Session{
		Key: "testing123",
	}

	// Build headers from user and session object
	headers := HeadersFromUserSession(user, session)
	assert.NotEmpty(t, headers.Get(tokenHeaderKey), "got empty X-Kite-Token")
	assert.NotEmpty(t, headers.Get(tokenDataHeaderKey), "got empty X-Kite-TokenData")

	buf, err := base64.StdEncoding.DecodeString(headers.Get(tokenDataHeaderKey))
	assert.NoError(t, err, "error base64 decoding token data")

	var data tokenData
	err = json.Unmarshal(buf, &data)
	assert.NoError(t, err, "error unmarshalling token data")

	// Set expiration date to now
	data.ExpiresAt = time.Now()
	buf, err = json.Marshal(&data)
	assert.NoError(t, err, "error marshalling token data")

	// Re-encode the token data
	headers.Set(tokenDataHeaderKey, base64.StdEncoding.EncodeToString(buf))

	// Build request using new token data
	req, err := http.NewRequest("GET", "http://localhost", nil)
	assert.NoError(t, err, "error building request")

	for k, v := range headers {
		req.Header.Set(k, v[0])
	}

	// Make sure we get errHMACExpired
	_, err = CheckRequest(req)
	assert.Equal(t, err, ErrExpired)
}
