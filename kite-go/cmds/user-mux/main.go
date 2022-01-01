package main

import (
	_ "expvar"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"path"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/health"
	"github.com/kiteco/kiteco/kite-go/web/midware"
	"github.com/kiteco/kiteco/kite-golib/contextutil"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	_ "github.com/lib/pq"
)

type targetNode struct {
	name        string
	routePrefix string
	debugPrefix string
	handlers    *proxyHandlers
}

var (
	logger = contextutil.BasicLogger()
	nodes  []*targetNode
)

func init() {
	// Set default logger flags and prefix to BasicLogger values.
	log.SetPrefix(logger.Prefix())
	log.SetFlags(logger.Flags())

	nodes = []*targetNode{
		&targetNode{
			routePrefix: "/api/websandbox/",
			debugPrefix: "/web-sandbox/",
			handlers: &proxyHandlers{
				node: "user-node", // point this to user-node; easiest way to merge the two
			},
		},
		&targetNode{
			routePrefix: "/",
			debugPrefix: "/",
			handlers: &proxyHandlers{
				node: "user-node",
			},
		},
	}
}

func main() {
	var (
		port            string
		communityURI    string
		communityDriver string
	)

	flag.StringVar(&port, "port", envutil.GetenvDefault("USER_MUX_PORT", ":9090"), "")
	flag.StringVar(&communityURI, "communityURI", envutil.MustGetenv("COMMUNITY_DB_URI"), "")
	flag.StringVar(&communityDriver, "communityDriver", envutil.MustGetenv("COMMUNITY_DB_DRIVER"), "")
	flag.Parse()

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
		log.Fatal(http.ListenAndServe(envutil.GetenvDefault("USER_MUX_DEBUG_PORT", ":9091"), debugRouter))
	}()

	db := community.DB(communityDriver, communityURI)
	manager := community.NewUserManager(db, nil)
	auth := &userAuthMidware{
		users:   manager,
		logger:  logger,
		website: newHybridWebsite(),
	}

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 200 if the user is authenticated, 403 if not
	http.HandleFunc("/require-auth", requireAuthHandler)

	for _, node := range nodes {
		http.HandleFunc(path.Join(node.routePrefix, health.ReadyEndpoint), node.handlers.handleReady)
		http.HandleFunc(node.routePrefix, node.handlers.handleHTTP)

		debugRouter.HandleFunc(path.Join(node.debugPrefix, health.ReadyEndpoint), node.handlers.handleReady)
		debugRouter.HandleFunc(path.Join(node.debugPrefix, "/bad-gateways"), node.handlers.handleGetBadGateway)
		debugRouter.HandleFunc(path.Join(node.debugPrefix, "/bad-gateway-paths"), node.handlers.handleGetBadGatewayPaths)

		err := node.handlers.refreshTargets()
		if err != nil {
			log.Println(err)
		}

		go node.handlers.refreshLoop()

		go node.handlers.watchBadGateways()
	}

	neg := negroni.New(
		midware.NewRecovery(),
		auth,
		negroni.Wrap(http.DefaultServeMux),
	)

	log.Println("listening on", port, "...")
	log.Fatalln(http.ListenAndServe(port, neg))
}
