package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/curation"
)

const (
	logPrefix = "[suggestions] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
	port      = ":8081"
)

type byPath map[string][]string

var (
	suggestions map[string]byPath
)

func loadSuggestions(input string) {
	decoder := newDecoder(input)
	for {
		var entry curation.Suggestions
		err := decoder.Decode(&entry)
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				log.Printf("Encountered unexpected EOF for %s\n", input)
				break
			} else if err == io.EOF {
				log.Printf("Reached EOF. Exit gracefully. %s\n", input)
				break
			} else {
				log.Fatal(err)
				break
			}
		}

		// save in map according to source
		table, ok := suggestions[entry.Source]
		if !ok {
			table = make(byPath)
			suggestions[entry.Source] = table
		}
		table[entry.Ident] = entry.Suggestions
	}
}

func newDecoder(path string) *json.Decoder {
	in, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	decomp, err := gzip.NewReader(in)
	if err != nil {
		log.Fatal(err)
	}

	return json.NewDecoder(decomp)
}

func handler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(r.FormValue("query"))

	set := make(map[string]struct{})
	for src, table := range suggestions {
		strings, ok := table[path]
		if !ok {
			log.Printf("%s path not found in source %s\n", path, src)
			continue
		}
		for _, str := range strings {
			set[str] = struct{}{}
		}
	}

	var result []string
	for str := range set {
		result = append(result, str)
	}

	js, err := json.Marshal(result)
	if err != nil {
		log.Fatal("Error constructing json response: ", err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*") // allow cross domain AJAX requests
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func main() {
	var input, port string
	flag.StringVar(&input, "input", "", "directories with suggestions files (json.gz)")
	flag.StringVar(&port, "port", ":8081", "specify port")
	flag.Parse()

	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)

	log.Println("Loading suggestions...")
	suggestions = make(map[string]byPath)
	for _, dir := range strings.Split(input, ",") {
		files, err := ioutil.ReadDir(dir)
		if err != nil || len(files) == 0 {
			log.Printf("Error reading or no files in dir %s\n", dir)
			continue
		}

		for _, f := range files {
			loadSuggestions(filepath.Join(dir, f.Name()))
		}
	}

	r := mux.NewRouter()
	r.HandleFunc("/suggestions", handler).Methods("POST")
	http.Handle("/", r)

	log.Printf("Ready! Hit up /suggestions at port %s\n", port)
	http.ListenAndServe(port, nil)
}
