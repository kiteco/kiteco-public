package auth

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cleanup(client *Client) error {
	err := os.Remove(client.userCacheFile)
	return err
}

func Test_TokenUpdate(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{"user@example.com": "secret"})
	require.NoError(t, err)
	defer s.Close()

	SetupWithAuthDefaults(s)

	s.Backend.AddPrefixRequestHandler("/api/hmac1", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Kite-Token", "kite-token-value")
		w.Header().Set("Kite-TokenData", "kite-token-data-value")
		w.WriteHeader(http.StatusOK)
	})

	s.Backend.AddPrefixRequestHandler("/api/hmac2", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Kite-Token", "kite-token-value2")
		w.Header().Set("Kite-TokenData", "kite-token-data-value2")
		w.WriteHeader(http.StatusOK)
	})

	resp, err := s.AuthClient.Get(context.Background(), "/api/hmac1")
	assert.NoError(t, err)
	resp.Body.Close()

	// Make sure that the token data has been updated
	req, err := s.AuthClient.NewRequest("GET", "/api/test", "text/plain", nil)
	assert.NoError(t, err)
	assert.Equal(t, "kite-token-value", req.Header.Get("Kite-Token"))
	assert.Equal(t, "kite-token-data-value", req.Header.Get("Kite-TokenData"))

	resp, err = s.AuthClient.Get(context.Background(), "/api/hmac2")
	assert.NoError(t, err)
	resp.Body.Close()

	// Make sure that the token data has been updated (again)
	req, err = s.AuthClient.NewRequest("GET", "/api/test", "text/plain", nil)
	assert.NoError(t, err)
	assert.Equal(t, "kite-token-value2", req.Header.Get("Kite-Token"))
	assert.Equal(t, "kite-token-data-value2", req.Header.Get("Kite-TokenData"))
}

func Test_PostForm(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{})
	assert.NoError(t, err)
	defer s.Close()

	SetupWithAuthDefaults(s)

	s.Backend.AddPrefixRequestHandler("/client-request", []string{"POST"}, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	resp, err := s.AuthClient.(*Client).postForm(context.Background(), "/client-request-post", map[string][]string{"first": {"a", "b"}})
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
}

func Test_ServerHTTPCookies(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{})
	assert.NoError(t, err)
	defer s.Close()

	auth := NewTestClient(defaultHTTPTimeoutTest)
	defer cleanup(auth)
	s.SetupWithCustomAuthClient(auth)
	auth.SetTarget(s.Backend.URL)

	s.Backend.AddPrefixRequestHandler("/cookie-request", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("kite-cookie"); err == nil && c.Value == "kite-cookie-value" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	auth.httpClient().Jar.SetCookies(auth.Target(), []*http.Cookie{{Name: "kite-cookie", Value: "kite-cookie-value"}})

	r, err := http.NewRequest("GET", "/cookie-request", nil)
	assert.NoError(t, err)

	resp := httptest.NewRecorder()
	auth.ServeHTTP(resp, r)
	assert.NoError(t, err)

	assert.EqualValues(t, http.StatusOK, resp.Code)
}

func Test_RegionResponseHeader(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{})
	assert.NoError(t, err)
	defer s.Close()

	SetupWithAuthDefaults(s)

	s.Backend.AddPrefixRequestHandler("/region-eu", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Kite-Region", "EU")
	})
	s.Backend.AddPrefixRequestHandler("/region-us", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Kite-Region", "US")
	})
	s.Backend.AddPrefixRequestHandler("/metrics-disabled", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Kite-Disabled", "any-value")
	})

	assert.EqualValues(t, "", s.Metrics.GetRegion())

	s.AuthClient.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/region-eu", nil))
	assert.EqualValues(t, "EU", s.Metrics.GetRegion())

	s.AuthClient.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/region-us", nil))
	assert.EqualValues(t, "US", s.Metrics.GetRegion())
}

func Test_DoRequest(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{})
	assert.NoError(t, err)
	defer s.Close()

	auth := NewTestClient(defaultHTTPTimeoutTest)
	defer cleanup(auth)
	s.SetupWithCustomAuthClient(auth)
	auth.SetTarget(s.Backend.URL)

	s.Backend.AddPrefixRequestHandler("/client-request", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		machine := r.Header.Get("Kite-Machine")
		token := r.Header.Get("Kite-Token")
		tokenData := r.Header.Get("Kite-Tokendata")
		if machine != "" || token == "" || tokenData == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response-data"))
	})

	r, err := http.NewRequest("GET", "/client-request", nil)
	r.URL, _ = s.Backend.URL.Parse("/client-request") // update the URL to use the right scheme, host, and post
	assert.NoError(t, err)

	// Do() must send the headers Kite-Token and Kite-TokenData
	// It must not send the Kite-Machine header (this is how the initial implementation was)
	headers := map[string][]string{"Kite-Token": {"token-value"}, "Kite-Tokendata": {"token-data-value"}}
	auth.token.UpdateFromHeader(headers)
	resp, err := auth.Do(context.Background(), r)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.EqualValues(t, "response-data", string(body))
}

func Test_UpdateTarget(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{})
	assert.NoError(t, err)
	defer s.Close()
	s.Backend.AddPrefixRequestHandler("/client-request", []string{"GET"}, func(w http.ResponseWriter, request *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	backend2, err := mockserver.NewBackend(map[string]string{"user@example.com": "secret"})
	assert.NoError(t, err)
	defer backend2.Close()
	backend2.AddPrefixRequestHandler("/client-request", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	SetupWithAuthDefaults(s)

	assert.EqualValues(t, s.Backend.URL, s.AuthClient.Target())
	resp, err := s.AuthClient.Get(context.Background(), "/client-request")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.EqualValues(t, http.StatusInternalServerError, resp.StatusCode, "1st backend must be used")

	s.AuthClient.SetTarget(backend2.URL)
	assert.EqualValues(t, backend2.URL, s.AuthClient.Target())
	resp, err = s.AuthClient.Get(context.Background(), "/client-request")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.EqualValues(t, http.StatusOK, resp.StatusCode, "2nd backend must be used")

	rec := httptest.NewRecorder()
	s.AuthClient.ServeHTTP(rec, httptest.NewRequest("GET", "/client-request", nil))
	assert.EqualValues(t, http.StatusOK, rec.Result().StatusCode, "serverHTTP must use the new backend")
}

func Test_StripRequestCookies(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{})
	assert.NoError(t, err)
	defer s.Close()

	auth := NewTestClient(defaultHTTPTimeoutTest)
	defer cleanup(auth)
	s.SetupWithCustomAuthClient(auth)
	auth.SetTarget(s.Backend.URL)

	var requestCookies int32
	s.Backend.AddPrefixRequestHandler("/client-request", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCookies, int32(len(r.Cookies())))
	})

	r, err := auth.NewRequest("GET", "/client-request", "application/json", nil)
	r.AddCookie(&http.Cookie{Name: "custom-cookie", Value: "cookie-value"})
	assert.NoError(t, err)

	auth.ServeHTTP(httptest.NewRecorder(), r)
	assert.EqualValues(t, 0, requestCookies)
}

func Test_RequestTimeout(t *testing.T) {
	s, err := mockserver.NewTestClientServer(map[string]string{})
	assert.NoError(t, err)
	defer s.Close()

	SetupWithAuthDefaults(s)

	s.Backend.AddPrefixRequestHandler("/client-request", []string{"GET", "POST"}, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(400 * time.Millisecond) // longer than the timeout below
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	resp, err := s.AuthClient.Get(ctx, "/client-request")
	assertContextDeadlineExceeded(t, err)
	if resp != nil {
		defer resp.Body.Close()
	}

	ctx, cancel = context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	resp, err = s.AuthClient.Post(ctx, "/client-request", "application/json", strings.NewReader(""))
	assertContextDeadlineExceeded(t, err)
	if resp != nil {
		resp.Body.Close()
	}
}

func Test_ResponseRecorder(t *testing.T) {
	r := responseWriterRecorder{ResponseWriter: httptest.NewRecorder()}
	r.Write([]byte{1, 2})
	assert.EqualValues(t, http.StatusOK, r.status)
	assert.EqualValues(t, 2, r.bytes)

	r = responseWriterRecorder{ResponseWriter: httptest.NewRecorder()}
	r.WriteHeader(http.StatusInternalServerError)
	r.Write([]byte{1, 2, 3})
	assert.EqualValues(t, http.StatusInternalServerError, r.status)
	assert.EqualValues(t, 3, r.bytes)
}

func assertContextDeadlineExceeded(t *testing.T, err error) {
	if err == context.DeadlineExceeded {
		return
	}
	switch e := err.(type) {
	case *url.Error:
		if e.Err == context.DeadlineExceeded {
			return
		}
	}

	assert.Fail(t, "error was not deadline exceeded: %s", err)
}
