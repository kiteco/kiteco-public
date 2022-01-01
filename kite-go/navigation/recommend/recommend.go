package recommend

import (
	"errors"
	"path/filepath"
	"sort"
	"time"

	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/metrics"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Recommender recommends files and code blocks for predictive code navigation
type Recommender interface {
	Recommend(ctx kitectx.Context, request Request) ([]File, error)
	RecommendBlocks(ctx kitectx.Context, request BlockRequest) ([]File, error)
	RankedFiles() ([]File, error)
	ShouldRebuild() (bool, error)
}

// Options for NewRecommender
type Options struct {
	UseCommits           bool
	ComputedCommitsLimit int
	Root                 localpath.Absolute
	MaxFileSize          int64
	MaxFiles             int

	// Flag for reproducing the "underscores" experiment:
	// See kite-go/navigation/offline/experiments/underscores/README.md
	KeepUnderscores bool
}

var (
	errNonPositiveMaxFileSize = errors.New("MaxFileSize must be positive")
	errNonPositiveMaxFiles    = errors.New("MaxFiles must be positive")
)

func (o Options) validate() error {
	if o.MaxFileSize <= 0 {
		return errNonPositiveMaxFileSize
	}
	if o.MaxFiles <= 0 {
		return errNonPositiveMaxFiles
	}
	return nil
}

// Request for Recommend
type Request struct {
	// don't refresh before finding related files.
	SkipRefresh bool

	// use -1 to get all recs/keywords.
	MaxFileRecs      int
	MaxBlockRecs     int
	MaxFileKeywords  int
	MaxBlockKeywords int

	Location Location
	// optional param that will be used instead of
	// reading from the Location if provided
	BufferContents []byte
}

// BlockRequest for RecommendBlocks
type BlockRequest struct {
	Request

	// typically these are Files returned by Recommend, with nil Blocks and Keywords
	InspectFiles []File
}

// Location contains the requested region
type Location struct {
	CurrentPath string `json:"filename"`
	currentPath localpath.Absolute

	// uses 1-based indexing.
	// zero represents a path with no specific cursor line.
	CurrentLine int `json:"line,omitempty"`
}

var (
	// ErrInvalidCurrentLine ...
	ErrInvalidCurrentLine  = errors.New("Invalid CurrentLine")
	errRelativeCurrentPath = errors.New("CurrentPath must be absolute path")
	errRelativeInspectPath = errors.New("InspectPath must be absolute path")
)

func (r Request) validate() error {
	return r.Location.Validate()
}

// Validate reports whether the filepath is valid
// It's exported for use in the codenav APIs
func (l Location) Validate() error {
	if !filepath.IsAbs(l.CurrentPath) {
		return errRelativeCurrentPath
	}
	if l.CurrentLine < 0 {
		return ErrInvalidCurrentLine
	}
	return nil
}

func (r BlockRequest) validateBlockRequest() error {
	for _, file := range r.InspectFiles {
		if !filepath.IsAbs(file.Path) {
			return errRelativeInspectPath
		}
	}
	return r.Location.Validate()
}

// File data included in recommendation
type File struct {
	Path        string    `json:"absolute_path"`
	Probability float64   `json:"score"`
	Blocks      []Block   `json:"blocks"`
	Keywords    []Keyword `json:"keywords"`

	id   fileID
	path localpath.Absolute
}

// Block represents a code block.
// Lines numbers use 1-based indexing.
// FirstLine and LastLine are both inclusive bounds.
// This follows GitHub's UI. For example,
// https://github.com/kiteco/kiteco/blob/master/.gitignore#L1-L6
type Block struct {
	Content     string    `json:"content"`
	FirstLine   int       `json:"firstline"`
	LastLine    int       `json:"lastline"`
	Probability float64   `json:"score"`
	Keywords    []Keyword `json:"keywords"`
}

// Keyword explains a recommendation
type Keyword struct {
	Word  string  `json:"keyword"`
	Score float64 `json:"-"`
}

type recommender struct {
	fileIndex  *fileIndex
	graph      graph
	vectorizer vectorizer
	fileOpener *fileOpener
	ignorer    ignore.Ignorer
	gitStorage git.Storage
	params     parameters
	opts       Options
}

type parameters struct {
	vectorizerCoef        float64
	graphCoef             float64
	maxFileOpensPerSecond int
	maxMatrixSize         int
}

// NewRecommender loads a Recommender
func NewRecommender(ctx kitectx.Context, opts Options, i ignore.Ignorer, s git.Storage) (Recommender, error) {
	return newRecommender(ctx, opts, i, s)
}

func newRecommender(ctx kitectx.Context, opts Options, i ignore.Ignorer, s git.Storage) (recommender, error) {
	start := time.Now()
	err := opts.validate()
	if err != nil {
		return recommender{}, err
	}

	var r recommender
	r.opts = opts
	r.params = parameters{
		maxFileOpensPerSecond: 500,
		maxMatrixSize:         1e7,
	}
	r.ignorer = i
	r.gitStorage = s

	r.fileIndex = r.newFileIndex()
	r.fileOpener = r.newFileOpener()
	defer r.fileOpener.releaseMax()

	// Call loadVectorizer before loadGraph, because loadVectorizer handles
	// counting files and fast failing if there are too many files.
	err = r.loadVectorizer(ctx)
	if err != nil {
		return recommender{}, err
	}
	indexMetrics := metrics.Index{
		Duration: time.Since(start),
		NumFiles: int64(r.fileOpener.counterSize()),
	}
	indexMetrics.Log()

	if !opts.UseCommits {
		r.params.vectorizerCoef = 1
		r.params.graphCoef = 0
		return r, nil
	}

	r.graph, err = r.loadGraph(ctx)
	if err != nil {
		return recommender{}, err
	}
	r.params.vectorizerCoef = 7. / 8
	r.params.graphCoef = 1. / 8
	return r, nil
}

// Recommend file paths for predictive code navigation
func (r recommender) Recommend(ctx kitectx.Context, request Request) ([]File, error) {
	err := request.validate()
	if err != nil {
		return nil, err
	}
	request.Location.currentPath, err = localpath.NewAbsolute(request.Location.CurrentPath)
	if err != nil {
		return nil, err
	}
	currentID, err := r.fileIndex.toID(request.Location.currentPath)
	if err != nil {
		return nil, err
	}
	bs, err := r.baseContent(request)
	if err != nil {
		return nil, err
	}
	content := string(bs)
	return r.recommendFiles(ctx, currentID, content, request)
}

// RecommendBlocks finds the blocks in request.InspectPaths most relevant to the current position
func (r recommender) RecommendBlocks(ctx kitectx.Context, request BlockRequest) ([]File, error) {
	start := time.Now()
	err := request.validateBlockRequest()
	if err != nil {
		return nil, err
	}
	request.Location.currentPath, err = localpath.NewAbsolute(request.Location.CurrentPath)
	if err != nil {
		return nil, err
	}
	current, err := r.baseContent(request.Request)
	if err != nil {
		return nil, err
	}
	var files []File
	for _, inspectFile := range request.InspectFiles {
		inspectFile.path, err = localpath.NewAbsolute(inspectFile.Path)
		if err != nil {
			return nil, err
		}
		inspect, err := r.read(inspectFile.path)
		if err != nil {
			return nil, err
		}
		blocks, keywords, err := r.recommendBlocks(string(current), string(inspect), request.Request)
		if err != nil {
			return nil, err
		}
		files = append(files, File{
			Path:        inspectFile.Path,
			Probability: inspectFile.Probability,
			Blocks:      blocks,
			Keywords:    keywords,
		})
	}
	batchMetrics := metrics.Batch{
		Duration: time.Since(start),
		NumFiles: int64(len(files)),
	}
	batchMetrics.Log()
	return files, nil
}

// RankedFiles recommends an ordering of files
// An ordering based on activity used to be possible.
// Now the ordering is always lexicographic.
func (r recommender) RankedFiles() ([]File, error) {
	r.vectorizer.vectorSet.m.RLock()
	defer r.vectorizer.vectorSet.m.RUnlock()

	root := r.opts.Root.Clean()
	files := []File{{Path: string(root)}}
	seen := make(map[localpath.Absolute]bool)
	seen[root] = true
	for leafID := range r.vectorizer.vectorSet.data {
		leaf, err := r.fileIndex.fromID(leafID)
		if err != nil {
			return nil, err
		}
		for path := leaf; !seen[path]; path = path.Dir() {
			files = append(files, File{Path: string(path)})
			seen[path] = true
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func (r recommender) ShouldRebuild() (bool, error) {
	return r.ignorer.ShouldRebuild()
}

func (r recommender) recommendFiles(ctx kitectx.Context, currentID fileID, content string, request Request) ([]File, error) {
	start := time.Now()
	graphProbs := make(map[fileID]float64)
	if r.opts.UseCommits {
		graphRecs := r.graph.recommendFiles(currentID)
		for _, rec := range graphRecs {
			graphProbs[rec.id] = rec.Probability
		}
	}

	var numRefreshedFiles int
	if !request.SkipRefresh {
		var err error
		numRefreshedFiles, err = r.refreshVectorSet(ctx)
		if err != nil {
			return nil, err
		}
	}

	recs, err := r.vectorizer.recommendFiles(currentID, content, request)
	if err != nil {
		return nil, err
	}

	for i, rec := range recs {
		vectorizerPart := rec.Probability * r.params.vectorizerCoef
		graphPart := graphProbs[rec.id] * r.params.graphCoef
		recs[i].path, err = r.fileIndex.fromID(rec.id)
		if err != nil {
			return nil, err
		}
		recs[i].Probability = vectorizerPart + graphPart
		recs[i].Path = string(recs[i].path)
	}
	sort.Slice(recs, func(i, j int) bool {
		if recs[i].Probability == recs[j].Probability {
			return recs[i].Path < recs[j].Path
		}
		return recs[i].Probability > recs[j].Probability
	})

	numFiles := len(recs)
	if request.MaxFileRecs != -1 && len(recs) > request.MaxFileRecs {
		recs = recs[:request.MaxFileRecs]
	}
	rankMetrics := metrics.Rank{
		Duration:          time.Since(start),
		NumFiles:          int64(numFiles),
		NumRefreshedFiles: int64(numRefreshedFiles),
	}
	rankMetrics.Log()
	return recs, nil
}

func (r recommender) recommendBlocks(base, inspect string, request Request) ([]Block, []Keyword, error) {
	if request.MaxBlockRecs == 0 {
		return nil, nil, nil
	}
	blocks, err := r.vectorizer.recommendBlocks(base, inspect, request)
	if err != nil {
		return nil, nil, err
	}
	if request.MaxBlockRecs != -1 && len(blocks) > request.MaxBlockRecs {
		blocks = blocks[:request.MaxBlockRecs]
	}

	var keywords []Keyword
	seenForFile := make(map[string]bool)
	for _, block := range blocks {
		for _, keyword := range block.Keywords {
			if seenForFile[keyword.Word] {
				continue
			}
			keywords = append(keywords, keyword)
			seenForFile[keyword.Word] = true
		}
	}
	sort.Slice(keywords, func(i, j int) bool {
		if keywords[i].Score == keywords[j].Score {
			return keywords[i].Word < keywords[j].Word
		}
		return keywords[i].Score > keywords[j].Score
	})
	if request.MaxFileKeywords != -1 && len(keywords) > request.MaxFileKeywords {
		keywords = keywords[:request.MaxFileKeywords]
	}
	return blocks, keywords, nil
}

func (r recommender) baseContent(req Request) ([]byte, error) {
	if req.BufferContents != nil {
		return req.BufferContents, nil
	}
	return r.read(req.Location.currentPath)
}
