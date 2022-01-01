package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/web/midware"
)

func main() {
	var (
		port         int
		models       string
		repositories string
		tunedModels  string
	)

	flag.IntVar(&port, "port", 8502, "port to listen for http requests")
	flag.StringVar(&models, "models", "/models", "location of models")
	flag.StringVar(&repositories, "repositories", "/repositories", "location of source code")
	flag.StringVar(&tunedModels, "tunedModels", "/tuned-models", "location of tuned models")
	flag.Parse()

	handlers := newServer(models, repositories, tunedModels)

	router := mux.NewRouter()
	router.HandleFunc("/model-assets/{lang}/{asset}", handlers.handleModelAsset)
	router.HandleFunc("/api/list", handlers.handleList)
	router.HandleFunc("/api/tune", handlers.handleTune)
	router.HandleFunc("/api/swap", handlers.handleSwap)
	router.HandleFunc("/api/upload", handlers.handleUpload)
	router.HandleFunc("/api/delete", handlers.handleDelete)

	logger := log.New(os.Stderr, log.Prefix(), log.LstdFlags|log.Lshortfile|log.Lmicroseconds)

	// Middleware
	neg := negroni.New(
		midware.NewRecovery(),
		midware.NewLogger(logger),
		negroni.Wrap(router),
	)

	http.ListenAndServe(fmt.Sprintf(":%d", port), neg)
}
