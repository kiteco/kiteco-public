//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

type app struct {
	defaultPath string
	templates   *templateset.Set
}

func toHTML(s interface{}) template.HTML {
	return template.HTML(fmt.Sprintf("%v", s))
}

func newApp(dbPath string) (app, error) {
	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}

	return app{
		defaultPath: dbPath,
		templates:   templateset.NewSet(staticfs, "templates", template.FuncMap{"toHTML": toHTML}),
	}, nil
}

type listing struct {
	Timestamp string
	Pipeline  string
	Name      string
	Status    string
	URL       string
}

func newListing(run rundb.RunInfo) listing {
	return listing{
		Timestamp: run.CreatedAt.Format(time.RFC3339),
		Pipeline:  run.Pipeline,
		Name:      run.Name,
		Status:    string(run.Status),
		URL:       runURL(run),
	}
}

func (a app) handleRoot(w http.ResponseWriter, r *http.Request) {
	url := strings.Replace(a.defaultPath, "s3://", "/list/s3/", 1)
	http.Redirect(w, r, url, http.StatusFound)
}

func (a app) handleList(w http.ResponseWriter, r *http.Request) {
	s3Path := strings.Replace(r.URL.Path, "/list/s3/", "s3://", 1)

	rdb, err := rundb.NewRunDB(s3Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("bad S3 path '%s': %v", s3Path, err), http.StatusBadRequest)
		return
	}

	runs, err := rdb.ListRuns(false)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting runs: %v", err), http.StatusInternalServerError)
		return
	}

	listings := make([]listing, 0, len(runs))
	for _, run := range runs {
		listings = append(listings, newListing(run))
	}

	noCache(w)
	err = a.templates.Render(w, "list.html", map[string]interface{}{
		"S3Dir": rdb.S3Dir(),
		"Runs":  listings,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a app) handleArtifact(w http.ResponseWriter, r *http.Request) {
	artifactPath := strings.Replace(r.URL.Path, "/artifact/s3/", "s3://", 1)
	log.Printf("reading artifact: %s", artifactPath)

	inf, err := fileutil.NewCachedReader(artifactPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot get reader for %s: %v", artifactPath, err), http.StatusNotFound)
		return
	}
	defer inf.Close()

	buf, err := ioutil.ReadAll(inf)
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading %s: %v", artifactPath, err), http.StatusInternalServerError)
		return
	}

	w.Write(buf)
}

func (a app) handleRun(w http.ResponseWriter, r *http.Request) {
	runPath := strings.Replace(r.URL.Path, "/run/s3/", "s3://", 1)

	run, err := rundb.NewRunInfoFromPath(runPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot get run for path '%s': %v", runPath, err), http.StatusNotFound)
		return
	}

	type feedStat struct {
		Feed  string
		Stats rundb.FeedStats
	}
	var feedStats []feedStat
	for k, v := range run.FeedStats {
		feedStats = append(feedStats, feedStat{Feed: k, Stats: v})
	}
	sort.Slice(feedStats, func(i, j int) bool {
		return feedStats[i].Feed < feedStats[j].Feed
	})

	type param struct {
		Name  string
		Value interface{}
	}
	var runParams []param
	for k, v := range run.Params {
		runParams = append(runParams, param{Name: k, Value: v})
	}
	sort.Slice(runParams, func(i, j int) bool {
		return runParams[i].Name < runParams[j].Name
	})

	type artifact struct {
		Name string
		URL  string
	}

	arts := run.Artifacts()
	artifacts := make([]artifact, 0, len(arts))
	for _, art := range arts {
		artifacts = append(artifacts, artifact{
			Name: art,
			URL:  artifactURL(run, art),
		})
	}

	type childRun struct {
		RelativePath string
		Run          listing
	}

	cr := run.ChildRuns()
	childRuns := make([]childRun, 0, len(cr))
	for _, c := range cr {
		childRuns = append(childRuns, childRun{RelativePath: c.RelativePath, Run: newListing(c.Info)})
	}

	noCache(w)
	err = a.templates.Render(w, "run.html", map[string]interface{}{
		"Run":       run,
		"RunPath":   run.S3Path(),
		"Params":    runParams,
		"Artifacts": artifacts,
		"ChildRuns": childRuns,
		"FeedStats": feedStats,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func runURL(run rundb.RunInfo) string {
	return fmt.Sprintf("/run/%s", rewriteS3URL(run.S3Path()))
}

func artifactURL(run rundb.RunInfo, artifact string) string {
	return fmt.Sprintf("/artifact/%s/%s", rewriteS3URL(run.S3Path()), artifact)
}

func rewriteS3URL(url string) string {
	return strings.Replace(url, "s3://", "s3/", 1)
}

func noCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Expires", "0")
}

func main() {
	args := struct {
		Path string
		Port int
	}{
		Path: rundb.DefaultRunDB,
		Port: 4444,
	}
	arg.MustParse(&args)

	a, err := newApp(args.Path)
	if err != nil {
		log.Fatalln(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/", a.handleRoot).Methods("GET")
	r.NewRoute().PathPrefix("/list/s3/").HandlerFunc(a.handleList).Methods("GET")
	r.NewRoute().PathPrefix("/run/s3/").HandlerFunc(a.handleRun).Methods("GET")
	r.NewRoute().PathPrefix("/artifact/s3/").HandlerFunc(a.handleArtifact).Methods("GET")

	host := "localhost"
	if n, err := os.Hostname(); err == nil {
		host = n
	}

	log.Printf("binding to address http://%s:%d", host, args.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", args.Port), r))
}
