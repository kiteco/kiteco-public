package data

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
)

// DiffOperation ...
type DiffOperation diff.Operation

// String ...
func (d DiffOperation) String() string {
	switch diff.Operation(d) {
	case diff.Equal:
		return "equal"
	case diff.Add:
		return "add"
	case diff.Delete:
		return "delete"
	default:
		panic(fmt.Sprintf("unsupported diff.Operation %v", diff.Operation(d)))
	}
}

// GitDiffChunk is a serializable form of
// go-git/plumbing/format/diff/Chunk, that represents
// a portion of a file transformation.
type GitDiffChunk struct {
	// Content contains the portion of the file
	Content string
	// Type of operation to do with this chunk
	Type DiffOperation
}

// GitCommitFile groups the changes to a commit file.
type GitCommitFile struct {
	// TODO: support file renames
	Path   string
	Chunks []GitDiffChunk
}

// GitCommit groups a commit message along with the files modified as part of the commit
// this data comes from the raw git repo + go-git.
type GitCommit struct {
	RepoOwner string
	RepoName  string
	Message   string
	Files     []GitCommitFile
}

// SampleTag implements pipeline.Sample
func (GitCommit) SampleTag() {}

// GitCommits ...
type GitCommits []GitCommit

// SampleTag implements pipeline.Sample
func (GitCommits) SampleTag() {}

// NewGitCommitsSource is a source that returns `GitCommit`s as the sample,
// SEE: kiteco/local-pipelines/summarize/Makefile.datasets for more details
func NewGitCommitsSource(opts source.DatasetOpts, name string, dts ...DatasetType) (pipeline.Source, error) {
	fs, err := Datasets(dts...)
	if err != nil {
		return nil, err
	}
	return source.NewDataset(opts, name, source.JSONProcessFn(GitCommit{}), fs...), nil
}

// GitCommitIter is an iterator over commits, it returns
// a GitCommit and a boolean indicating if the result is ok to use,
// once the iterator returns (GitCommit{}, false) it will always return
// (GitCommit{}, false).
type GitCommitIter func() (GitCommit, bool)

// NewGitCommitIter returns an iterator over the provided GitCommit datasets
func NewGitCommitIter(dts ...DatasetType) (GitCommitIter, error) {
	return NewGitCommitIterWithOpts(source.DatasetOpts{
		NumGo:        1,
		Epochs:       1,
		PanicOnError: true,
	}, dts...)
}

// NewGitCommitIterWithOpts returns an iterator over the provided GitCommit datasets
func NewGitCommitIterWithOpts(opts source.DatasetOpts, dts ...DatasetType) (GitCommitIter, error) {
	src, err := NewGitCommitsSource(opts, "git-commits-iter", dts...)
	if err != nil {
		return nil, err
	}

	src, err = src.ForShard(0, 1)
	if err != nil {
		return nil, err
	}

	return func() (GitCommit, bool) {
		if rec := src.SourceOut(); rec != (pipeline.Record{}) {
			return rec.Value.(GitCommit), true
		}
		return GitCommit{}, false
	}, nil
}
