package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"github.com/kiteco/kiteco/kite-go/community/account/checkout"
	"github.com/kiteco/kiteco/kite-go/community/student"
	"go.uber.org/zap"

	"github.com/kiteco/kiteco/kite-golib/gkeutil"
	"github.com/kiteco/kiteco/kite-golib/licensing"

	"github.com/codegangsta/negroni"
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
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/stripe"
)

var (
	region   = envutil.GetenvDefault("AWS_REGION", "")
	hostname = envutil.GetenvDefault("HOSTNAME", "")
	logger   = gkeutil.Logger.With(zap.String("region", region), zap.String("hostname", hostname)).Sugar()
)

func main() {
	defer logger.Sync()

	var port string
	flag.StringVar(&port, "port", fmt.Sprintf(":%s", envutil.GetenvDefault("USER_NODE_SERVICE_PORT", "9090")), "port to listen on (e.g :9090)")
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
		logger.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", envutil.GetenvDefault("USER_NODE_SERVICE_PORT_DEBUG", "9091")), debugRouter))
	}()

	router := mux.NewRouter()

	communityDBDriver := envutil.MustGetenv("COMMUNITY_DB_DRIVER")
	communityDBURI := envutil.MustGetenv("COMMUNITY_DB_URI")

	var DB gorm.DB
	DB = community.DB(communityDBDriver, communityDBURI)

	var settings community.SettingsProvider
	settings = community.NewSettingsManager()

	// users
	studentDomains, err := student.LoadStudentDomains()
	if err != nil {
		logger.Fatal(err)
	}
	app := community.NewApp(DB, settings, studentDomains)
	if err := app.Migrate(); err != nil {
		logger.Fatal(err)
	}
	comm := community.NewServer(app)
	comm.SetupRoutes(router)

	// accounts
	stripeSecret := envutil.MustGetenv("STRIPE_SECRET")
	stripeWebhookSecret := envutil.MustGetenv("STRIPE_WEBHOOK_SECRET")
	octobatSecret := envutil.MustGetenv("OCTOBAT_SECRET")
	octobatPublishable := envutil.MustGetenv("OCTOBAT_PUBLISHABLE")
	slackToken := envutil.GetenvDefault("SLACK_TOKEN", "")
	discourseSecret := envutil.GetenvDefault("DISCOURSE_SECRET", "")
	mixpanelSecret := envutil.GetenvDefault("MIXPANEL_SECRET", "")
	delightedSecret := envutil.GetenvDefault("DELIGHTED_SECRET", "")
	quickEmailToken := envutil.GetenvDefault("QUICK_EMAIL_TOKEN", "")
	licenseAuthorityKey := envutil.GetenvDefault("LICENSE_RSA_KEY", "")

	account.InitStripe(stripeSecret, stripeWebhookSecret, octobatSecret, octobatPublishable, "")
	plans, err := stripe.PlansFromStripe()
	if err != nil {
		logger.Fatalf("Error while fetching plan information from stripe: %v", err)
	}
	authority, err := licensing.NewAuthorityFromPEMString(licenseAuthorityKey)
	if err != nil {
		logger.Fatal("Error while creating licensing.Authority: ", err)
	}

	ams := account.NewServer(app, slackToken, discourseSecret, mixpanelSecret, delightedSecret, quickEmailToken, authority, plans)
	if err := ams.Migrate(); err != nil {
		logger.Fatalf("failed to migrate databases for account management server: %v", err)
	}
	ams.SetupRoutes(router.PathPrefix("/api/account").Subrouter())
	ams.SetupWebRoutes(router.PathPrefix("/web/account").Subrouter())
	checkout.SetupRoutes(plans, router.PathPrefix("/api/checkout").Subrouter())

	// client log uploader
	logServer := clientlogs.NewServer(app.Auth.WrapNoBlock)
	logServer.SetupRoutes(router)
	defer logServer.Close()

	// capture endpoint
	capture.NewServer(app.Auth.WrapNoBlock).SetupRoutes(router)

	// TODO (naman) do we need both of these now?
	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	router.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	cors := handlers.CORS(
		// These are headers we use in webapp fetch requests that aren't part of
		// CORS whitelisted headers.
		// https://fetch.spec.whatwg.org/#cors-safelisted-request-header
		handlers.AllowedHeaders([]string{"content-type", "pragma", "cache-control"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "DELETE", "PATCH", "PUT"}),
		handlers.AllowedOriginValidator(originValidator),
		handlers.AllowCredentials(),
	)

	// Middleware
	neg := negroni.New(
		midware.NewRecovery(),
		&midware.StatusResponseCodes{},
		midware.NewLogger(zap.NewStdLog(logger.Desugar())),
		negroni.Wrap(cors(router)),
	)

	debugRouter.HandleFunc(health.ReadyEndpoint, health.ReadyHandler)

	logger.Info("Listening on", port, "...")
	err = http.ListenAndServe(port, neg)
	if err != nil {
		logger.Fatal(err)
	}
}

// TODO(naman) reenable origin validation, no more XXXXXXX

var validHosts = [...]string{"www.kite.com", "XXXXXXX"}

// Our origin validator will allow any origin thats HTTPS and in the
// kite.com domain. This should become a whitelist of hostnames to mitigate
// people doing tricky things with their /etc/hosts.
func originValidator(origin string) bool {

	// This is an epic hack to allow the NPM debug server's request to get through.
	// This should be thought through more, but right now don't want to interrupt dev.
	/* if origin == "http://localhost:3000" {
		return true
	}

	o, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if o.Scheme != "https" {
		return false
	}

	o.Host

	for _, host := range validHosts {
		if strings.HasSuffix(o.Host, fmt.Sprintf(".%s", host)) || o.Host == host {
			return true
		}
	} */
	return true
}
