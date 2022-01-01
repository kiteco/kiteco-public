package main

import (
	_ "expvar"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/kiteco/kiteco/kite-go/community/account/checkout"
	"github.com/kiteco/kiteco/kite-go/community/student"

	"github.com/kiteco/kiteco/kite-golib/domains"
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
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-go/search"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-go/websandbox"
	"github.com/kiteco/kiteco/kite-golib/contextutil"
	"github.com/kiteco/kiteco/kite-golib/envutil"
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

func main() {
	// Misc flags
	var (
		port string

		noCommunity    bool
		noDataServices bool
		noWebSandbox   bool
	)

	flag.StringVar(&port, "port", envutil.GetenvDefault("USER_NODE_PORT", ":9090"), "port to listen on (e.g :9090)")
	flag.BoolVar(&noCommunity, "no-community", false, "disable community server")
	flag.BoolVar(&noDataServices, "no-data-services", false, "disable data services")
	flag.BoolVar(&noWebSandbox, "no-web-sandbox", false, "disable web sandbox")
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

	router := mux.NewRouter()

	// community
	if !noCommunity {
		communityDBDriver := envutil.MustGetenv("COMMUNITY_DB_DRIVER")
		communityDBURI := envutil.MustGetenv("COMMUNITY_DB_URI")

		var DB gorm.DB
		DB = community.DB(communityDBDriver, communityDBURI)

		var settings community.SettingsProvider
		settings = community.NewSettingsManager()

		// users
		studentDomains, err := student.LoadStudentDomains()
		if err != nil {
			log.Fatal(err)
		}
		app := community.NewApp(DB, settings, studentDomains)
		if err := app.Migrate(); err != nil {
			log.Fatalln(err)
		}
		comm := community.NewServer(app)
		comm.SetupRoutes(router)

		// accounts
		stripeSecret := envutil.MustGetenv("STRIPE_SECRET")
		stripeWebhookSecret := envutil.MustGetenv("STRIPE_WEBHOOK_SECRET")
		octobatSecret := envutil.MustGetenv("OCTOBAT_SECRET")
		octobatPublishable := envutil.MustGetenv("OCTOBAT_PUBLISHABLE")
		beanieConfigID := envutil.MustGetenv("BEANIE_CONFIG_ID")
		slackToken := envutil.GetenvDefault("SLACK_TOKEN", "")
		discourseSecret := envutil.GetenvDefault("DISCOURSE_SECRET", "")
		mixpanelSecret := envutil.GetenvDefault("MIXPANEL_SECRET", "")
		delightedSecret := envutil.GetenvDefault("DELIGHTED_SECRET", "")
		quickEmailToken := envutil.GetenvDefault("QUICK_EMAIL_TOKEN", "")
		licenseAuthorityKey := envutil.GetenvDefault("LICENSE_RSA_KEY", "")

		account.InitStripe(stripeSecret, stripeWebhookSecret, octobatSecret, octobatPublishable, beanieConfigID)
		plans, err := stripe.PlansFromStripe()
		if err != nil {
			log.Fatalf("Error while fetching plan information from stripe: %v", err)
		}

		authority, err := licensing.NewAuthorityFromPEMString(licenseAuthorityKey)
		if err != nil {
			log.Fatalln("Error while creating licensing.Authority : ", err)
		}

		ams := account.NewServer(app, slackToken, discourseSecret, mixpanelSecret, delightedSecret, quickEmailToken, authority, plans)
		if err := ams.Migrate(); err != nil {
			log.Fatalf("failed to migrate databases for account management server: %v", err)
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
	}

	// data services
	if !noDataServices {
		// Python
		pythonOpts := python.DefaultServiceOptions

		// Combined services
		searchOpts := search.Options{
			PythonOptions: &pythonOpts,
		}

		searchServices, err := search.NewServices(&searchOpts)
		if err != nil {
			log.Fatalln("error building search services:", err)
		}

		searchHandler, err := search.NewServicesHandler(searchServices)
		if err != nil {
			log.Fatalln("error building search services handler:", err)
		}

		// Code examples
		router.HandleFunc("/api/python/curation/{id:[0-9]+}", searchHandler.Python.HandleCuratedExample)
		router.HandleFunc("/api/python/curation/examples", searchHandler.Python.HandleCuratedExamples)

		// Kite Answers
		router.HandleFunc("/api/python/answers/{slug}", searchServices.Python.Answers.HandleHTTP)

		// Driver endpoints
		eapi := editorapi.NewServer(python.NewEditorEndpoint(searchServices.Python, nil))
		router.PathPrefix("/api/editor/").Handler(eapi)

		// Web-sandbox
		if !noWebSandbox {
			sandboxOpts := &websandbox.Options{
				Services:            searchServices.Python,
				IDCCCompleteOptions: api.IDCCCompleteOptions,
			}
			sandboxServer := websandbox.NewServer(sandboxOpts)

			// Setup websandbox routes
			sandboxServer.SetupRoutes(router)

			// Websandbox ping Handler
			router.HandleFunc("/api/websandbox/ping", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("pong"))
			})
		}
	}

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
		midware.NewLogger(logger),
		negroni.Wrap(cors(router)),
	)

	debugRouter.HandleFunc(health.ReadyEndpoint, health.ReadyHandler)

	log.Printf("Listening on %s...\n", port)
	err := http.ListenAndServe(port, neg)
	if err != nil {
		log.Fatal(err)
	}
}

// TODO(naman) reenable origin validation, no more XXXXXXX

var validHosts = [...]string{
	domains.WWW,
	"XXXXXXX",
}

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
