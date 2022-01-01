//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"log"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/contextutil"

	"net/http"
	_ "net/http/pprof"

	_ "expvar"
)

func main() {
	fail(datadeps.Enable())
	args := struct {
		Port string
	}{
		Port: ":3037",
	}
	arg.MustParse(&args)

	start := time.Now()

	app := newApp()

	r := mux.NewRouter()
	r.HandleFunc("/", app.HandleHome)
	r.HandleFunc("/build-samples", app.HandleBuildSamples).Methods("POST")
	r.HandleFunc("/sample", app.HandleSample).Methods("GET")

	r.PathPrefix("/debug/").Handler(http.DefaultServeMux)

	neg := negroni.New(
		midware.NewRecovery(),
		midware.NewLogger(contextutil.BasicLogger()),
		negroni.Wrap(r),
	)

	log.Printf("listening on http://localhost%s, took %v to load\n", args.Port, time.Since(start))
	fail(http.ListenAndServe(args.Port, neg))
}
