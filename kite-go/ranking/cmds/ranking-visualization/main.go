//go:generate go-bindata -o bindata.go templates static
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-golib/envutil"
	"github.com/kiteco/kiteco/kite-golib/syntaxcolors"
	_ "github.com/mattn/go-sqlite3"
)

var (
	entrySlice  = make([]map[string]*entry, 0)
	seenQueries = make(map[string]struct{})
	queries     []*query

	codeExampleDB  = curation.GormDB(envutil.MustGetenv("CODEEXAMPLE_DB_DRIVER"), envutil.MustGetenv("CODEEXAMPLE_DB_URI"))
	runs           = curation.NewRunManager(codeExampleDB)
	snippetManager = curation.NewCuratedSnippetManager(codeExampleDB, runs)

	soPagesClient *stackoverflow.Client
)

// rankingResult is data structure we use to store results
// of ranking for a single query in the test set.
type rankingResult struct {
	QueryID       string      `json:"query_id"`
	QueryText     string      `json:"query_text"`
	QueryCode     string      `json:"query_code"`
	FeatureLabels []string    `json:"featurelabels"`
	SnapshotIDs   []int64     `json:"example_ids"`
	Labels        []float64   `json:"labels"`
	NDCG          float64     `json:"ndcg"`
	ExpectedRank  []int       `json:"expected_rank"`
	Scores        []float64   `json:"scores"`
	Features      [][]float64 `json:"features"`
}

// snippet is the generic structure that holds the data we
// want to display for each documnet.
type snippet struct {
	Code          []byte
	Title         string
	Rank          int
	Label         float64
	ExpectedRank  int
	Score         float64
	Features      []float64
	FeatureLabels []string
	IsSO          bool
	SOPage        soPage
}

// entry stores the ndcg score the ranker obtained for
// a given query.
type entry struct {
	Score    float64
	Snippets []snippet
}

// query encapsulates a query for display.
type query struct {
	Text     string
	IsActive bool
	URL      string
	Score    float64
}

// queriesByScore is used for sorting the queries by ndcg score.
type queriesByScore []*query

func (qs queriesByScore) Len() int           { return len(qs) }
func (qs queriesByScore) Swap(i, j int)      { qs[j], qs[i] = qs[i], qs[j] }
func (qs queriesByScore) Less(i, j int) bool { return qs[j].Score < qs[i].Score }

// Define a type named "folderSlice" as a slice of strings
type folderSlice []string

// String implements the flag.Value interface
func (f *folderSlice) String() string {
	return fmt.Sprint(*f)
}

// Set implements the flag.Value interface
func (f *folderSlice) Set(value string) error {
	for _, folder := range strings.Split(value, ",") {
		*f = append(*f, folder)
	}
	return nil
}

func loadEntries(folder string, entries map[string]*entry) {
	in, err := os.Open(path.Join(folder, "test-results.json"))
	if err != nil {
		log.Fatal(err)
	}

	cs := make(map[int64]*curation.CuratedSnippet)

	decoder := json.NewDecoder(in)
	for {
		var result rankingResult
		err := decoder.Decode(&result)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if _, seen := seenQueries[result.QueryText]; !seen {
			queries = append(queries, &query{
				Text:  result.QueryText,
				Score: result.NDCG,
				URL:   "/viewer?query=" + result.QueryText,
			})
			seenQueries[result.QueryText] = struct{}{}
		}

		var snippets []snippet
		var snip *curation.CuratedSnippet
		var exists bool

		for i := 0; i < len(result.SnapshotIDs); i++ {
			if snip, exists = cs[result.SnapshotIDs[i]]; !exists {
				var err error
				snip, err = snippetManager.GetBySnapshotID(result.SnapshotIDs[i])
				if err != nil {
					log.Fatalf("cannot get snippet with snapshot id %d\n", result.SnapshotIDs[i])
				}
				cs[result.SnapshotIDs[i]] = snip
			}
			snippets = append(snippets, snippet{
				Code:          []byte(snip.Prelude + "\n" + snip.Code),
				Title:         snip.Title,
				Rank:          i + 1,
				Label:         result.Labels[i],
				ExpectedRank:  result.ExpectedRank[i] + 1,
				Score:         result.Scores[i],
				Features:      result.Features[i],
				FeatureLabels: result.FeatureLabels,
			})
		}

		entries[result.QueryText] = &entry{
			Score:    result.NDCG,
			Snippets: snippets,
		}
	}
}

func main() {
	var (
		port    string
		folders folderSlice
		useSO   bool
		start   = time.Now()
	)

	flag.StringVar(&port, "port", ":3010", "port number with colon")
	flag.Var(&folders, "folders", "folders that contain test-results.json files. Seprated with ',' witout space, e.g., folder1,folder2")
	flag.BoolVar(&useSO, "so", false, "show results for stack overflow ranking")
	flag.Parse()

	if useSO {
		var err error
		soPagesClient, err = stackoverflow.NewClient(nil)
		if err != nil {
			log.Fatalf("unable to load stackoverflow.Client: %s", err)
		}
	}

	if flag.NArg() != 0 {
		log.Fatalf("%s cannot be processed successfully.\n", flag.Args())
	}

	for _, folder := range folders {
		entries := make(map[string]*entry)
		if useSO {
			loadEntriesSO(folder, entries)
		} else {
			loadEntries(folder, entries)
		}
		entrySlice = append(entrySlice, entries)
	}

	sort.Sort(sort.Reverse(queriesByScore(queries)))

	fmt.Println("took ", time.Now().Sub(start).Seconds(), "seconds to load data")

	http.HandleFunc("/static/syntaxcolors.css", handleSyntaxStylesheet)
	http.Handle("/static/", http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir}))
	http.HandleFunc("/viewer", handleReview)

	log.Println("Listening on " + port + "...")
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// Function map for the templating system
var templateFuncs = template.FuncMap{
	"colorize":           colorize,
	"colorizeWithCursor": colorizeWithCursor,
}

func colorize(code []byte) template.HTML {
	return syntaxcolors.Colorize(code, -1)
}

func colorizeWithCursor(code []byte, cursor int) template.HTML {
	return syntaxcolors.Colorize(code, cursor)
}

func handleSyntaxStylesheet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", mime.TypeByExtension(".css"))
	w.Write([]byte(syntaxcolors.DefaultStylesheet()))
}

type payload struct {
	Queries []*query
	Entries []*entry
}

func handleReview(w http.ResponseWriter, r *http.Request) {
	m, _ := url.ParseQuery(r.URL.RawQuery)
	if m != nil {
		var entries []*entry
		if len(m["query"]) > 0 {
			query := m["query"][0]
			for _, q := range queries {
				if q.Text == query {
					q.IsActive = true
				} else {
					q.IsActive = false
				}
			}
			for _, entry := range entrySlice {
				entries = append(entries, entry[query])
			}
		}
		payload := payload{
			Queries: queries,
			Entries: entries,
		}

		err := runTemplateWithFuncs(w, payload, templateFuncs, "index.html")
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
