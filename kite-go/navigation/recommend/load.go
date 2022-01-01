package recommend

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

func (r recommender) loadGraph(ctx kitectx.Context) (graph, error) {
	fileEdits, numEdits, err := r.getCommits(ctx)
	if err != nil {
		return graph{}, err
	}
	g := graph{
		files:    make(map[fileID][]commitID, len(fileEdits)),
		editSize: make(map[commitID]uint32, numEdits),
		opts:     defaultGraphOptions,
	}
	for file, edits := range fileEdits {
		f, err := r.fileIndex.toID(file.ToLocalFile(r.opts.Root))
		if err != nil {
			return graph{}, err
		}
		// We copy edits into a slice with precise capacity to use less memory (after GC).
		// This also allows fileEdits to be garbage collected.
		g.files[f] = make([]commitID, len(edits))
		for _, edit := range edits {
			g.editSize[edit]++
		}
		copy(g.files[f], edits)
	}
	g.computeEditScores()
	return g, nil
}

// The returned map associates commits with modified files.
func (r recommender) getCommits(ctx kitectx.Context) (map[git.File][]commitID, int, error) {
	repo, err := git.Open(r.opts.Root, r.opts.ComputedCommitsLimit, r.gitStorage)
	if err != nil {
		return nil, 0, err
	}

	files := make(map[git.File][]commitID)
	var numFiles, numEdits int
	for commit, err := repo.Next(ctx); err != git.ErrDoneIterating; commit, err = repo.Next(ctx) {
		if err != nil {
			return nil, 0, err
		}
		ctx.CheckAbort()
		if len(commit.Files) <= 1 {
			continue
		}
		for _, file := range commit.Files {
			if _, ok := files[file]; !ok {
				numFiles++
			}
		}
		numEdits++
		if numFiles*numEdits > r.params.maxMatrixSize {
			break
		}
		for _, file := range commit.Files {
			files[file] = append(files[file], commitID(numEdits))
		}
	}

	if err := repo.Save(r.gitStorage); err != nil {
		return nil, 0, err
	}
	return files, numEdits, nil
}

// Assumes that parent directories have already been checked and can be used.
func (r recommender) canUseDir(path localpath.Absolute) bool {
	return r.canUse(path, true, 0)
}

// Assumes that parent directories have already been checked and can be used.
func (r recommender) canUseFile(path localpath.Absolute, size int64) bool {
	return r.canUse(path, false, size)
}

// Assumes that parent directories have already been checked and can be used.
func (r recommender) canUse(path localpath.Absolute, isDir bool, size int64) bool {
	if r.ignorer.Ignore(path, isDir) {
		return false
	}
	if isDir {
		return true
	}
	if !path.HasSupportedExtension() {
		return false
	}
	return size <= r.opts.MaxFileSize
}

// ErrOpenedTooManyFiles ...
var ErrOpenedTooManyFiles = errors.New("opened too many files")

func (r *recommender) loadVectorizer(ctx kitectx.Context) error {
	var validFiles []localpath.Absolute
	err := filepath.Walk(string(r.opts.Root), func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		ctx.CheckAbort()
		path, err := localpath.NewAbsolute(p)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if !r.canUseDir(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !r.canUseFile(path, info.Size()) {
			return nil
		}
		if len(validFiles) >= r.opts.MaxFiles {
			return ErrOpenedTooManyFiles
		}
		validFiles = append(validFiles, path)
		return nil
	})
	if err != nil {
		return err
	}

	c := newCounter(r.opts.KeepUnderscores)
	for _, path := range validFiles {
		ctx.CheckAbort()
		contents, err := r.read(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		c.add(string(contents))
	}

	r.vectorizer = c.newVectorizer()
	info, err := r.opts.Root.Lstat()
	if err != nil {
		return err
	}
	if !r.canUseDir(r.opts.Root) {
		return errors.New("cannot use Root")
	}
	r.vectorizer.watchDirs.data[r.opts.Root] = info.ModTime().Add(-time.Second)
	_, err = r.refreshVectorSet(ctx)
	return err
}

func (r recommender) read(path localpath.Absolute) ([]byte, error) {
	reader, err := r.fileOpener.open(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return ioutil.ReadAll(io.LimitReader(reader, r.opts.MaxFileSize))
}

func (r recommender) refreshVectorSet(ctx kitectx.Context) (int, error) {
	changes, err := r.computeVectorSetChanges(ctx)
	if err != nil {
		return 0, err
	}
	r.vectorizer.vectorSet.update(changes)
	return len(changes.updates), nil
}

func (r recommender) computeVectorSetChanges(ctx kitectx.Context) (vectorSetChanges, error) {
	// It would also be possible to compute the `vectorSetChanges` with just one
	// loop over `watchDirs`, without a loop over `vectorSet`.
	// But this would involve reading every directory in `watchDirs`.
	// The benchmarks in `recommend_test.go` show this is noticeably slower
	// in the case where the project is not modified at all.
	// In production we expect that there will only be a few modifications
	// each time `computeVectorSetChanges` is called.
	// So we expect this approach would generally be slower in production.

	r.vectorizer.watchDirs.m.Lock()
	defer r.vectorizer.watchDirs.m.Unlock()

	r.vectorizer.vectorSet.m.RLock()
	defer r.vectorizer.vectorSet.m.RUnlock()

	changes := newVectorSetChanges()
	for path := range r.vectorizer.watchDirs.data {
		ctx.CheckAbort()
		info, err := path.Lstat()
		if os.IsNotExist(err) {
			delete(r.vectorizer.watchDirs.data, path)
			continue
		}
		if err != nil {
			return vectorSetChanges{}, err
		}
		if !info.IsDir() {
			delete(r.vectorizer.watchDirs.data, path)
			continue
		}
		if r.vectorizer.watchDirs.data[path] == info.ModTime() {
			continue
		}
		if !r.canUseDir(path) {
			delete(r.vectorizer.watchDirs.data, path)
			continue
		}
		r.vectorizer.watchDirs.data[path] = info.ModTime()
		batch, err := r.refreshDir(ctx, path)
		if err != nil {
			return vectorSetChanges{}, err
		}
		changes.add(batch)
	}

	for pathID := range r.vectorizer.vectorSet.data {
		path, err := r.fileIndex.fromID(pathID)
		if err != nil {
			return vectorSetChanges{}, err
		}
		info, err := path.Lstat()
		if os.IsNotExist(err) {
			changes.deletes = append(changes.deletes, pathID)
			continue
		}
		if err != nil {
			return vectorSetChanges{}, err
		}
		if info.IsDir() {
			changes.deletes = append(changes.deletes, pathID)
			continue
		}
		if r.vectorizer.vectorSet.data[pathID].modTime == info.ModTime() {
			continue
		}
		if changes.updates[pathID].modTime == info.ModTime() {
			continue
		}
		if !r.canUseFile(path, info.Size()) {
			changes.deletes = append(changes.deletes, pathID)
			continue
		}
		contents, err := r.read(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return vectorSetChanges{}, err
		}
		if len(contents) == 0 {
			continue
		}
		vector := r.vectorizer.makeVector(string(contents))
		changes.updates[pathID] = shingleVector{
			coords:  vector.coords,
			norm:    vector.norm,
			modTime: info.ModTime(),
		}
	}
	return changes, nil
}

func (r recommender) refreshDir(ctx kitectx.Context, dirPath localpath.Absolute) (vectorSetChanges, error) {
	ctx.CheckAbort()
	children, err := dirPath.Readdirnames(-1)
	if os.IsNotExist(err) {
		delete(r.vectorizer.watchDirs.data, dirPath)
		return vectorSetChanges{}, nil
	}
	if err != nil {
		return vectorSetChanges{}, err
	}

	changes := newVectorSetChanges()
	for _, child := range children {
		childPath := dirPath.Join(child)
		info, err := childPath.Lstat()
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return vectorSetChanges{}, err
		}
		if info.IsDir() {
			if r.vectorizer.watchDirs.data[childPath] == info.ModTime() {
				continue
			}
			if !r.canUseDir(childPath) {
				delete(r.vectorizer.watchDirs.data, childPath)
				continue
			}
			r.vectorizer.watchDirs.data[childPath] = info.ModTime()
			batch, err := r.refreshDir(ctx, childPath)
			if err != nil {
				return vectorSetChanges{}, err
			}
			changes.add(batch)
			continue
		}
		if !r.canUseFile(childPath, info.Size()) {
			continue
		}
		childPathID, err := r.fileIndex.toID(childPath)
		if err != nil {
			return vectorSetChanges{}, err
		}
		if r.vectorizer.vectorSet.data[childPathID].modTime == info.ModTime() {
			continue
		}
		contents, err := r.read(childPath)
		if os.IsNotExist(err) {
			changes.deletes = append(changes.deletes, childPathID)
			continue
		}
		if err != nil {
			return vectorSetChanges{}, err
		}
		vector := r.vectorizer.makeVector(string(contents))
		changes.updates[childPathID] = shingleVector{
			coords:  vector.coords,
			norm:    vector.norm,
			modTime: info.ModTime(),
		}
	}
	return changes, nil
}

type fileOpener struct {
	// We open many files twice while building a recommender,
	// because a recommender is built with two passes over the code base.
	// In the first pass, we learn how common words are.
	// In the second pass, we cache a vector for each file.
	// We want to return the number of unique files opened,
	// so we use a map to count the number of unique files opened.
	counter map[localpath.Absolute]struct{}
	max     int
	prev    time.Time
	rate    time.Duration
	m       sync.Mutex
}

func (r recommender) newFileOpener() *fileOpener {
	var rate time.Duration
	if r.params.maxFileOpensPerSecond != 0 {
		rate = time.Second / time.Duration(r.params.maxFileOpensPerSecond)
	}
	return &fileOpener{
		counter: make(map[localpath.Absolute]struct{}),
		prev:    time.Now(),
		max:     r.opts.MaxFiles,
		rate:    rate,
	}
}

func (f *fileOpener) open(path localpath.Absolute) (*os.File, error) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.counter != nil {
		f.counter[path] = struct{}{}
		if len(f.counter) > f.max {
			return nil, ErrOpenedTooManyFiles
		}
	}

	time.Sleep(time.Until(f.prev.Add(f.rate)))
	f.prev = time.Now()
	return path.Open()
}

func (f *fileOpener) counterSize() int {
	f.m.Lock()
	defer f.m.Unlock()

	return len(f.counter)
}

func (f *fileOpener) releaseMax() {
	f.m.Lock()
	defer f.m.Unlock()

	f.counter = nil
}
