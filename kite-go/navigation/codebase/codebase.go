package codebase

import (
	"errors"
	"io/ioutil"
	"log"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"

	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-golib/filters"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Errors exported for testing codenav API
var (
	ErrPathNotInSupportedProject   = errors.New("ErrPathNotInSupportedProject")
	ErrPathInFilteredDirectory     = errors.New("ErrPathInFilteredDirectory")
	ErrPathHasUnsupportedExtension = errors.New("ErrPathHasUnsupportedExtension")
	ErrProjectNotLoaded            = errors.New("ErrProjectNotLoaded")
	ErrProjectStillIndexing        = errors.New("ErrProjectStillIndexing")
	ErrProjectBuildFailed          = errors.New("ErrProjectBuildFailed")

	errWasTerminated = errors.New("navigator was terminated")
)

var defaultMaxProjects = 5

// Options ...
type Options struct {
	ComputedCommitsLimit int
	GitStorageOpts       git.StorageOptions
}

// Navigator finds related code within projects.
type Navigator struct {
	opts          Options
	isProjectRoot func(localpath.Absolute) (bool, error)
	load          loadFunc
	gitStorage    git.Storage
	projects      *lru.Cache
	m             *sync.Mutex
	indexing      chan struct{}
	term          terminator
}

// NewNavigator ...
func NewNavigator(opts Options) (Navigator, error) {
	cache, err := lru.New(defaultMaxProjects)
	if err != nil {
		return Navigator{}, err
	}
	s, err := git.NewStorage(opts.GitStorageOpts)
	if err != nil {
		return Navigator{}, err
	}
	return Navigator{
		opts:          opts,
		isProjectRoot: isProjectRoot,
		load:          load,
		gitStorage:    s,
		projects:      cache,
		m:             new(sync.Mutex),
		indexing:      make(chan struct{}, 1),
		term:          newTerminator(),
	}, nil
}

// Navigate finds related files in the project associated to the given path.
// It returns ErrShouldLoad, if the status is Inactive or the project should be rebuilt.
// In these cases, MaybeLoad should typically be called.
func (n *Navigator) Navigate(request recommend.Request) (FileIterator, error) {
	if request.SkipRefresh {
		return n.navigate(request)
	}
	select {
	case n.indexing <- struct{}{}:
		defer func() { <-n.indexing }()
		return n.navigate(request)
	default:
		request.SkipRefresh = true
		return n.navigate(request)
	}
}

func (n *Navigator) navigate(request recommend.Request) (FileIterator, error) {
	if n.term.wasTerminated() {
		return FileIterator{}, errWasTerminated
	}

	request.Location.CurrentPath = normalize(runtime.GOOS, request.Location.CurrentPath)
	abs, err := localpath.NewAbsolute(request.Location.CurrentPath)
	if err != nil {
		return FileIterator{}, err
	}
	err = blockPath(runtime.GOOS, abs)
	if err != nil {
		return FileIterator{}, err
	}

	project, err := n.getProjectNavigator(abs)
	if err != nil {
		return FileIterator{}, err
	}
	if request.SkipRefresh {
		return project.navigate(kitectx.Background(), request)
	}
	var iter FileIterator
	closure, cancel := n.term.closureWithCancel(func(ctx kitectx.Context) error {
		iter, err = project.navigate(ctx, request)
		return nil
	})
	defer cancel()
	closure()
	return iter, err
}

// MaybeLoad loads the project associated to the given path, if its status is Inactive.
func (n *Navigator) MaybeLoad(path string, maxFileSize int64, maxFiles int) {
	path = normalize(runtime.GOOS, path)
	abs, err := localpath.NewAbsolute(path)
	if err != nil {
		return
	}
	err = blockPath(runtime.GOOS, abs)
	if err != nil {
		return
	}

	n.indexing <- struct{}{}
	defer func() { <-n.indexing }()

	if n.term.wasTerminated() {
		return
	}

	project, err := n.getProjectNavigator(abs)
	if err != nil {
		return
	}
	closure, cancel := n.term.closureWithCancel(func(ctx kitectx.Context) error {
		project.maybeLoad(ctx, n.gitStorage, maxFileSize, maxFiles)
		return nil
	})
	defer cancel()
	closure()
}

// ProjectInfo returns the project status, root, and useful errors
func (n Navigator) ProjectInfo(path string) (ProjectStatus, string, error) {
	path = normalize(runtime.GOOS, path)
	abs, err := localpath.NewAbsolute(path)
	if err != nil {
		return "", "", err
	}
	err = blockPath(runtime.GOOS, abs)
	if err != nil {
		return "", "", err
	}

	n.m.Lock()
	defer n.m.Unlock()

	root, err := n.getProjectRoot(abs)
	if err != nil {
		return "", "", err
	}
	v, ok := n.projects.Peek(root)
	if !ok {
		return "", string(root), ErrProjectNotLoaded
	}
	return v.(*projectNavigator).getState().status, string(root), nil
}

// Terminate cancels all work and prevents new work from being started.
func (n *Navigator) Terminate() {
	n.term.terminate()

	n.m.Lock()
	defer n.m.Unlock()

	n.projects.Purge()
}

// WasTerminated returns if the navigator is terminated
func (n *Navigator) WasTerminated() bool {
	return n.term.wasTerminated()
}

// MaybeUnload unloads the projects that are stale
func (n *Navigator) MaybeUnload(unloadInterval time.Duration) {
	n.m.Lock()
	defer n.m.Unlock()

	for _, root := range n.projects.Keys() {
		v, ok := n.projects.Peek(root)
		if !ok {
			continue
		}
		p := v.(*projectNavigator)
		if p.getState().status != Active {
			continue
		}
		if lu := p.getState().lastUsed; !lu.IsZero() && time.Since(lu) >= unloadInterval {
			log.Println("unloading project for project root", root)
			n.projects.Remove(root)
		}
	}
}

func (n Navigator) getProjectNavigator(path localpath.Absolute) (*projectNavigator, error) {
	n.m.Lock()
	defer n.m.Unlock()

	root, err := n.getProjectRoot(path)
	if err != nil {
		return nil, err
	}
	n.maybeAddProject(root)
	v, ok := n.projects.Get(root)
	if !ok {
		return nil, ErrProjectNotLoaded
	}
	p := v.(*projectNavigator)
	p.updateLastUsed(time.Now())
	return p, nil
}

func (n Navigator) maybeAddProject(root localpath.Absolute) {
	if _, ok := n.projects.Peek(root); ok {
		return
	}
	opts := recommend.Options{
		Root:                 root,
		UseCommits:           true,
		ComputedCommitsLimit: n.opts.ComputedCommitsLimit,
	}
	n.projects.Add(root, newProjectNavigator(opts, n.load))
}

func (n Navigator) getProjectRoot(path localpath.Absolute) (localpath.Absolute, error) {
	// We assume `path` is not a directory, so it cannot be the project root.
	// We walk up the tree and call `n.isProjectRoot` on each directory above the path.
	return n.getProjectRootRecursive(path.Dir())
}

func (n Navigator) getProjectRootRecursive(path localpath.Absolute) (localpath.Absolute, error) {
	ok, err := n.isProjectRoot(path)
	if err != nil {
		return "", err
	}
	if ok {
		return path, nil
	}
	dir := path.Dir()
	if path == dir {
		return "", ErrPathNotInSupportedProject
	}
	return n.getProjectRootRecursive(dir)
}

func isProjectRoot(path localpath.Absolute) (bool, error) {
	info, err := path.Lstat()
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, nil
	}
	children, err := ioutil.ReadDir(string(path))
	if err != nil {
		return false, err
	}
	for _, child := range children {
		if child.IsDir() && child.Name() == ".git" {
			return true, nil
		}
	}
	return false, nil
}

func blockPath(operatingSystem string, path localpath.Absolute) error {
	if filters.IsFilteredDir(operatingSystem, string(path)) {
		return ErrPathInFilteredDirectory
	}
	if !path.HasSupportedExtension() {
		return ErrPathHasUnsupportedExtension
	}
	return nil
}

func normalize(operatingSystem, path string) string {
	if operatingSystem != "windows" {
		return path
	}
	return regexp.MustCompile("^[a-z]:").ReplaceAllStringFunc(path, strings.ToUpper)
}
