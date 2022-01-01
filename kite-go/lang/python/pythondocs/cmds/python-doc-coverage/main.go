//go:generate go-bindata -o bindata.go templates static/...
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythondocs"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

var (
	codeexampleRoot     = "s3://kite-emr/users/tarak/python-code-examples/2015-05-19_10-28-59-PM/%s/output/part-00000"
	defaultPackageStats = fmt.Sprintf(codeexampleRoot, "merge_count_incantations")
	defaultGroupedStats = fmt.Sprintf(codeexampleRoot, "merge_count_pkg_usages")
)

func main() {
	var port string
	var docfile, docstringsFile string
	var target string
	flag.StringVar(&port, "port", ":3030", "port to listen on")
	flag.StringVar(&docfile, "docfile", pythondocs.DefaultSearchOptions.DocPath, "python documentation file (gob.gz)")
	flag.StringVar(&docstringsFile, "docstringsfile", pythondocs.DefaultSearchOptions.DocstringsPath, "python documentation file for docstrings (gob.gz)")
	flag.StringVar(&target, "target", "static/target.txt", "target packages (defaults to all)")
	flag.Parse()

	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	templates := templateset.NewSet(staticfs, "templates", nil)
	h := newHandlers(docfile, docstringsFile, target, templates)

	http.HandleFunc("/package", h.handlePackage)
	http.HandleFunc("/documentation", h.handleDocumentation)
	http.Handle("/static/", http.FileServer(staticfs))
	http.HandleFunc("/", h.handleIndex)

	log.Println("listening on", port)
	log.Fatal(http.ListenAndServe(port, midware.Wrap(nil)))
}
