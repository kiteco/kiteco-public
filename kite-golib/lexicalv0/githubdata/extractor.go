package githubdata

import (
	"errors"
	"fmt"

	"github.com/google/go-github/github"
	"github.com/kiteco/kiteco/kite-golib/githubcorpus"
)

// Options ...
type Options struct {
	// Commit Data
	AllowedStatus map[string]bool // Possible values: added, modified
}

// Extractor ...
type Extractor struct {
	opts Options
}

// NewExtractor initializes a Extractor with certain options
func NewExtractor(opts Options) (Extractor, error) {
	return Extractor{
		opts: opts,
	}, nil
}

var errSkip = errors.New("skipped")

// ExtractPredictionSites ...
func (e Extractor) ExtractPredictionSites(
	pull *github.PullRequest, file *github.CommitFile, contents githubcorpus.Contents) (sites []PredictionSite, err error) {

	defer func() {
		if r := recover(); r != nil {
			sites = nil
			err = fmt.Errorf("panic: %+v", r)
		}
	}()

	if !e.opts.AllowedStatus[file.GetStatus()] {
		return nil, errSkip
	}
	if file.Patch == nil {
		return nil, errSkip
	}

	switch file.GetStatus() {
	case "modified":
		sites, err = e.fromModifiedFile(pull, file, contents)
		return
	case "added":
		sites, err = e.fromAddedFile(pull, file, contents)
		return
	}

	return nil, errSkip
}

func (e Extractor) fromModifiedFile(pull *github.PullRequest, file *github.CommitFile, contents githubcorpus.Contents) ([]PredictionSite, error) {
	filename := file.GetFilename()
	baseSHA, headSHA := githubcorpus.BaseAndHeadSHA(pull)

	oldBuf, err := contents.GetFile(baseSHA, filename)
	if err != nil {
		return nil, err
	}

	newBuf, err := contents.GetFile(headSHA, filename)
	if err != nil {
		return nil, err
	}

	oldRaw := string(oldBuf)
	newRaw := string(newBuf)

	diffs, err := computeDiffs(oldRaw, newRaw)
	if err != nil {
		return nil, err
	}

	sites := e.extractSamplesFromDiff(pull, filename, diffs)

	return sites, nil

}

func (e Extractor) fromAddedFile(pull *github.PullRequest, file *github.CommitFile, contents githubcorpus.Contents) ([]PredictionSite, error) {
	_, headSHA := githubcorpus.BaseAndHeadSHA(pull)
	buf, err := contents.GetFile(headSHA, file.GetFilename())
	if err != nil {
		return nil, err
	}

	raw := string(buf)

	site := PredictionSite{
		PullNumber: pull.GetNumber(),
		PullTime:   pull.GetClosedAt().Format("20060102150405"),
		FilePath:   file.GetFilename(),
		DstWindow:  raw,
	}

	return []PredictionSite{site}, nil
}
