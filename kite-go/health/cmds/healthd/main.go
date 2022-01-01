//go:generate go-bindata -o bindata.go templates

package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

// endpoint is a simple tuple of name and endpoint
// that is provided via a json array that specifies
// the endpoints that healthd should be checking.
type endpoint struct {
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	GracePeriod int64  `json:"grace_period"`
}

// --

const (
	logPrefix = "[healthd] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
}

// --

func main() {
	var (
		port      string
		watchlist string
		interval  int64
	)

	flag.StringVar(&port, "port", ":4111", "port to listen on")
	flag.StringVar(&watchlist, "watchlist", "", "list of hosts to watch")
	flag.Int64Var(&interval, "interval", 30, "interval to poll hosts")
	flag.Parse()

	buf, err := ioutil.ReadFile(watchlist)
	if err != nil {
		log.Fatal("could not read watchlist:", err)
	}

	var endpoints []endpoint
	err = json.Unmarshal(buf, &endpoints)
	if err != nil {
		log.Fatal("could not unmarshal watchlist:", err)
	}

	// Make sure all endpoints are valid
	for _, ep := range endpoints {
		_, err = url.Parse(ep.Endpoint)
		if err != nil {
			log.Fatal("invalid url:", ep.Endpoint)
		}
	}

	tracker := newPollTracker(endpoints)

	// Fire off first check
	go tracker.poll()

	http.Handle("/", tracker)
	go func() {
		log.Println("Listening on", port, "...")
		log.Fatal(http.ListenAndServe(port, nil))
	}()

	// Check according to provided interval
	ticker := time.Tick(time.Duration(interval) * time.Second)
	for {
		<-ticker
		go tracker.poll()
	}
}
