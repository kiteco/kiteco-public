package community

import (
	"flag"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var logOut bool

func init() {
	flag.BoolVar(&logOut, "logout", false, "turn on logging")
}

func makeTestApp() *App {
	db := DB("postgres", "postgres://communityuser:kite@localhost/community_test?sslmode=disable")
	db.LogMode(logOut)
	db.DropTableIfExists(&User{})
	db.DropTableIfExists(&Session{})
	db.DropTableIfExists(&PasswordReset{})
	db.DropTableIfExists(&EmailVerification{})
	db.DropTableIfExists(&Signup{})
	db.DropTableIfExists(&Download{})
	db.DropTableIfExists(&Nonce{})
	app := NewApp(db, NewSettingsManager(), nil)
	err := app.Migrate()
	if err != nil {
		log.Fatal(err)
	}

	emailVerifier := &mockEmailVerifier{}
	app.EmailVerifier = emailVerifier

	return app
}

func requireCleanupApp(t *testing.T, app *App) {
	require.NoError(t, app.DB.Close())
}

func makeTestURL(base, endpoint string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		log.Fatal(err)
	}

	endpointURL, err := baseURL.Parse(endpoint)
	if err != nil {
		log.Fatal(err)
	}

	return endpointURL.String()
}

func makeTestServer() (*httptest.Server, *Server, *App) {
	app := makeTestApp()
	server := NewServer(app)
	mux := mux.NewRouter()
	server.SetupRoutes(mux)
	ts := httptest.NewServer(mux)

	return ts, server, app
}

func makeTestClient() *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	return &http.Client{Jar: jar}
}
