//go:generate go-bindata -o bindata.go templates
package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/gorilla/mux"
)

func main() {
	var port string
	flag.StringVar(&port, "port", ":3030", "port to listen on")
	flag.Parse()

	handlers := newHandlers()

	mux := mux.NewRouter()
	mux.HandleFunc("/", handlers.handleIndex)
	mux.HandleFunc("/staging", handlers.handleStaging)
	mux.HandleFunc("/clients", handlers.handleClients)
	mux.HandleFunc("/stagingclients", handlers.handleStagingClients)

	log.Println("listening on", port)
	log.Fatalln(http.ListenAndServe(port, mux))
}
