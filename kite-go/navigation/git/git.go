package git

import (
	"context"
	"errors"
	"io"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// ErrDoneIterating ...
var ErrDoneIterating = errors.New("done iterating")

const (
	// DefaultComputedCommitsLimit ...
	DefaultComputedCommitsLimit = 1e5

	commitPause           = 20 * time.Millisecond
	commitDiffTimeout     = time.Second
	defaultMaxStorageSize = 1e7 // 10 MB
	storagePermissions    = 0600
)

// Repo ...
type Repo struct {
	commitIter           object.CommitIter
	computedCommitsLimit int
	computedCommits      int
	hit                  bool
	oldCache             repoCache
	newCache             repoCache
	key                  repoKey
	m                    *sync.Mutex
}

// CommitHash ...
type CommitHash string

// Commit ...
type Commit struct {
	Hash  CommitHash
	Files []File
}

// File is a slash-separated path, relative to the repo root.
type File string

// ToLocalFile ...
func (f File) ToLocalFile(root localpath.Absolute) localpath.Absolute {
	return root.Join(localpath.Relative(filepath.FromSlash(string(f))))
}

// HasSupportedExtension ...
func (f File) HasSupportedExtension() bool {
	return localpath.Extension(path.Ext(string(f))).IsSupported()
}

// Dir ...
func (f File) Dir() File {
	return File(path.Dir(string(f)))
}

// Open ...
func Open(root localpath.Absolute, computedCommitsLimit int, s Storage) (Repo, error) {
	key := repoKey(root.Clean())
	r, err := git.PlainOpen(string(root))
	if err != nil {
		return Repo{}, err
	}
	opts := git.LogOptions{
		Order: git.LogOrderCommitterTime,
	}
	commitIter, err := r.Log(&opts)
	if err != nil {
		return Repo{}, err
	}

	s.lock()
	defer s.unlock()

	bundle, err := readFromStorage(s, defaultMaxStorageSize)
	if err != nil {
		return Repo{}, err
	}
	oldCache, hit := bundle.get(key)
	repo := Repo{
		commitIter:           commitIter,
		computedCommitsLimit: computedCommitsLimit,
		computedCommits:      0,
		hit:                  hit,
		oldCache:             oldCache,
		newCache:             newRepoCache(),
		key:                  key,
		m:                    new(sync.Mutex),
	}
	return repo, nil
}

// Save updates the cache for the repo, so that the commits
// in the cache are the ones accessed by calls to `Next`.
func (r Repo) Save(s Storage) error {
	s.lock()
	defer s.unlock()

	bundle, err := readFromStorage(s, defaultMaxStorageSize)
	if err != nil {
		return err
	}
	bundle.add(r.key, r.newCache)
	data, evict, err := bundle.evictAndMarshal(defaultMaxStorageSize)
	if err != nil {
		return err
	}
	h := hitEvict{
		hit:   r.hit,
		evict: evict,
	}
	bundle.logCacheMetrics(len(data), h)
	return s.write(data, defaultMaxStorageSize)
}

// Next ...
func (r *Repo) Next(ctx kitectx.Context) (Commit, error) {
	r.m.Lock()
	defer r.m.Unlock()

	commit, err := r.commitIter.Next()
	if err == io.EOF || err == plumbing.ErrObjectNotFound {
		return Commit{}, ErrDoneIterating
	}
	if err != nil {
		return Commit{}, err
	}
	next, err := r.getCommit(ctx, commit)
	if err != nil {
		return Commit{}, err
	}
	r.newCache.add(getHash(commit), next)
	return next, nil
}

func (r *Repo) getCommit(ctx kitectx.Context, commit *object.Commit) (Commit, error) {
	fromCache, ok, err := r.oldCache.get(getHash(commit))
	if err != nil {
		return Commit{}, err
	}
	if ok {
		return fromCache, nil
	}

	if r.computedCommits >= r.computedCommitsLimit {
		return Commit{}, ErrDoneIterating
	}
	r.computedCommits++

	processed := Commit{
		Hash:  getHash(commit),
		Files: processCommitFiles(ctx, commit),
	}
	return processed, nil
}

func getHash(commit *object.Commit) CommitHash {
	return CommitHash(commit.Hash.String())
}

func processCommitFiles(ctx kitectx.Context, commit *object.Commit) []File {
	if commit.NumParents() != 1 {
		return nil
	}
	parent, err := commit.Parents().Next()
	if err != nil {
		return nil
	}

	parentTree, err := parent.Tree()
	if err != nil {
		return nil
	}
	commitTree, err := commit.Tree()
	if err != nil {
		return nil
	}

	time.Sleep(commitPause)
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Context(), commitDiffTimeout)
	defer cancel()
	changes, err := object.DiffTreeContext(ctxWithTimeout, parentTree, commitTree)
	if err != nil {
		return nil
	}

	var files []File
	for _, change := range changes {
		file := File(change.To.Name)
		if file == "" {
			// the change is a delete, so we skip it.
			continue
		}
		if !file.HasSupportedExtension() {
			continue
		}
		files = append(files, file)
	}
	return files
}
