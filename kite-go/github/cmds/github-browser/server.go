//go:generate go-bindata -o bindata.go templates static
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/github"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
	gorp "gopkg.in/gorp.v1"
)

const (
	numSnippetToShow = 10
)

var (
	dbpath          string
	port            string
	stats           string
	patternsFile    string
	clusterDir      string
	defaultGithubDB = curation.DatabaseURI
	githubDB        *gorp.DbMap // the database that stores the clustering results
	packages        []*github.Package
	templates       map[string]*template.Template
	htmlTemplates   = []string{"templates/index.html"}
)

type snippet struct {
	Code      string `json:"code"`
	Statement string `json:"statement"`
}

type cluster struct {
	ID             int
	Snippets       []*snippet `json:"snippets"`
	Representative *snippet   `json:"representative"`
	X              float64    `json:"x"`
	Y              float64    `json:"y"`
	Size           int        `json:"size"`
	Percentage     float64    `json:"percentage"`
}

type queryResult struct {
	CodeClusters    []*cluster `json:"codeClusters"`
	CooccurClusters []*cluster `json:"cooccurClusters"`
	PatternClusters []*cluster `json:"patternClusters"`
}

func handleFetchPackageStats(w http.ResponseWriter, r *http.Request) {
	// Write json
	var packageOnly []*github.Package
	for _, p := range packages {
		packageOnly = append(packageOnly, &github.Package{
			Name: p.Name,
			Cdf:  p.Cdf,
			Freq: p.Freq,
		})
	}

	encoder := json.NewEncoder(w)
	err := encoder.Encode(packageOnly)
	if err != nil {
		webutils.ReportError(w, "cannot encode package stats: %v\n", err)
	}
}

func handleFetchSubmoduleStats(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("packageName")
	// Write json
	submodules := findSubmodules(name)
	if submodules == nil {
		webutils.ReportBadRequest(w, "cannot find the requested package name %s", name)
		return
	}
	var submoduleOnly []*github.Submodule
	for _, s := range submodules {
		submoduleOnly = append(submoduleOnly, &github.Submodule{
			Name: s.Name,
			Cdf:  s.Cdf,
			Freq: s.Freq,
		})
	}
	encoder := json.NewEncoder(w)
	err := encoder.Encode(submoduleOnly)
	if err != nil {
		webutils.ReportError(w, "cannot encode submodule stats: %v\n", err)
	}
}

func handleFetchMethodStats(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("submoduleName")
	packageName := strings.Split(name, `.`)[0]
	log.Println("Package name: ", packageName)
	log.Println("Submodule name: ", name)

	submodules := findSubmodules(packageName)
	if submodules == nil {
		webutils.ReportBadRequest(w, "cannot find the requested package name %s\n", name)
	}

	var methods []*github.Method
	for _, s := range submodules {
		if s.Name == name {
			methods = s.Methods
			break
		}
	}
	if methods == nil {
		webutils.ReportBadRequest(w, "cannot find the requested submodules %s\n", name)
	}

	var methodOnly []*github.Method
	for _, m := range methods {
		methodOnly = append(methodOnly, &github.Method{
			Name: m.Name,
			Cdf:  m.Cdf,
			Freq: m.Freq,
		})
	}
	encoder := json.NewEncoder(w)
	err := encoder.Encode(methodOnly)
	if err != nil {
		webutils.ReportError(w, "cannot encode method stats: %v\n", err)
	}
}

func handleNavigation(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "templates/index.html", nil)
}

func handleStatsQuery(w http.ResponseWriter, r *http.Request) {
	// Decode json to get queries
	query := r.FormValue("query")
	num, err := strconv.Atoi(r.FormValue("numClusters"))
	if err != nil {
		webutils.ReportBadRequest(w, "unable to parse numClusters into an int: %v\n", err)
		return
	}
	log.Printf("query: %s and num: %d", query, num)

	codeClusters := fetchCodeClusters(w, query, num)
	cooccurClusters := fetchCooccurClusters(w, query)
	patternClusters := clustersForMethod(query)

	result := queryResult{
		CodeClusters:    codeClusters,
		CooccurClusters: cooccurClusters,
		PatternClusters: patternClusters,
	}

	// Write json
	encoder := json.NewEncoder(w)
	if err = encoder.Encode(result); err != nil {
		webutils.ReportError(w, "cannot encode clusters: %v\n", err)
	}
}

func fetchCodeClusters(w http.ResponseWriter, query string, num int) []*cluster {
	// Query the database and get the clusters
	var gcs []curation.GithubCluster
	sql := `SELECT * FROM GithubCluster WHERE FullIdent=? AND NumClusters=? ORDER BY ID ASC`
	_, err := githubDB.Select(&gcs, sql, query, num)
	if err != nil {
		webutils.ReportError(w, "cannot retrieve github clusters: %v\n", err)
		return nil
	}

	// Retrieve the code snippets for each cluster
	var clusters []*cluster
	for _, gc := range gcs {
		var gsnippets []*curation.GithubSnippet
		sql := `SELECT * FROM GithubSnippet WHERE FullIdent=? AND NumClusters=? AND ClusterID=?`
		_, err := githubDB.Select(&gsnippets, sql, query, num, gc.ID)
		if err != nil {
			webutils.ReportError(w, "cannot retrieve github snippets: %v\n", err)
			return nil
		}

		// Get the number of code snippets to show
		l := numSnippetToShow
		if len(gsnippets) < numSnippetToShow {
			l = len(gsnippets)
		}

		var snippets []*snippet
		for _, s := range gsnippets[:l] {
			statement := []byte(strings.Trim(string(s.Statement), " \t\n"))
			snippet := snippet{
				Code:      webutils.ColorizeCode(s.Code),
				Statement: webutils.ColorizeCode(statement),
			}
			snippets = append(snippets, &snippet)
		}

		// Use the most concise snippet as the title
		titleID := mostConcise(gsnippets)
		code := gsnippets[titleID].Code
		statementSTR := string(gsnippets[titleID].Statement)
		statement := []byte(strings.Trim(normalizeText(statementSTR), " \t\n"))

		rep := &snippet{
			Code:      webutils.ColorizeCode(code),
			Statement: webutils.ColorizeCode(statement),
		}
		cluster := cluster{
			ID:             gc.ID,
			X:              gc.X,
			Y:              gc.Y,
			Size:           gc.Size,
			Snippets:       snippets,
			Percentage:     gc.Percentage,
			Representative: rep,
		}
		clusters = append(clusters, &cluster)
	}
	return clusters
}

func fetchCooccurClusters(w http.ResponseWriter, query string) []*cluster {
	// Decode json to get queries
	num := 5
	log.Printf("query: %s and num: %d", query, num)

	// Query the database and get the clusters
	var gcs []curation.FunctionCluster
	sql := `SELECT * FROM FunctionCluster WHERE FullIdent=? AND NumClusters=? ORDER BY ID ASC`
	_, err := githubDB.Select(&gcs, sql, query, num)
	if err != nil {
		webutils.ReportError(w, "cannot retrieve function clusters: %v\n", err)
		return nil
	}

	// Retrieve the code snippets for each cluster
	var clusters []*cluster
	for _, gc := range gcs {
		var gsnippets []*curation.FunctionSnippet
		sql := `SELECT * FROM FunctionSnippet WHERE FullIdent=? AND NumClusters=? AND ClusterID=?`
		_, err := githubDB.Select(&gsnippets, sql, query, num, gc.ID)
		if err != nil {
			webutils.ReportError(w, "cannot retrieve function snippets: %v\n", err)
			return nil
		}

		// Get the number of code snippets to show
		l := numSnippetToShow
		if len(gsnippets) < numSnippetToShow {
			l = len(gsnippets)
		}

		var snippets []*snippet
		for _, s := range gsnippets[:l] {
			snippet := snippet{
				Statement: webutils.ColorizeCode(s.Code),
			}
			snippets = append(snippets, &snippet)
		}

		// Use the most concise snippet as the title
		rep := &snippet{
			Statement: string(gc.Statement),
		}

		cluster := cluster{
			ID:             gc.ID,
			Size:           gc.Size,
			Snippets:       snippets,
			Percentage:     gc.Percentage,
			Representative: rep,
		}
		clusters = append(clusters, &cluster)
	}

	return clusters
}

func main() {
	flag.StringVar(&stats, "stats", "", "file that contains the stats of packages")
	flag.StringVar(&dbpath, "db", defaultGithubDB, "default db to github cluster data")
	flag.StringVar(&patternsFile, "patternsFile", "", "")
	flag.StringVar(&clusterDir, "clusterDir", "", "")
	flag.StringVar(&port, "port", ":8008", "path to documentation")
	flag.Parse()

	err := loadCanonicalClusters(patternsFile, clusterDir)
	if err != nil {
		log.Fatalf("cannot load canonical clusters: %v\n", err)
	}

	if stats == "" {
		log.Fatalln("must specify package stats file using --stats")
	}

	// Connect to DB
	db, err := sql.Open("mysql", defaultGithubDB)
	if err != nil {
		log.Fatalf("cannot connect to databse: %v\n", err)
	}

	// Open curation database
	githubDB, err = curation.OpenGithubClustersDb(db, gorp.MySQLDialect{
		Engine:   "InnoDB",
		Encoding: "UTF8",
	})
	if err != nil {
		log.Fatalf("cannot open the curation database: %v\n", err)
	}

	// Load tempaltes
	templates = make(map[string]*template.Template)
	log.Println("Loading templates...")
	loadTemplates()

	// Load package stats
	packages = github.LoadPackageStats(stats)
	computeStats(packages)
	for _, p := range packages {
		log.Println(p.Name, p.Freq)
	}
	log.Printf("Loaded stats for %d packages...\n", len(packages))

	http.Handle("/static/", http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir}))

	http.HandleFunc("/packages", handleFetchPackageStats)
	http.HandleFunc("/submodules", handleFetchSubmoduleStats)
	http.HandleFunc("/methods", handleFetchMethodStats)

	http.HandleFunc("/clusters", handleStatsQuery)
	http.HandleFunc("/", handleNavigation)

	log.Println("Listening on " + port + "...")
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("cannot listen and serve at port %s: %v\n", port, err)
	}
}
