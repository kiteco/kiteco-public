package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-answers/go/execution"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

func originValidator(origin string) bool {
	o, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if hostParts := strings.Split(o.Host, ":"); hostParts[0] == "localhost" {
		return true
	}

	if o.Scheme != "https" {
		return false
	}

	if strings.HasSuffix(o.Host, ".kite.com") || o.Host == "kite.com" {
		return true
	}

	return false
}

func main() {
	var port string
	flag.StringVar(&port, "port", ":80", "port to listen on")
	flag.Parse()
	datadeps.Enable()
	datadeps.SetLocalOnly()

	cors := handlers.CORS(
		// These are headers we use in webapp fetch requests that aren't part of
		// CORS whitelisted headers.
		// https://fetch.spec.whatwg.org/#cors-safelisted-request-header
		handlers.AllowedHeaders([]string{"content-type", "pragma", "cache-control"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "DELETE", "PATCH", "PUT"}),
		handlers.AllowedOriginValidator(originValidator),
		handlers.AllowCredentials(),
	)

	rootAssets, app := rootAssetsAndApp()
	router := mux.NewRouter().UseEncodedPath().StrictSlash(true)

	sandbox := execution.NewManager(kitectx.Background())
	resourceMgr, errc := pythonresource.NewManager(pythonresource.DefaultOptions.SymbolOnly())
	<-errc

	// local rendering
	err := handleLocal(router, sandbox, resourceMgr)
	if err != nil {
		panic(err)
	}

	// GitHub rendering
	err = handleGitHub(router, sandbox, resourceMgr, app)
	if err != nil {
		panic(err)
	}

	// Serve preview web app with these routes
	router.Path("/live/").Handler(http.StripPrefix("/live/", rootAssets))
	router.PathPrefix("/").Handler(rootAssets)

	log.Println("listening on", port)
	log.Fatalln(http.ListenAndServe(port, handlers.LoggingHandler(os.Stderr, cors(router))))
}
