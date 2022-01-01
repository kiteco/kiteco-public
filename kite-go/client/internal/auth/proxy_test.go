package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/permissions"
	"github.com/kiteco/kiteco/kite-go/client/internal/metrics"
	"github.com/kiteco/kiteco/kite-go/client/internal/mockserver"
	"github.com/kiteco/kiteco/kite-go/client/internal/settings"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/community/account"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_FetchUser(t *testing.T) {
	s, client, err := newTestServerLocal()
	assert.NoError(t, err)
	defer s.Close()

	_, err = client.FetchUser(context.Background())
	assert.Error(t, err, "Expected error if there's no auth cookie set")

	//let the backend return an auth cookie, the client only extracts cookies when ServerHTTP is called
	s.Backend.AddPrefixRequestHandler("/cookie-request", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Kite-Token", "kite-token-value")
		w.Header().Set("Kite-TokenData", "kite-token-data-value")
		http.SetCookie(w, &http.Cookie{Name: sessionKey, Value: "kite-cookie-value"})
		w.WriteHeader(200)
	})
	r, err := http.NewRequest("GET", "/cookie-request", nil)
	assert.NoError(t, err)
	client.ServeHTTP(httptest.NewRecorder(), r)

	// now try again FetchSessionedUser with an auth cookie set
	s.Backend.AddPrefixRequestHandler("/api/account/user", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		body, err := json.Marshal(community.User{Email: "user@example.com"})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(body)
	})
	s.Backend.AddPrefixRequestHandler("/api/account/plan", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		body, err := json.Marshal(account.PlanResponse{Name: "pro", Status: "active"})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(body)
	})
	user, err := client.FetchUser(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.EqualValues(t, "user@example.com", user.Email, "Expected the user returned by the backend")

	//check for error after the backend returned "unauthorized"
	s.Backend.AddPrefixRequestHandler("/api/account/user", []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	user, err = client.FetchUser(context.Background())
	assert.EqualError(t, err, ErrNotAuthenticated.Error(), "Expected ErrNotAuthenticated when the backend reported an unauthorized user")
	assert.Nil(t, user)
}

func Test_NilClientHttpMethods(t *testing.T) {
	s, authClient, err := newTestServerLocal()
	require.NoError(t, err)
	defer s.Close()

	c := authClient.(*Client)
	s.Backend.AddPrefixRequestHandler("/api/dummy", []string{"GET", "POST"}, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"value": "my-value"}`))
	})

	ctx := context.Background()

	var stop int64
	defer func() {
		atomic.StoreInt64(&stop, 1)
	}()
	go func() {
		for {
			if atomic.LoadInt64(&stop) == 1 {
				break
			}
			c.SetTarget(s.Backend.URL)
		}
	}()

	type dummyResponse struct {
		Value string `json:"value"`
	}

	for i := 0; i < 200; i++ {
		resp, err := c.Get(ctx, "/api/dummy")
		require.NoError(t, err)
		resp.Body.Close()
		require.EqualValues(t, http.StatusOK, resp.StatusCode)

		resp, err = c.getNoHMAC(ctx, "/api/dummy")
		require.NoError(t, err)
		resp.Body.Close()
		require.EqualValues(t, http.StatusOK, resp.StatusCode)

		resp, err = c.Post(ctx, "/api/dummy", "text/plain", strings.NewReader(""))
		require.NoError(t, err)
		resp.Body.Close()
		require.EqualValues(t, http.StatusOK, resp.StatusCode)

		r, err := c.NewRequest("GET", "/api/dummy", "text/plain", nil)
		require.NoError(t, err)
		resp, err = c.Do(ctx, r)
		require.NoError(t, err)
		resp.Body.Close()
		require.EqualValues(t, http.StatusOK, resp.StatusCode)
	}
}

// newTestServer returns a new test server with kite local enabled by the kiteLocal parameter value
func newTestServerLocal() (*mockserver.TestClientServer, component.AuthClient, error) {
	clientServer, err := mockserver.NewTestClientServerFeatures(map[string]string{"user1@example.com": "password1"}, nil)
	if err != nil {
		return nil, nil, err
	}

	auth := NewTestClient(300 * time.Millisecond)
	defer os.Remove(auth.userCacheFile)
	settingsMgr := settings.NewTestManager()
	permManager := permissions.NewTestManager(lang.Python)

	err = clientServer.SetupComponents(auth, settingsMgr, permManager, metrics.NewMockManager())
	if err != nil {
		return nil, nil, err
	}

	auth.SetTarget(clientServer.Backend.URL)

	return clientServer, auth, nil
}
