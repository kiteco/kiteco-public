package main

import (
	_ "expvar"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/contextutil"
)

func main() {
	if err := datadeps.Enable(); err != nil {
		log.Fatal(err)
	}

	args := struct {
		Port  string
		Cache string
	}{
		Port:  ":3039",
		Cache: "/data",
	}
	arg.MustParse(&args)

	log.Printf("will bind to port %s", args.Port)

	start := time.Now()

	app, err := newApp(args.Cache)
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/symbol/meta-info", app.handleSymbolMetaInfo).Methods("POST")
	r.HandleFunc("/symbol/members", app.handleSymbolMembers).Methods("POST")
	r.HandleFunc("/symbol/sources", app.handleSymbolSources).Methods("POST")
	r.HandleFunc("/symbol/scores", app.handleSymbolScores).Methods("POST")
	r.HandleFunc("/symbol/packages", app.handleSymbolPackages).Methods("GET")
	r.HandleFunc("/symbol/imports", app.handleSymbolImports).Methods("POST")

	r.HandleFunc("/hash/source", app.handleHashSource).Methods("POST")

	r.HandleFunc("/session", app.handleSession).Methods("POST")
	r.HandleFunc("/session/kill", app.handleSessionKill).Methods("POST")

	r.HandleFunc("/sessions/info", app.handleSessionsInfo).Methods("GET")
	r.HandleFunc("/session/ping", app.handleSessionPing).Methods("POST")

	r.PathPrefix("/debug/").Handler(http.DefaultServeMux)

	neg := negroni.New(
		midware.NewRecovery(),
		midware.NewLogger(contextutil.BasicLogger()),
		negroni.Wrap(r),
	)

	log.Printf("listening on %s, took %v to load\n", args.Port, time.Since(start))
	log.Fatal(http.ListenAndServe(args.Port, neg))
}
