package main

import (
	"errors"
	_ "expvar"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/kiteco/kiteco/kite-go/community/student"

	"github.com/codegangsta/negroni"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/kiteco/kiteco/kite-go/capture"
	"github.com/kiteco/kiteco/kite-go/clientlogs"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/community/account"
	"github.com/kiteco/kiteco/kite-go/health"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/contextutil"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/licensing"
	"github.com/kiteco/kiteco/kite-golib/stripe"
)

var (
	logger = contextutil.BasicLogger()
)

func init() {
	// Set default logger flags and prefix to BasicLogger values.
	log.SetPrefix(logger.Prefix())
	log.SetFlags(logger.Flags())
}

const (
	stripeTestSecretKey  = "XXXXXXX"
	stripeTestPubKey     = "XXXXXXX"
	octobatTestPublicKey = "XXXXXXX"
	octobatTestSecretKey = "XXXXXXX"
	postgresDBUri        = "postgres://accountuser:kite@localhost/account_test?sslmode=disable"
	postgresDriver       = "postgres"
	beanieConfigID       = "XXXXXXX"

	userAEmail   = "XXXXXXX@kite.com"
	userBEmail   = "XXXXXXX@kite.com"
	studentEmail = "XXXXXXX@myschool.ac-grenoble.fr"
	testPassword = "XXXXXXX"
)

func initTestUsers(app *community.App) {
	_, _, _ = app.Users.Create("John A", userAEmail, testPassword, "")
	_, _, _ = app.Users.Create("John B", userBEmail, testPassword, "")
	_, _, _ = app.Users.Create("John French", studentEmail, testPassword, "")
}

func main() {

	var (
		// Environment configuration
		communityDBDriver   string
		communityDBURI      string
		stripeSecret        string
		octobatSecret       string
		octobatPublishable  string
		licenseAuthorityKey string
	)

	communityDBDriver = postgresDriver
	communityDBURI = postgresDBUri
	stripeSecret = stripeTestSecretKey
	octobatSecret = octobatTestSecretKey
	octobatPublishable = octobatTestPublicKey
	licenseAuthorityKey = testRsaKey
	stripeWebhookSecret := envutil.MustGetenv("STRIPE_WEBHOOK_SECRET")

	// Misc flags
	var (
		port string
	)

	flag.StringVar(&port, "port", envutil.GetenvDefault("USER_NODE_PORT", ":9090"), "port to listen on (e.g :9090)")
	flag.Parse()

	debugRouter := mux.NewRouter()
	go func() {
		// This is to let us do profiling and look at expvar on a non SSL
		// port. It also registers the default http.ServeMux in which
		// expvar and net/http/pprof register their handlers. It also becomes
		// available BEFORE data starts loading.
		debugRouter.PathPrefix("/debug/").Handler(http.DefaultServeMux)

		// Register health status endpoint
		debugRouter.HandleFunc(health.Endpoint, health.Handler)

		// Note that any handlers registered via `http` will be available on this port
		log.Fatal(http.ListenAndServe(envutil.GetenvDefault("USER_NODE_DEBUG_PORT", ":9091"), debugRouter))
	}()

	studentDomains, err := student.LoadStudentDomains()
	if err != nil {
		log.Fatal(err)
	}
	var DB gorm.DB
	DB = community.DB(communityDBDriver, communityDBURI)

	var settings community.SettingsProvider
	settings = community.NewSettingsManager()

	app := community.NewApp(DB, settings, studentDomains)
	if err := app.Migrate(); err != nil {
		log.Fatalln(err)
	}
	initTestUsers(app)

	comm := community.NewServer(app)

	// URL Routing
	router := mux.NewRouter()

	// Community
	comm.SetupRoutes(router)

	// Ping Handler
	router.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	authority, err := buildAuthority(licenseAuthorityKey)
	if err != nil {
		log.Fatalln("Error while creating LicenseAuthority : ", err)
	}

	account.InitStripe(stripeSecret, stripeWebhookSecret, octobatSecret, octobatPublishable, beanieConfigID)
	plans, err := stripe.PlansFromStripe()
	if err != nil {
		log.Fatalf("Error while fetching plan information from stripe: %v", err)
	}

	ams := account.NewServer(app, "", "", "", "", "", authority, plans)
	if err := ams.Migrate(); err != nil {
		log.Fatalf("failed to migrate databases for account management server: %v", err)
	}

	checkout := buildApp(plans, app.Users)
	checkout.setupRoutes(router)

	ams.SetupRoutes(router.PathPrefix("/api/account").Subrouter())
	ams.SetupWebRoutes(router.PathPrefix("/web/account").Subrouter())

	// client log uploader
	logServer := clientlogs.NewServer(app.Auth.WrapNoBlock)
	logServer.SetupRoutes(router)
	defer logServer.Close()

	// capture endpoint
	capture.NewServer(app.Auth.WrapNoBlock).SetupRoutes(router)

	// Ping handler (note we also have /api/ping, but the clients are pointing to this right now
	// so we need to add this handler so the appui below doesn't handle it by sending 10kb of HTML)
	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cors := handlers.CORS(
		// These are headers we use in webapp fetch requests that aren't part of
		// CORS whitelisted headers.
		// https://fetch.spec.whatwg.org/#cors-safelisted-request-header
		handlers.AllowedHeaders([]string{"content-type", "pragma", "cache-control"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "DELETE", "PATCH", "PUT"}),
		handlers.AllowCredentials(),
	)

	// Middleware
	neg := negroni.New(
		midware.NewRecovery(),
		&midware.StatusResponseCodes{},
		midware.NewLogger(logger),
		negroni.Wrap(cors(router)),
	)

	debugRouter.HandleFunc(health.ReadyEndpoint, health.ReadyHandler)

	log.Printf("Listening on %s...\n", port)
	err = http.ListenAndServe(port, neg)
	if err != nil {
		log.Fatal(err)
	}
}

func buildAuthority(keyContent string) (*licensing.Authority, error) {
	if keyContent == "" {
		return nil, errors.New("Please provide the RSA key to use to sign the licenses (env var LICENSE_RSA_KEY)")
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(keyContent))
	if err != nil {
		return nil, err
	}
	return licensing.NewAuthorityWithKey(privateKey)

}

const testRsaKey = `-----BEGIN RSA PRIVATE KEY-----
XXXXXXX
-----END RSA PRIVATE KEY-----
`
