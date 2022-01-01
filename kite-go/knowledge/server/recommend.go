package knowledge

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	errors "github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

func (a *App) setupRecommender(paths PathConfig) error {
	rootDir := paths.Root
	if !strings.HasSuffix(rootDir, string(os.PathSeparator)) {
		rootDir += string(os.PathSeparator)
	}

	ignoreOpts := ignore.Options{
		Root:            localpath.Absolute(rootDir),
		IgnoreFilenames: []localpath.Relative{ignore.GitIgnoreFilename, ignore.KiteIgnoreFilename},
	}
	ignorer, err := ignore.New(ignoreOpts)
	if err != nil {
		return err
	}

	opts := recommend.Options{
		UseCommits:           true,
		ComputedCommitsLimit: git.DefaultComputedCommitsLimit,
		Root:                 localpath.Absolute(rootDir),
		MaxFileSize:          1e6,
		MaxFiles:             1e5,
	}
	s, err := git.NewStorage(git.StorageOptions{
		UseDisk: true,
		Path: filepath.Join(
			os.Getenv("GOPATH"),
			"src", "github.com", "kiteco", "kiteco",
			"kite-go", "knowledge", "git-cache.json",
		),
	})
	if err != nil {
		return err
	}
	a.recommender, err = recommend.NewRecommender(kitectx.Background(), opts, ignorer, s)
	if err != nil {
		return err
	}

	files, err := a.recommender.RankedFiles()
	if err != nil {
		return err
	}
	a.index = newPathIndex(files)
	return nil
}

func newPathIndex(files []recommend.File) pathIndex {
	var index pathIndex
	index.isCode = make(map[string]bool)
	index.m = new(sync.RWMutex)
	for _, file := range files {
		if index.isCode[file.Path] {
			continue
		}
		index.isCode[file.Path] = true
		index.paths = append(index.paths, file.Path)
	}
	for file := range index.isCode {
		index.isCode[filepath.Dir(file)] = false
	}
	index.inverted = make(map[string]int)
	for i, path := range index.paths {
		index.inverted[path] = i
	}
	return index
}

// RecommendDisplay ...
type RecommendDisplay struct {
	Current   string
	Parent    Link
	HasParent bool
	File      string
	IsCode    bool
	Links     []Link
	RawPath   string
	NextNum   int
}

// Link ...
type Link struct {
	Path         string
	Probability  float64
	ScoreDisplay string
	Blocks       []BlockDisplay
	ID           int
	IsCode       bool
	FullPath     string
	GitHubURL    string
	VSCodeURL    string
}

// BlockDisplay ...
type BlockDisplay struct {
	recommend.Block
	ScoreDisplay string
	Lines        string
}

func (s *Server) runRecommend(w http.ResponseWriter, r *http.Request) {
	id, err := getID(r)
	if err != nil {
		s.showError(w, err)
		return
	}
	line, err := getLine(r)
	if err != nil {
		s.showError(w, err)
		return
	}
	path := s.app.paths.Root
	if id != -1 {
		path, err = s.app.index.getPathFromID(id)
		if err != nil {
			s.showError(w, err)
			return
		}
	}
	log.Printf("code nav: recommending id: %d, line: %d -> path: %s", id, line, path)
	request := recommend.Request{
		Location: recommend.Location{
			CurrentPath: path,
			CurrentLine: line,
		},
		MaxFileRecs:      maxFileRecs,
		MaxBlockRecs:     maxBlockRecs,
		MaxBlockKeywords: maxBlockKeywords,
	}
	display, err := s.MakeRecommendDisplay(request)
	if err != nil {
		s.showError(w, err)
		return
	}
	err = s.templates.Render(w, "recommend.html", display)
	if err != nil {
		s.showError(w, err)
		return
	}
}

func (s *Server) runRecommendFromPathImpl(w http.ResponseWriter, r *http.Request) {
	s.app.recommenderLock.RLock()
	defer s.app.recommenderLock.RUnlock()

	if !s.app.recommenderInit {
		s.showOops(w, errors.New("recommender not ready yet"))
		return
	}

	path, rawPath, err := s.getPath(r)
	if err != nil {
		s.showOops(w, err)
		return
	}

	numFiles := maxFileRecs
	if s, ok := r.URL.Query()["num"]; ok {
		if n, err := strconv.Atoi(s[0]); err == nil {
			numFiles = n
		}
	}

	log.Printf("code nav: recommending %d files for path: %s", numFiles, path)

	s.app.index.maybeAddPath(path)

	request := recommend.Request{
		Location: recommend.Location{
			CurrentPath: path,
		},
		MaxFileRecs:      numFiles,
		MaxBlockRecs:     maxBlockRecs,
		MaxBlockKeywords: maxBlockKeywords,
	}
	display, err := s.MakeRecommendDisplay(request)
	if err != nil {
		s.showOops(w, err)
		return
	}
	display.RawPath = rawPath
	err = s.templates.Render(w, "related.html", display)
	if err != nil {
		s.showOops(w, err)
		return
	}
}

// MakeRecommendDisplay ...
func (s *Server) MakeRecommendDisplay(request recommend.Request) (RecommendDisplay, error) {
	display := RecommendDisplay{
		Current: s.app.paths.ProjectName(),
		NextNum: request.MaxFileRecs + maxFileRecs,
	}
	if request.Location.CurrentPath != s.app.paths.Root {
		rel, err := filepath.Rel(s.app.paths.Root, request.Location.CurrentPath)
		if err != nil {
			return RecommendDisplay{}, err
		}
		display.Current = filepath.ToSlash(rel)
		display.Parent, err = s.makeLink(recommend.File{Path: filepath.Dir(request.Location.CurrentPath)})
		if err != nil {
			return RecommendDisplay{}, err
		}
		display.HasParent = true
		_, display.File = filepath.Split(request.Location.CurrentPath)
	}

	if s.app.index.checkIsCode(request.Location.CurrentPath) {
		display.IsCode = true
		files, err := s.app.recommender.Recommend(kitectx.Background(), request)
		if err != nil {
			return RecommendDisplay{}, err
		}
		blockRequest := recommend.BlockRequest{
			Request:      request,
			InspectFiles: files,
		}
		recs, err := s.app.recommender.RecommendBlocks(kitectx.Background(), blockRequest)
		if err != nil {
			return RecommendDisplay{}, err
		}
		display.Links, err = s.makeLinks(recs)
		if err != nil {
			return RecommendDisplay{}, err
		}
		return display, nil
	}
	recs := s.app.index.collectChildren(request.Location.CurrentPath)
	if len(recs) > maxFiles {
		recs = recs[:maxFiles]
	}
	var err error
	display.Links, err = s.makeLinks(recs)
	if err != nil {
		return RecommendDisplay{}, err
	}
	return display, nil
}

func (s *Server) makeLinks(recs []recommend.File) ([]Link, error) {
	var links []Link
	for _, file := range recs {
		link, err := s.makeLink(file)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

func (s *Server) makeLink(file recommend.File) (Link, error) {
	if file.Path == s.app.paths.Root {
		return Link{
			Path: fmt.Sprintf("%s/", s.app.paths.ProjectName()),
			ID:   -1,
		}, nil
	}
	s.app.index.maybeAddPath(file.Path)
	id := s.app.index.getIDFromPath(file.Path)

	var blocks []BlockDisplay
	for _, block := range file.Blocks {
		blocks = append(blocks, makeBlockDisplay(block))
	}

	rel, err := filepath.Rel(s.app.paths.Root, file.Path)
	if err != nil {
		return Link{}, err
	}
	slashRel := filepath.ToSlash(rel)

	link := Link{
		Path:         slashRel,
		Probability:  file.Probability,
		ID:           id,
		IsCode:       s.app.index.checkIsCode(file.Path),
		ScoreDisplay: makeScoreDisplay(file.Probability),
		Blocks:       blocks,
		FullPath:     file.Path,
		GitHubURL:    makeGitHubURL(s.app.paths.GitHubURL, slashRel),
		VSCodeURL:    makeVSCodeURL(s.app.paths.Root, slashRel),
	}
	if !link.IsCode {
		link.Path += "/"
	}
	return link, nil
}

func makeBlockDisplay(block recommend.Block) BlockDisplay {
	return BlockDisplay{
		Block:        block,
		Lines:        makeLines(block.FirstLine, block.LastLine),
		ScoreDisplay: makeScoreDisplay(block.Probability),
	}
}

func makeScoreDisplay(p float64) string {
	return fmt.Sprintf("%.1f%%", p*100)
}

func makeLines(begin, end int) string {
	s := ""
	for i := begin; i <= end; i++ {
		s += fmt.Sprintf("%d\n", i)
	}
	return s
}

func makeGitHubURL(url, path string) string {
	return fmt.Sprintf("%s/blob/master/%s", url, path)
}

func localFilePath(root, path string) string {
	return filepath.Join(root, filepath.FromSlash(path))
}

func makeVSCodeURL(root, path string) string {
	return fmt.Sprintf("vscode://file/%s", localFilePath(root, path))
}

type pathIndex struct {
	paths    []string
	inverted map[string]int
	isCode   map[string]bool
	m        *sync.RWMutex
}

func (p *pathIndex) maybeAddPath(path string) {
	p.m.Lock()
	defer p.m.Unlock()

	if _, ok := p.inverted[path]; ok {
		return
	}
	p.inverted[path] = len(p.paths)
	p.paths = append(p.paths, path)
	p.isCode[path] = true
	return
}

func (p pathIndex) getIDFromPath(path string) int {
	p.m.RLock()
	defer p.m.RUnlock()
	return p.inverted[path]
}

func (p pathIndex) getPathFromID(id int) (string, error) {
	p.m.RLock()
	defer p.m.RUnlock()

	if id < 0 || id >= len(p.paths) {
		return "", errors.New("id not found in pathIndex")
	}
	return p.paths[id], nil
}

func (p pathIndex) checkIsCode(path string) bool {
	p.m.RLock()
	defer p.m.RUnlock()
	return p.isCode[path]
}

func (p pathIndex) collectChildren(currentPath string) []recommend.File {
	p.m.RLock()
	defer p.m.RUnlock()

	var recs []recommend.File
	for _, path := range p.paths {
		if strings.HasPrefix(filepath.Base(path), ".") {
			continue
		}
		if filepath.Dir(path) != currentPath {
			continue
		}
		recs = append(recs, recommend.File{Path: path})
	}
	return recs
}
