package githubcorpus

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"

	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"

	"github.com/google/go-github/github"
)

var (
	// ErrStoppedEarly is the error returned if ScanRepo returns before
	// fetching the number of pull requests specified via options. e.g
	// the PullRequestFunc returns false.
	ErrStoppedEarly = errors.New("stopped repo early")
)

// Corpus ...
type Corpus interface {
	// ScanRepo takes owner, repo and a PullRequestFunc to iterater over PR in owner/repo
	ScanRepo(string, string, PullRequestFunc) error
}

// PullRequestBundle ...
type PullRequestBundle struct {
	Owner       string
	Repo        string
	PullRequest *github.PullRequest

	// NOTE: these are only populated if APIScanOptions.IncludeCommitFiles
	CommitFiles []*github.CommitFile

	// NOTE: these are only populated if APIScanOptions.IncludeCommits, additionally
	// the Stats and Files fields are not populated, clients must call
	// Contents.GetCommit to get this information.
	Commits []*github.RepositoryCommit

	// DataFiles is a map from filename-sha to contents. This is only populated
	// for the PullRequestCorpus implementation.
	// - For a CommitFile dataset it stores the base and head
	// versions of the files listed in CommitFiles.
	// - For a Commit dataset it stores the base versions of the files for each file
	// in a commit.
	// - For a full dataset it stores both of the above.
	DataFiles map[string][]byte
}

// PullRequestFunc ...
type PullRequestFunc func(bundle PullRequestBundle, contents Contents) bool

// PullRequestCorpus will scan over PullRequestBundles located at the provided root
type PullRequestCorpus struct {
	root    string
	bundles []string
}

// NewPullRequestCorpus ...
func NewPullRequestCorpus(root string) (*PullRequestCorpus, error) {
	bundles, err := fileutil.ListDir(root)
	if err != nil {
		return nil, err
	}
	return &PullRequestCorpus{
		root:    root,
		bundles: bundles,
	}, nil
}

// NewPullRequestCorpusSingleBundle ...
func NewPullRequestCorpusSingleBundle(bundle string) (*PullRequestCorpus, error) {
	return &PullRequestCorpus{
		root:    fileutil.Dir(bundle),
		bundles: []string{bundle},
	}, nil
}

// Scan will apply the PullRequestFunc across all repos found at the root of the corpus
func (l *PullRequestCorpus) Scan(f PullRequestFunc) error {
	for _, bundle := range l.bundles {
		err := l.readBundle(bundle, f)
		if err != nil {
			return err
		}
	}

	return nil
}

// ScanRepo implements Corpus
func (l *PullRequestCorpus) ScanRepo(owner, name string, f PullRequestFunc) error {
	bundle := PRCorpusFilename(owner, name)
	err := l.readBundle(fileutil.Join(l.root, bundle), f)
	if err != nil {
		return err
	}
	return nil
}

// --

func (l *PullRequestCorpus) readBundle(bundle string, f PullRequestFunc) error {
	r, err := newJSONGzReader(bundle)
	if err != nil {
		return err
	}
	defer r.Close()

	for {
		var prb PullRequestBundle
		err = r.Next(&prb)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		next := f(prb, BundleContents{prb.DataFiles})
		if !next {
			return ErrStoppedEarly
		}
	}
	return nil

}

// --

var errNotFound = errors.New("contents not found")

// BundleContents ...
type BundleContents struct {
	DataFiles map[string][]byte
}

// GetFile ...
func (b BundleContents) GetFile(sha string, fn string) ([]byte, error) {
	key := contentsKey(sha, fn)
	buf, ok := b.DataFiles[key]
	if !ok {
		return nil, errNotFound
	}
	return buf, nil
}

// GetCommit ...
func (b BundleContents) GetCommit(sha string) (*github.RepositoryCommit, error) {
	panic(fmt.Sprintf("TODO: get commit not implemented for BundleContents"))
}

// --

type jsonGzReader struct {
	r  io.ReadCloser
	gz *gzip.Reader
	js *json.Decoder
}

func (j *jsonGzReader) Next(obj interface{}) error {
	return j.js.Decode(obj)
}

func (j *jsonGzReader) Close() error {
	j.gz.Close()
	j.r.Close()
	return nil
}

func newJSONGzReader(path string) (*jsonGzReader, error) {
	r, err := fileutil.NewCachedReader(path)
	if err != nil {
		return nil, err
	}
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	js := json.NewDecoder(gz)
	return &jsonGzReader{
		r:  r,
		gz: gz,
		js: js,
	}, nil
}

// --

func contentsKey(sha, fn string) string {
	return fmt.Sprintf("%s-%x", sha, spooky.Hash32([]byte(fn)))
}

// --

// PRCorpusFilename returns the filename for the provided repo owner/name
func PRCorpusFilename(owner, name string) string {
	return fmt.Sprintf("%s_%s.json.gz", owner, name)
}

// BaseAndHeadSHA returns the base and head/merge SHA of the pull request
func BaseAndHeadSHA(pull *github.PullRequest) (string, string) {
	baseSHA := pull.GetBase().GetSHA()
	headSHA := pull.GetHead().GetSHA()
	if mergeSHA := pull.GetMergeCommitSHA(); mergeSHA != "" {
		return baseSHA, mergeSHA
	}
	return baseSHA, headSHA
}
