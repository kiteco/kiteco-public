package main

import (
	"log"
	"net/http"

	arg "github.com/alexflint/go-arg"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/contextutil"
)

func main() {
	args := struct {
		RemoteDir string
		LocalDir  string
		Hosts     []string
		Port      string
	}{}
	arg.MustParse(&args)

	log.Printf("syncing files from remote dir %s to local dir %s for hosts:", args.RemoteDir, args.LocalDir)
	for _, h := range args.Hosts {
		log.Printf("* %s", h)
	}

	a := newApp(args.LocalDir, args.RemoteDir, args.Hosts)

	for _, h := range args.Hosts {
		go func(host string) {
			s := syncer{
				host:      host,
				remoteDir: args.RemoteDir,
				localDir:  args.LocalDir,
				isUsed:    a.IsUsed,
			}
			s.SyncLoop()
		}(h)
	}

	go func() {
		a.DeleteLoop()
	}()

	r := mux.NewRouter()
	r.HandleFunc("/list", a.handleList).Methods("POST")
	r.HandleFunc("/used", a.handleUsed).Methods("POST")
	r.HandleFunc("/reset", a.handleReset).Methods("POST")

	r.PathPrefix("/debug/").Handler(http.DefaultServeMux)

	neg := negroni.New(
		midware.NewRecovery(),
		midware.NewLogger(contextutil.BasicLogger()),
		negroni.Wrap(r),
	)

	log.Printf("listening on %s\n", args.Port)
	log.Fatal(http.ListenAndServe(args.Port, neg))
}
