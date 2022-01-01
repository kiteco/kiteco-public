package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/codegangsta/negroni"
	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-go/stackoverflow/cmd/internal/search"
	"github.com/kiteco/kiteco/kite-go/web/midware"
)

var (
	logPrefix = fmt.Sprintf("[%s] ", "so-searcher")
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
}

type searcher interface {
	Search(string, stackoverflow.SearchType, int) ([]int64, error)
}

type handler struct {
	searchers map[stackoverflow.SearchMode]searcher
	pf        search.PageFinder
}

// Search handles retrieving a response for a given search query.
func (h handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	mode := stackoverflow.SearchMode(r.URL.Query().Get("mode"))
	if _, found := h.searchers[mode]; !found {
		http.Error(w, fmt.Sprintf("unsupported search mode %v", mode), http.StatusBadRequest)
		return
	}

	mr, err := strconv.Atoi(r.URL.Query().Get("mr"))
	if err != nil {
		http.Error(w, fmt.Sprintf("error parsing max results value %s", err), http.StatusBadRequest)
		return
	}

	st := stackoverflow.SearchType(r.URL.Query().Get("st"))
	switch st {
	case stackoverflow.Disjunction, stackoverflow.Conjunction:
	default:
		http.Error(w, fmt.Sprintf("unsupported search type %v", st), http.StatusBadRequest)
		return
	}

	ids, err := h.searchers[mode].Search(query, st, mr)
	if err != nil {
		http.Error(w, fmt.Sprintf("searcher error %s", err), http.StatusInternalServerError)
		return
	}

	var pages []*stackoverflow.StackOverflowPage
	for _, id := range ids {
		page, err := h.pf.Find(id)
		if err != nil {
			log.Printf("error finding page: %s \n", err)
			continue
		}
		pages = append(pages, page)
	}
	write(pages, w)
}

// Lookup handles looking up pages directly by id.
func (h handler) Lookup(w http.ResponseWriter, r *http.Request) {
	idList := r.URL.Query().Get("ids")
	if idList == "" {
		http.Error(w, "No post ids provided", http.StatusBadRequest)
		return
	}

	var pages []*stackoverflow.StackOverflowPage
	parts := strings.Split(idList, ",")
	for _, idstr := range parts {
		postID, err := strconv.ParseInt(idstr, 10, 64)
		if err != nil {
			log.Println("ignoring invalid post id:", idstr)
			continue
		}
		page, err := h.pf.Find(postID)
		if err != nil {
			log.Println(err)
			continue
		}
		pages = append(pages, page)
	}
	write(pages, w)
}

// write writes the provived pages to w as json.
func write(pages []*stackoverflow.StackOverflowPage, w http.ResponseWriter) {
	buf, err := json.Marshal(&pages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func main() {
	var (
		port       string
		pagesDB    string
		useInMemPF bool
	)
	flag.StringVar(&port, "port", ":8090", "port to listen on")
	flag.StringVar(&pagesDB, "pages", "", "path to PageFinder data (leveldb root or so pages dump in GOB format)")
	flag.BoolVar(&useInMemPF, "inmempf", false, "use in memory page finder")
	flag.Parse()

	var pf search.PageFinder
	if useInMemPF {
		pages := make(search.PageFinderCompressed)
		f, err := os.Open(pagesDB)
		if err != nil {
			log.Fatal(err)
		}
		err = pages.LoadFromPagesDump(f)
		if err != nil {
			log.Fatal(err)
		}
		pf = pages
	} else {
		pages, err := search.NewPageFinderLevelDB(pagesDB)
		if err != nil {
			log.Fatal(err)
		}
		pf = pages
	}

	handler := handler{
		searchers: map[stackoverflow.SearchMode]searcher{
			stackoverflow.Bing:   newBingSearcher(),
			stackoverflow.Google: newGoogleSearcher(),
			stackoverflow.Kite:   newKiteSearcher(pf),
		},
		pf: pf,
	}

	http.HandleFunc("/search", handler.Search)
	http.HandleFunc("/posts", handler.Lookup)

	logger := log.New(os.Stdout, logPrefix, logFlags)
	middleware := negroni.New(
		midware.NewRecovery(),
		midware.NewLogger(logger),
		negroni.Wrap(http.DefaultServeMux),
	)

	log.Println("Listening on", port, "...")
	log.Fatal(http.ListenAndServe(port, middleware))
}
