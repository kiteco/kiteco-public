package knowledge

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/offline/validation"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// PreValidator ...
type PreValidator struct {
	Pulls        []Pull
	Stats        validation.Stats
	fileDisplay  map[int]Pull
	blockDisplay []Document
}

// Pull ...
type Pull struct {
	Number             string
	Stats              validation.Stats
	Bases              []Base
	NumFiles           int
	isRelevant         map[string]bool
	files              []recommend.File
	rawStats           []validation.Stats
	pathRelevantBlocks map[string][]Block
}

// Base ...
type Base struct {
	Path        string
	Line        int
	Retrieved   []Document
	Relevant    []Document
	Stats       validation.Stats
	retrieved   []string
	relevant    []string
	isRetrieved map[string]Document
}

// Document ...
type Document struct {
	ID          int
	Path        string
	IsHit       bool
	Retrieved   []Block
	Relevant    []Block
	Stats       validation.Stats
	Probability float64
}

// Line ...
type Line struct {
	Number  string
	Content string
	IsHit   bool
}

// Block ...
type Block struct {
	Lines       []Line
	Probability float64
}

func (a *App) setupValidation() error {
	ignoreOpts := ignore.Options{
		Root:            localpath.Absolute(a.paths.Root),
		IgnoreFilenames: []localpath.Relative{ignore.GitIgnoreFilename},
	}
	ignorer, err := ignore.New(ignoreOpts)
	if err != nil {
		return err
	}
	l := validation.Loader{
		PullsPath: localpath.Absolute(a.paths.OpenPullsPath),
		Root:      localpath.Absolute(a.paths.Root),
		Ignorer:   ignorer,
	}
	validationData, err := l.Load()
	if err != nil {
		return err
	}

	for pullNumber, files := range validationData {
		err := a.addPull(string(pullNumber), files)
		if err != nil {
			return err
		}
	}

	sort.Slice(a.validator.Pulls, func(i, j int) bool {
		return a.validator.Pulls[i].Stats.F1 < a.validator.Pulls[j].Stats.F1
	})
	a.validator.fileDisplay = make(map[int]Pull)
	for _, pull := range a.validator.Pulls {
		pullNumber, err := strconv.Atoi(pull.Number)
		if err != nil {
			return err
		}
		a.validator.fileDisplay[pullNumber] = pull
	}

	var pullStats []validation.Stats
	for _, pull := range a.validator.Pulls {
		pullStats = append(pullStats, pull.Stats)
	}
	a.validator.Stats = validation.Mean(pullStats)

	return nil
}

func (a *App) addPull(pullNumber string, files []recommend.File) error {
	pull := Pull{
		Number: pullNumber,
		files:  files,
	}
	if len(pull.files) <= 1 {
		return nil
	}

	pull.isRelevant = make(map[string]bool)
	for _, file := range pull.files {
		pull.isRelevant[file.Path] = true
	}

	pull.pathRelevantBlocks = make(map[string][]Block)
	for _, file := range files {
		pull.pathRelevantBlocks[file.Path] = processBlocks(file.Blocks)
	}

	for _, file := range pull.files {
		base, err := a.addFile(file, pull)
		if err == recommend.ErrInvalidCurrentLine {
			continue
		}
		if err != nil {
			return err
		}
		pull.Bases = append(pull.Bases, base)
		pull.rawStats = append(pull.rawStats, base.Stats)
	}
	pull.NumFiles = len(pull.Bases)
	pull.Stats = validation.Mean(pull.rawStats)
	a.validator.Pulls = append(a.validator.Pulls, pull)
	return nil
}

func (a *App) addFile(file recommend.File, pull Pull) (Base, error) {
	request := recommend.Request{
		Location: recommend.Location{
			CurrentPath: file.Path,
			CurrentLine: validation.PickLine(file),
		},
		MaxFileRecs:  maxFileRecs,
		MaxBlockRecs: maxBlockRecs,
	}
	files, err := a.recommender.Recommend(kitectx.Background(), request)
	if err != nil {
		return Base{}, err
	}
	blockRequest := recommend.BlockRequest{
		Request:      request,
		InspectFiles: files,
	}
	recs, err := a.recommender.RecommendBlocks(kitectx.Background(), blockRequest)
	if err != nil {
		return Base{}, err
	}

	base := Base{
		Path: request.Location.CurrentPath,
		Line: request.Location.CurrentLine,
	}
	base.isRetrieved = make(map[string]Document)
	for _, rec := range recs {
		base.retrieved = append(base.retrieved, rec.Path)
		doc := a.addRec(rec, pull)
		base.Retrieved = append(base.Retrieved, doc)
		base.isRetrieved[rec.Path] = doc
	}

	for _, other := range pull.files {
		if other.Path == file.Path {
			continue
		}
		base.relevant = append(base.relevant, other.Path)
		doc, isHit := base.isRetrieved[other.Path]
		if isHit {
			base.Relevant = append(base.Relevant, doc)
			continue
		}

		relevantBlocks := pull.copyRelevantBlocks(other.Path)
		doc = Document{
			ID:          len(a.validator.blockDisplay),
			Path:        other.Path,
			IsHit:       isHit,
			Retrieved:   nil,
			Relevant:    relevantBlocks,
			Stats:       lineStats(nil, relevantBlocks),
			Probability: 0,
		}
		a.validator.blockDisplay = append(a.validator.blockDisplay, doc)
		base.Relevant = append(base.Relevant, doc)
	}
	base.Stats = validation.Count(base.relevant, base.retrieved)
	return base, nil
}

func (a *App) addRec(rec recommend.File, pull Pull) Document {
	retrievedBlocks := processBlocks(rec.Blocks)
	relevantBlocks := pull.copyRelevantBlocks(rec.Path)
	isRetrievedLine := make(map[string]bool)
	isHit := make(map[string]bool)
	for _, block := range retrievedBlocks {
		for _, line := range block.Lines {
			isRetrievedLine[line.Number] = true
		}
	}
	for i, block := range relevantBlocks {
		for j, line := range block.Lines {
			if !isRetrievedLine[line.Number] {
				continue
			}
			isHit[line.Number] = true
			relevantBlocks[i].Lines[j].IsHit = true
		}
	}
	for i, block := range retrievedBlocks {
		for j, line := range block.Lines {
			retrievedBlocks[i].Lines[j].IsHit = isHit[line.Number]
		}
	}

	doc := Document{
		ID:          len(a.validator.blockDisplay),
		Path:        rec.Path,
		IsHit:       pull.isRelevant[rec.Path],
		Retrieved:   retrievedBlocks,
		Relevant:    relevantBlocks,
		Stats:       lineStats(retrievedBlocks, relevantBlocks),
		Probability: rec.Probability,
	}
	a.validator.blockDisplay = append(a.validator.blockDisplay, doc)
	return doc
}

func (a *App) maybeComputeClosed(pullID int) error {
	if _, ok := a.validator.fileDisplay[pullID]; ok {
		return nil
	}
	ignoreOpts := ignore.Options{
		Root:            localpath.Absolute(a.paths.Root),
		IgnoreFilenames: []localpath.Relative{ignore.GitIgnoreFilename},
	}
	ignorer, err := ignore.New(ignoreOpts)
	if err != nil {
		return err
	}
	l := validation.Loader{
		PullsPath: localpath.Absolute(a.paths.ClosedPullsPath),
		Root:      localpath.Absolute(a.paths.Root),
		Ignorer:   ignorer,
	}
	closedPulls, err := l.Load()
	if err != nil {
		return err
	}
	pullNumber := strconv.Itoa(pullID)
	files, ok := closedPulls[validation.PullRequestID(pullNumber)]
	if !ok {
		return errors.New("pull number not found")
	}
	err = a.addPull(pullNumber, files)
	if err != nil {
		return err
	}
	a.validator.fileDisplay[pullID] = a.validator.Pulls[len(a.validator.Pulls)-1]
	return nil
}

func (p Pull) copyRelevantBlocks(path string) []Block {
	lines := p.pathRelevantBlocks[path]
	copied := make([]Block, len(lines))
	for i, chunk := range lines {
		copied[i] = Block{Lines: make([]Line, len(chunk.Lines))}
		copy(copied[i].Lines, chunk.Lines)
	}
	return copied
}

func processBlocks(blocks []recommend.Block) []Block {
	var processed []Block
	for _, block := range blocks {
		var lines []Line
		number := block.FirstLine
		for _, content := range strings.Split(block.Content, "\n") {
			content = strings.Replace(content, "\t", "  ", -1)
			lineCutoff := 95
			replacement := "..."
			if len(content) > lineCutoff {
				content = content[:lineCutoff] + replacement
			}
			lines = append(lines, Line{
				Number:  strconv.Itoa(number),
				Content: content,
			})
			number++
		}
		processed = append(processed, Block{
			Lines:       lines,
			Probability: block.Probability,
		})
	}
	return processed
}

func lineStats(retrieved, relevant []Block) validation.Stats {
	var retrievedNumbers, relevantNumbers []string
	for _, block := range retrieved {
		for _, line := range block.Lines {
			retrievedNumbers = append(retrievedNumbers, line.Number)
		}
	}
	for _, block := range relevant {
		for _, line := range block.Lines {
			relevantNumbers = append(relevantNumbers, line.Number)
		}
	}
	return validation.Count(relevantNumbers, retrievedNumbers)
}

func (s *Server) runOpen(w http.ResponseWriter, r *http.Request) {
	err := s.templates.Render(w, "validate.html", s.app.validator)
	if err != nil {
		s.showError(w, err)
	}
	return
}

func (s *Server) runFiles(w http.ResponseWriter, r *http.Request) {
	id, err := getID(r)
	if err != nil {
		s.showError(w, err)
		return
	}
	err = s.app.maybeComputeClosed(id)
	if err != nil {
		s.showError(w, err)
		return
	}
	if _, ok := s.app.validator.fileDisplay[id]; !ok {
		s.showError(w, errors.New("file display for id not found"))
		return
	}
	err = s.templates.Render(w, "validate-files.html", s.app.validator.fileDisplay[id])
	if err != nil {
		s.showError(w, err)
	}
}

func (s *Server) runBlocks(w http.ResponseWriter, r *http.Request) {
	id, err := getID(r)
	if err != nil {
		s.showError(w, err)
		return
	}
	if id < 0 || id >= len(s.app.validator.blockDisplay) {
		s.showError(w, errors.New("block display for id not found"))
		return
	}
	err = s.templates.Render(w, "validate-blocks.html", s.app.validator.blockDisplay[id])
	if err != nil {
		s.showError(w, err)
	}
}
