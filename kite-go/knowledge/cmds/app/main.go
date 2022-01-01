package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/alexflint/go-arg"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	knowledge "github.com/kiteco/kiteco/kite-go/knowledge/server"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/contextutil"
)

const addr = ":8080"

var args = struct {
	ClosedPullsDir string
	OpenPullsDir   string
	Root           string
}{}

func main() {
	arg.MustParse(&args)
	if args.ClosedPullsDir == "" {
		log.Fatal(errors.New("closedpullsdir required"))
	}
	if args.OpenPullsDir == "" {
		log.Fatal(errors.New("openpullsdir required"))
	}
	if args.Root == "" {
		log.Fatal(errors.New("root required"))
	}

	paths := knowledge.PathConfig{
		ClosedPullsPath:   args.ClosedPullsDir,
		OpenPullsPath:     args.OpenPullsDir,
		Root:              args.Root,
		IgnoredDirRegexps: []string{`^\.`, "vendor", "bindata", "node_modules"},
	}

	server, err := knowledge.NewServer(paths, true)
	if err != nil {
		log.Fatalf("error setting up server: %s", err.Error())
	}

	r := mux.NewRouter()
	server.RegisterHandlers(r)
	neg := negroni.New(
		midware.NewRecovery(),
		midware.NewLogger(contextutil.BasicLogger()),
		negroni.Wrap(r),
	)
	log.Printf("localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, neg))
}
