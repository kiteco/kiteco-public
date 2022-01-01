package main

import (
	_ "expvar"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/health"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/contextutil"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	_ "github.com/lib/pq"
)

func main() {
	logger := contextutil.BasicLogger()
	log.SetPrefix(logger.Prefix())
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Read environment vars
	var (
		localFilesDBDriver  = envutil.MustGetenv("LOCALFILES_DB_DRIVER")
		localFilesDBURI     = envutil.MustGetenv("LOCALFILES_DB_URI")
		localFilesStoreType = envutil.MustGetenv("LOCALFILES_STORE_TYPE")
		endpoint            = envutil.GetenvDefault("LOCAL_WORKER_PORT", ":9090")
		debugEndpoint       = envutil.GetenvDefault("LOCAL_WORKER_DEBUG_PORT", ":9091")
	)

	debugRouter := mux.NewRouter()
	go func() {
		// This is to let us do profiling and look at expvar on a non SSL
		// port. It also registers the default http.ServeMux in which
		// expvar and net/http/pprof register their handlers. It also becomes
		// available BEFORE data starts loading.
		debugRouter.PathPrefix("/debug/").Handler(http.DefaultServeMux)

		// Register health status endpoint
		debugRouter.HandleFunc(health.Endpoint, health.Handler)

		// Note that any handlers registered via `http` will be available on this port
		log.Println("debug endpoint is", debugEndpoint)
		log.Fatal(http.ListenAndServe(debugEndpoint, debugRouter))
	}()

	store, err := localfiles.ContentStoreFromEnv(localFilesDBDriver, localFilesDBURI, localFilesStoreType)
	if err != nil {
		log.Fatalln(err)
	}

	workerOpts := localcode.WorkerOptions{
		NumWorkers:          runtime.NumCPU(),
		FileCacheRoot:       envutil.GetenvDefault("LOCAL_WORKER_CACHE_DIR", "/var/kite/localfiles"),
		FileCacheSizeMB:     envutil.GetenvDefaultInt("LOCAL_WORKER_CACHE_SIZE_MB", 10240),
		ArtifactCacheRoot:   envutil.GetenvDefault("LOCAL_WORKER_ARTIFACT_DIR", "/var/kite/localartifacts"),
		ArtifactCacheSizeMB: envutil.GetenvDefaultInt("LOCAL_WORKER_ARTIFACT_SIZE_MB", 10240),
	}

	worker, err := localcode.NewWorker(workerOpts, store)
	if err != nil {
		log.Fatalln(err)
	}

	python, err := pythonbatch.NewBuilderLoader(pythonbatch.DefaultOptions)
	if err != nil {
		log.Fatalln(err)
	}

	// Register builders
	localcode.RegisterBuilder(lang.Python, python.Build)

	router := mux.NewRouter()
	worker.SetupRoutes(router)

	// Middleware
	neg := negroni.New(
		midware.NewRecovery(),
		&midware.StatusResponseCodes{},
		midware.NewLogger(logger),
		negroni.Wrap(router),
	)

	debugRouter.HandleFunc(health.ReadyEndpoint, health.ReadyHandler)

	log.Printf("listening on %s...\n", endpoint)
	err = http.ListenAndServe(endpoint, neg)
	if err != nil {
		log.Fatal(err)
	}
}
