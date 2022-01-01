//go:generate go-bindata -pkg main -o bindata.go scripts
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/api"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-go/websandbox"
)

func main() {
	datadeps.Enable()

	pythonOpts := python.DefaultServiceOptions
	services, err := websandbox.LoadServices(&pythonOpts)
	if err != nil {
		log.Fatalf("error loading python services: %v", err)
		return
	}

	sandboxOpts := &websandbox.Options{
		Services:            services,
		IDCCCompleteOptions: api.IDCCCompleteOptions,
		SandboxRecordMode:   true,
	}
	sandboxServer := websandbox.NewServer(sandboxOpts)

	router := mux.NewRouter()
	sandboxServer.SetupRoutes(router)

	cors := handlers.CORS(
		// These are headers we use in webapp fetch requests that aren't part of
		// CORS whitelisted headers.
		// https://fetch.spec.whatwg.org/#cors-safelisted-request-header
		handlers.AllowedHeaders([]string{"content-type", "pragma", "cache-control"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "DELETE", "PATCH", "PUT"}),
		handlers.AllowedOriginValidator(func(origin string) bool { return true }),
		handlers.AllowCredentials(),
	)

	midware := negroni.New(
		midware.NewLogger(log.New(os.Stdout, "[completions] ", log.Flags())),
		negroni.Wrap(cors(router)),
	)

	log.Println("listening on :3030")
	log.Fatalln(http.ListenAndServe(":3030", midware))
}
