package knowledge

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	errors "github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

// Default values that control responses from the server.
const (
	maxLineLength    = 150
	maxPulls         = 25
	maxCommentGroups = 25
	maxComments      = 25
	maxFileDiffs     = 25
	maxFiles         = 100
	maxLines         = 50
	maxFileRecs      = 5
	maxBlockRecs     = 5
	maxBlockKeywords = 3
)

// PathConfig contains the paths necessary to run recommendations.
type PathConfig struct {
	ClosedPullsPath       string
	OpenPullsPath         string
	Root                  string
	IgnoredDirRegexps     []string
	IgnoredPathSubstrings []string
	GitHubURL             string
}

// ProjectName returns the inferred project name from the root of the project.
func (p PathConfig) ProjectName() string {
	_, n := filepath.Split(p.Root)
	if n == "" {
		_, n = filepath.Split(p.Root[:len(p.Root)-1])
	}
	return n
}

// App recommends file blocks based on input file paths.
type App struct {
	paths       PathConfig
	recommender recommend.Recommender
	index       pathIndex
	validator   PreValidator

	debug           bool
	recommenderLock sync.RWMutex
	recommenderInit bool
}

// NewApp constructs a new App.
func NewApp(paths PathConfig, debug bool) (*App, error) {
	if paths.Root == "" {
		return nil, errors.New("Root must not be empty")
	}
	a := App{paths: paths, debug: debug}
	a.paths.Root = filepath.Clean(a.paths.Root)

	if !debug {
		go a.initRecommender()
		return &a, nil
	}

	if err := a.initRecommender(); err != nil {
		return nil, err
	}
	if err := a.initValidator(); err != nil {
		return nil, err
	}
	return &a, nil
}

func (a *App) initRecommender() error {
	log.Println("code nav: setting up recommender")
	if err := a.setupRecommender(a.paths); err != nil {
		log.Printf("code nav: recommender init failed %s\n", err.Error())
		return err
	}
	a.recommenderLock.Lock()
	a.recommenderInit = true
	a.recommenderLock.Unlock()
	log.Printf("code nav: recommender initialized")
	return nil
}

func (a *App) initValidator() error {
	log.Println("code nav: setting up validator")
	if err := a.setupValidation(); err != nil {
		log.Printf("code nav: validator init failed %s\n", err.Error())
		return nil
	}
	return nil
}

// ErrorDisplay ...
type ErrorDisplay struct {
	Error string
}

type handlerFunc func(w http.ResponseWriter, r *http.Request)

func (f handlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// Server forwards incoming requests to the App and either returns a response
// or renders a template to be viewed in the browser.
type Server struct {
	app                  *App
	fs                   *assetfs.AssetFS
	templates            *templateset.Set
	runRecommendFromPath handlerFunc
}

// NewServer creates a new Server.
func NewServer(path PathConfig, debug bool) (*Server, error) {
	app, err := NewApp(path, debug)
	if err != nil {
		return nil, err
	}
	fs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	s := &Server{
		app:       app,
		fs:        fs,
		templates: templateset.NewSet(fs, "server/templates", nil),
	}
	s.runRecommendFromPath = s.runRecommendFromPathImpl
	return s, nil
}

// Name implements component Core.
func (s *Server) Name() string {
	return "codenav"
}

// RegisterHandlers implements component Handlers. It associates endpoints with
// their handlers.
func (s *Server) RegisterHandlers(mux *mux.Router) {
	log.Println("code nav: registering handlers")
	mux.HandleFunc("/codenav", s.runIndex)
	mux.HandleFunc("/codenav/search/", s.runSearch)
	mux.HandleFunc("/codenav/recommend/", s.runRecommend)
	mux.HandleFunc("/codenav/recommend/{id}", s.runRecommend)
	mux.PathPrefix("/codenav/related/").Handler(s.runRecommendFromPath)
	mux.HandleFunc("/codenav/validate/", s.runOpen)
	mux.HandleFunc("/codenav/validate-files/{id}", s.runFiles)
	mux.HandleFunc("/codenav/validate-blocks/{id}", s.runBlocks)
	mux.PathPrefix("/server/static/").Handler(http.FileServer(s.fs))
}

func (s *Server) runIndex(w http.ResponseWriter, r *http.Request) {
	err := s.templates.Render(w, "index.html", nil)
	if err != nil {
		s.showError(w, err)
	}
}

func (s *Server) showError(w http.ResponseWriter, err error) {
	e := s.templates.Render(w, "error.html", ErrorDisplay{Error: err.Error()})
	if e != nil {
		log.Fatal(e)
	}
}

func (s *Server) showOops(w http.ResponseWriter, err error) {
	log.Printf("code nav: rendering error: %s", err.Error())
	e := s.templates.Render(w, "oops.html", ErrorDisplay{Error: err.Error()})
	if e != nil {
		log.Printf("code nav: error rendering error: %s", e.Error())
	}
}

func getID(r *http.Request) (int, error) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 {
		return 0, nil
	}
	if parts[3] == "" {
		return -1, nil
	}
	return strconv.Atoi(parts[3])
}

func getLine(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("line")
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}

var errNotInProject = errors.New("that file is not part of the project root")

func (s *Server) getPath(r *http.Request) (string, string, error) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		return "", "", errNotInProject
	}

	rawPath := strings.Join(parts[3:], "/")
	path := rawPath

	if runtime.GOOS == "windows" {
		// Capitalize the drive letter
		path = strings.ToUpper(string(path[0])) + path[1:]
	} else if !strings.HasPrefix(path, string(os.PathSeparator)) {
		// Chrome removes the leading slash
		path = string(os.PathSeparator) + path
	}

	log.Printf("code nav: raw path: %s", path)
	if !strings.HasPrefix(path, s.app.paths.Root) {
		log.Printf("code nav: path is not part of project root")
		return "", rawPath, errNotInProject
	}

	return path, rawPath, nil
}
