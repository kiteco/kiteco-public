//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"log"
	"net/http"
	"time"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/kiteco/kiteco/kite-golib/diskmapindex"
	"github.com/kiteco/kiteco/kite-golib/templateset"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"

	"github.com/alexflint/go-arg"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncode"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/contextutil"

	"github.com/kiteco/kiteco/local-pipelines/python-call-patterns/internal/data"
)

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	args := struct {
		Calls     string
		Patterns  string
		Port      string
		CacheRoot string
		Sources   string
	}{
		Port:      ":3095",
		CacheRoot: "/data",
		Sources:   pythoncode.HashToSourceIndexPath,
	}
	arg.MustParse(&args)

	start := time.Now()

	fail(datadeps.Enable())
	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	fail(<-errc)

	var idx *diskmapindex.Index
	if args.Calls != "" {
		var err error
		idx, err = diskmapindex.NewIndex(args.Calls, args.CacheRoot)
		fail(err)
	}

	fs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	templates := templateset.NewSet(fs, "templates", nil)

	var pbh data.PatternsByHash
	if args.Patterns != "" {
		var err error
		pbh, err = data.LoadPatterns(rm, args.Patterns)
		fail(err)
	}

	var sources *pythoncode.HashToSourceIndex
	if args.Sources != "" {
		var err error
		sources, err = pythoncode.NewHashToSourceIndex(args.Sources, args.CacheRoot)
		fail(err)
	}

	app := &app{
		rm:        rm,
		idx:       idx,
		templates: templates,
		patterns:  pbh,
		sources:   sources,
	}

	r := mux.NewRouter()
	r.HandleFunc("/", app.HandleHome)
	r.HandleFunc("/search", app.HandleSearch)
	r.HandleFunc("/source", app.HandleSource)

	neg := negroni.New(
		midware.NewRecovery(),
		midware.NewLogger(contextutil.BasicLogger()),
		negroni.Wrap(r),
	)

	log.Printf("listening on %s, took %v to load\n", args.Port, time.Since(start))
	fail(http.ListenAndServe(args.Port, neg))
}
