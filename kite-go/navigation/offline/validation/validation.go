package validation

import (
	"errors"
	"hash/crc64"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Repo represents a repo in the validation set
type Repo struct {
	Owner string
	Name  string
}

// ReadRepos lists the repos in the validation set
func ReadRepos(path string) ([]Repo, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var repos []Repo
	for _, line := range strings.Split(string(content), "\n") {
		parts := strings.Split(line, "/")
		if len(parts) != 2 {
			continue
		}
		repo := Repo{
			Owner: parts[0],
			Name:  parts[1],
		}
		repos = append(repos, repo)
	}
	return repos, nil
}

// Options for validation
type Options struct {
	UseCommits           bool
	ComputedCommitsLimit int
	KeepUnderscores      bool
	SkipLines            bool
	PullsPath            localpath.Absolute
	Root                 localpath.Absolute
	MaxFileRecs          int
	MaxBlockRecs         int
	IgnoreFilenames      []localpath.Relative
	IgnorePatterns       []string
	Storage              git.Storage
}

// Validate computes stats for files and lines
func Validate(opts Options) (Stats, Stats, []Record) {
	ignoreOpts := ignore.Options{
		Root:            opts.Root,
		IgnoreFilenames: opts.IgnoreFilenames,
		IgnorePatterns:  opts.IgnorePatterns,
	}
	ignorer, err := ignore.New(ignoreOpts)
	if err != nil {
		log.Fatal(err)
	}

	recOpts := recommend.Options{
		UseCommits:           opts.UseCommits,
		ComputedCommitsLimit: opts.ComputedCommitsLimit,
		KeepUnderscores:      opts.KeepUnderscores,
		Root:                 opts.Root,
		MaxFileSize:          1e6,
		MaxFiles:             1e5,
	}

	recommender, err := recommend.NewRecommender(kitectx.Background(), recOpts, ignorer, opts.Storage)
	if err != nil {
		log.Fatal(err)
	}

	l := Loader{
		PullsPath: opts.PullsPath,
		Root:      opts.Root,
		Ignorer:   ignorer,
	}
	validationData, err := l.Load()
	if err != nil {
		log.Fatal(err)
	}

	validator := evaluator{
		recommender:    recommender,
		validationData: validationData,
		opts:           opts,
	}

	var wg sync.WaitGroup
	wg.Add(2)

	var fileStats Stats
	var records []Record
	go func() {
		defer wg.Done()
		fileStats, records = validator.evaluateFiles()
	}()

	var lineStats Stats
	go func() {
		defer wg.Done()
		if opts.SkipLines {
			return
		}
		var err error
		lineStats, err = validator.evaluateLines()
		if err != nil {
			log.Fatal(err)
		}
	}()

	wg.Wait()

	return fileStats, lineStats, records
}

// Strings formats stats
func (s Stats) Strings() []string {
	return []string{
		strconv.FormatFloat(s.F1, 'g', 4, 64),
		strconv.FormatFloat(s.Precision, 'g', 4, 64),
		strconv.FormatFloat(s.Recall, 'g', 4, 64),
	}
}

// Stats holds computed metrics
type Stats struct {
	Precision float64
	Recall    float64
	F1        float64
}

// Mean aggregates stats, panics if given slice is empty
func Mean(s []Stats) Stats {
	var total Stats
	for _, sample := range s {
		total.add(sample)
	}
	total.divide(len(s))
	return total
}

func (s *Stats) add(other Stats) {
	s.Precision += other.Precision
	s.Recall += other.Recall
	s.F1 += other.F1
}

func (s *Stats) divide(N int) {
	s.Precision /= float64(N)
	s.Recall /= float64(N)
	s.F1 /= float64(N)
}

type evaluator struct {
	recommender    recommend.Recommender
	validationData map[PullRequestID][]recommend.File
	opts           Options
}

// Record holds unaggregated data
type Record struct {
	RepoOwner   string
	RepoName    string
	Base        string
	Recommended string
	Score       float64
	IsRelevant  bool
}

func (e evaluator) evaluateFiles() (Stats, []Record) {
	var raw []Stats
	var records []Record
	for _, files := range e.validationData {
		if len(files) == 1 {
			continue
		}

		isRelevant := make(map[string]bool)
		for _, file := range files {
			isRelevant[file.Path] = true
		}

		var pullStats []Stats
		for i, current := range files {
			request := recommend.Request{
				Location: recommend.Location{
					CurrentPath: current.Path,
					CurrentLine: PickLine(current),
				},
				MaxFileRecs:  e.opts.MaxFileRecs,
				MaxBlockRecs: 0,
			}
			retrieved, err := e.recommender.Recommend(kitectx.Background(), request)
			if err != nil {
				continue
			}

			var retrievedFiles []string
			for _, file := range retrieved {
				retrievedFiles = append(retrievedFiles, file.Path)
				records = append(records, Record{
					Base:        current.Path,
					Recommended: file.Path,
					Score:       file.Probability,
					IsRelevant:  isRelevant[file.Path],
				})
			}

			var relevantFiles []string
			for j, file := range files {
				if j == i {
					continue
				}
				relevantFiles = append(relevantFiles, file.Path)
			}

			pullStats = append(pullStats, Count(relevantFiles, retrievedFiles))
		}

		if len(pullStats) == 0 {
			continue
		}
		raw = append(raw, Mean(pullStats))
	}

	return Mean(raw), records
}

func (e evaluator) evaluateLines() (Stats, error) {
	var raw []Stats
	for _, files := range e.validationData {
		if len(files) == 1 {
			continue
		}

		var pullStats []Stats
		for i, current := range files {
			currentLine := PickLine(current)
			for j, other := range files {
				if j == i {
					continue
				}

				relevantLines, err := getLines(other.Blocks)
				if err != nil {
					return Stats{}, err
				}

				request := recommend.BlockRequest{
					Request: recommend.Request{
						MaxFileRecs:  e.opts.MaxFileRecs,
						MaxBlockRecs: e.opts.MaxBlockRecs,
						Location: recommend.Location{
							CurrentPath: current.Path,
							CurrentLine: currentLine,
						},
					},
					InspectFiles: []recommend.File{{Path: other.Path}},
				}

				retrieved, err := e.recommender.RecommendBlocks(kitectx.Background(), request)
				if err == recommend.ErrInvalidCurrentLine {
					continue
				}
				if err != nil {
					return Stats{}, err
				}
				retrievedBlocks := retrieved[0].Blocks
				retrievedLines, err := getLines(retrievedBlocks)
				if err != nil {
					return Stats{}, err
				}

				pullStats = append(pullStats, Count(relevantLines, retrievedLines))
			}
		}

		if len(pullStats) == 0 {
			continue
		}
		raw = append(raw, Mean(pullStats))
	}
	return Mean(raw), nil
}

// PickLine returns a line number from a block in Blocks
func PickLine(file recommend.File) int {
	var choices []int
	for _, block := range file.Blocks {
		for line := block.FirstLine; line <= block.LastLine; line++ {
			choices = append(choices, line)
		}
	}

	if len(choices) == 0 {
		return 0
	}
	sort.Ints(choices)
	table := crc64.MakeTable(crc64.ECMA)
	idx := crc64.Checksum([]byte(file.Path), table) % uint64(len(choices))
	return choices[idx]
}

var errOverlappingBlocks = errors.New("overlapping blocks")

func getLines(blocks []recommend.Block) ([]string, error) {
	var lines []string
	lineSet := make(map[string]bool)
	for _, block := range blocks {
		for i := block.FirstLine; i <= block.LastLine; i++ {
			line := strconv.Itoa(i)
			if lineSet[line] {
				return nil, errOverlappingBlocks
			}
			lines = append(lines, line)
			lineSet[line] = true
		}
	}
	return lines, nil
}

// Count computes stats given relevant and retrieved items
func Count(relevant, retrieved []string) Stats {
	if len(relevant) == 0 {
		return Stats{}
	}
	if len(retrieved) == 0 {
		return Stats{}
	}
	isRelevant := make(map[string]bool)
	for _, val := range relevant {
		isRelevant[val] = true
	}
	var hits int
	for _, document := range retrieved {
		if isRelevant[document] {
			hits++
		}
	}
	precision := float64(hits) / float64(len(retrieved))
	recall := float64(hits) / float64(len(relevant))
	var f1 float64
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}
	return Stats{
		Precision: precision,
		Recall:    recall,
		F1:        f1,
	}
}
