//go:generate lessc static/css/style.less static/css/style.css
//go:generate lessc unified-frontend/less/unified-curation.less static/css/unified-curation.css
//go:generate touch static/js/bundle.js
//go:generate rm static/js/bundle.js
//go:generate npm run build
//go:generate go-bindata -o bindata.go -ignore=\.module-cache templates static/...

package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/negroni"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/curation/titleparser"
	"github.com/kiteco/kiteco/kite-go/health"
	"github.com/kiteco/kiteco/kite-go/sandbox"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/templateset"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const (
	logPrefix   = "[codeexample-author] "
	logFlags    = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
	defaultPort = ":3005"
)

func init() {
	log.SetFlags(logFlags)
	log.SetPrefix(logPrefix)
}

func main() {
	var (
		port            string
		word2vec        string
		verbs           string
		parserServer    string
		referenceServer string
		devStaticDir    string
		defaultUnified  bool
		pythonImage     string
	)

	flag.StringVar(&port, "port", defaultPort, "port the server should listen to with colon in front")
	flag.StringVar(&verbs, "verbs", "", "list of preferred verbs")
	flag.StringVar(&word2vec, "word2vec", "", "pre-trained word2vec model")
	flag.StringVar(&pythonImage, "pythonImage", "", "docker image in which to execute python code examples")
	flag.StringVar(&parserServer, "parser", "", "endpoint of the parser server (host:port)")
	flag.StringVar(&referenceServer, "reference", "http://curation.kite.com:4040/", "endpoint of the reference server")
	flag.StringVar(&devStaticDir, "dev", "", "reload templates from disk on every request")
	// this flag will be removed once we're 100% comfortable with using the unified curation tool:
	flag.BoolVar(&defaultUnified, "defaultUnified", true, "route /packages links to unified curation tool rather than old code authoring tool")
	flag.Parse()

	// Test that the local docker environment is working and fail fast if not
	if pythonImage != "" {
		stdout, stderr, err := sandbox.RunPythonCodeContainerized(
			`import numpy; print "test"`,
			pythonImage,
			sandbox.DefaultLimits)
		if err != nil {
			log.Fatalln("cannot run python code in docker container:\n" + err.Error())
		} else if stderr != "" {
			log.Fatalln("cannot run python code in docker container:\n" + stderr)
		} else if stdout != "test\n" {
			log.Fatalf("cannot run python code in docker container: expected 'test' but output was '%s'\n", stdout)
		} else {
			log.Println("Successfully executed test code in docker container")
		}
	}

	var funcMap = template.FuncMap{
		"string": func(x []byte) string { return string(x) },
	}
	// Load templates
	var templates *templateset.Set
	var staticfs http.FileSystem

	if devStaticDir != "" {
		log.Printf("hosting static files from %s", devStaticDir)
		staticfs = http.Dir(devStaticDir)
	} else {
		staticfs = &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	}
	templates = templateset.NewSet(staticfs, "templates", funcMap)

	err := templates.Validate()
	if err != nil {
		log.Fatal(err)
	}

	// Setup title validator
	var titleValidator *titleparser.TitleValidator
	if word2vec != "" && verbs != "" && parserServer != "" {
		titleValidator, err = titleparser.NewTitleValidator(word2vec, verbs, parserServer)
		if err != nil {
			log.Fatal(err)
		}
	}

	r := mux.NewRouter()

	codeExampleDB := curation.GormDB(envutil.MustGetenv("CODEEXAMPLE_DB_DRIVER"), envutil.MustGetenv("CODEEXAMPLE_DB_URI"))
	authDB := curation.GormDB(envutil.MustGetenv("CURATION_DB_DRIVER"), envutil.MustGetenv("CURATION_DB_URI"))

	app := NewApp(AppOptions{
		CodeExampleDB:   codeExampleDB,
		AuthDB:          authDB,
		ReferenceServer: referenceServer,
		Templates:       templates,
		TitleValidator:  titleValidator,
		DefaultUnified:  defaultUnified,
		PythonImage:     pythonImage,
		FrontendDev:     devStaticDir != "",
	})
	if err := app.Migrate(); err != nil {
		log.Fatal(err)
	}

	// Setup http routes
	app.SetupRoutes(r)

	// Register health status endpoint
	r.HandleFunc(health.Endpoint, health.Handler)

	// Execute and autoformat
	r.HandleFunc("/api/{language}/autoformat", app.Auth.Wrap(handleAutoformat)).Methods("POST")

	// Statis assets
	r.PathPrefix("/static/").Handler(http.FileServer(staticfs))

	logger := log.New(os.Stdout, logPrefix, logFlags)
	middleware := negroni.New(
		midware.NewRecovery(),
		midware.NewLogger(logger),
		negroni.Wrap(r), // Add handlers at the end of the chain.
	)

	log.Println("Listening on " + port)
	log.Fatal(http.ListenAndServe(port, middleware))
}

// --

func handleAutoformat(w http.ResponseWriter, r *http.Request) {
	// Parse submission
	prelude := r.PostFormValue("prelude")
	code := r.PostFormValue("code")
	postlude := r.PostFormValue("postlude")

	// Run autoformatter
	log.Println("Running autoformatter...")
	formatted, err := autoformatPythonSegments(prelude, code, postlude)
	if err != nil {
		webutils.ReportError(w, "error formatting python code: %v", err)
		return
	}

	// Construct result
	payload := map[string]string{
		"prelude":  formatted[0],
		"code":     formatted[1],
		"postlude": formatted[2],
	}

	// Write json back to client
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(payload); err != nil {
		webutils.ReportError(w, "error encoding json: %v", err)
	}
}
