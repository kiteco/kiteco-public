package main

import (
	_ "expvar"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/capture"
	"github.com/kiteco/kiteco/kite-go/clientlogs"
	"github.com/kiteco/kiteco/kite-go/health"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-go/search"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-go/websandbox"
	"github.com/kiteco/kiteco/kite-golib/contextutil"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
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
		port         string
		noPrintStack bool
		noStackAll   bool
		stackSize    uint
	)

	flag.StringVar(&port, "port", envutil.GetenvDefault("USER_NODE_PORT", ":9090"), "port to listen on (e.g :9090)")
	flag.BoolVar(&noPrintStack, "noPrintStack", false, "disable printing stack traces")
	flag.BoolVar(&noStackAll, "noStackAll", false, "disable encluding other goroutines in stack traces")
	flag.UintVar(&stackSize, "stackSize", 1028*8, "stack size for stack traces")
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

	// Expose user list on private port (see goroutine above)
	debugRouter.HandleFunc("/localcode/artifacts", searchServices.Local.Handler)

	// Web-sandbox
	sandboxOpts := &websandbox.Options{
		Services:            searchServices.Python,
		IDCCCompleteOptions: api.IDCCCompleteOptions,
	}

	sandboxServer := websandbox.NewServer(sandboxOpts)

	// URL Routing
	router := mux.NewRouter()

	// Setup websandbox routes
	sandboxServer.SetupRoutes(router)

	// Websandbox ping Handler
	router.HandleFunc("/api/websandbox/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	nopMidware := func(h http.HandlerFunc) http.HandlerFunc { return h }

	// Code examples
	router.HandleFunc("/api/python/curation/{id:[0-9]+}", searchHandler.Python.HandleCuratedExample)
	router.HandleFunc("/api/python/curation/examples", searchHandler.Python.HandleCuratedExamples)

	// Kite Answers
	router.HandleFunc("/api/python/answers/{slug}", searchServices.Python.Answers.HandleHTTP)

	// Ping Handler
	router.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	// Driver endpoints
	eapi := editorapi.NewServer(python.NewEditorEndpoint(searchServices.Python, nil))
	router.PathPrefix("/api/editor/").Handler(eapi)

	// client log uploader
	logServer := clientlogs.NewServer(nopMidware)
	logServer.SetupRoutes(router)
	defer logServer.Close()

	// capture endpoint
	capture.NewServer(nopMidware).SetupRoutes(router)

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
	err = http.ListenAndServe(port, neg)
	if err != nil {
		log.Fatal(err)
	}
}

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
