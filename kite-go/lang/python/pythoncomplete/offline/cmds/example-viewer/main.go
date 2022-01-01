//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/completion"

	"github.com/kiteco/kiteco/kite-go/client/datadeps"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"

	"github.com/alexflint/go-arg"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythoncomplete/offline/example"
	"github.com/kiteco/kiteco/kite-go/web/templates"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

type app struct {
	collection         example.Collection
	templates          templates.TemplateSet
	playgroundEndpoint string
	completionProvider *completion.Provider
}

func main() {
	args := struct {
		Dir        string `arg:"positional,required"`
		Port       int
		Playground string
	}{
		Port:       5555,
		Playground: "http://localhost:3456",
	}

	arg.MustParse(&args)

	maybeQuit(datadeps.Enable())
	rm, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	maybeQuit(<-errc)

	collection, err := example.NewCollection(args.Dir)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("found %d examples in %s", len(collection.Examples), args.Dir)
	if len(collection.Examples) == 0 {
		log.Fatal("no examples found")
	}

	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}

	app := app{
		playgroundEndpoint: args.Playground,
		collection:         collection,
		templates:          templateset.NewSet(staticfs, "templates", nil),
		completionProvider: completion.NewProvider(rm),
	}
	//app.computeAndWriteStats("/data/kite/mixing/stats.json", collection)

	r := mux.NewRouter()
	r.HandleFunc("/", app.handleRoot)
	r.HandleFunc("/examples", app.handleExamples)
	r.HandleFunc("/example", app.handleExample)

	log.Printf("listening on port %d", args.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", args.Port), r))
}

func (a *app) computeAndWriteStats(targetFile string, examples example.Collection) {
	stats := completion.ComputeStats(examples, a.completionProvider)
	jsonContent, err := json.MarshalIndent(stats.Stats, "", "  ")
	maybeQuit(err)
	err = ioutil.WriteFile(targetFile, jsonContent, 0644)
	maybeQuit(err)

}

func maybeQuit(err error) {
	if err != nil {
		panic(err)
	}
}
