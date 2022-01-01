package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-go/search"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-go/websandbox"
	"github.com/kiteco/kiteco/kite-golib/contextutil"
	"github.com/kiteco/kiteco/kite-golib/envutil"
)

var (
	logger = contextutil.BasicLogger()
)

func main() {
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

	// Combined services
	pythonOpts := python.DefaultServiceOptions
	searchOpts := search.Options{
		PythonOptions: &pythonOpts,
	}
	searchServices, err := search.NewServices(&searchOpts)
	if err != nil {
		log.Fatalln("error building search services:", err)
	}

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
