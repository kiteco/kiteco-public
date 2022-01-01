package codebase

import (
	"errors"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type projectNavigator struct {
	load  loadFunc
	opts  recommend.Options
	m     *sync.Mutex
	state projectState
}

type loadFunc func(kitectx.Context, git.Storage, ignore.Options, recommend.Options) projectState

type projectState struct {
	status      ProjectStatus
	lastUsed    time.Time
	ignorer     ignore.Ignorer
	recommender recommend.Recommender
	err         error
}

// ProjectStatus for loading a project navigator
type ProjectStatus string

const (
	// Inactive indicates the project navigator has not attempted to build yet or has expired and unloaded.
	Inactive ProjectStatus = "inactive"

	// InProgress indicates the project navigator is building.
	InProgress ProjectStatus = "in progress"

	// Active indicates the project navigator finished building successfully and is currently active.
	Active ProjectStatus = "active"

	// Failed indicates the project navigator attempted to build but was unsuccessful.
	Failed ProjectStatus = "failed"

	// IgnorerFailed indicates the ignorer attempted to build but was unsuccessful.
	IgnorerFailed ProjectStatus = "ignorer failed"
)

// Errors exported for codenav API ...
var (
	ErrShouldLoad    = errors.New("project should be loaded")
	ErrEmptyIterator = errors.New("iterator is empty")

	errWasInProgress   = errors.New("project status was InProgress")
	errInvalidNumFiles = errors.New("numFiles must be positive")
)

func newProjectNavigator(opts recommend.Options, load loadFunc) *projectNavigator {
	return &projectNavigator{
		opts:  opts,
		load:  load,
		m:     new(sync.Mutex),
		state: projectState{status: Inactive},
	}
}

func (p *projectNavigator) getState() projectState {
	p.m.Lock()
	defer p.m.Unlock()

	return p.state
}

func (p *projectNavigator) updateLastUsed(t time.Time) {
	p.m.Lock()
	defer p.m.Unlock()

	p.state.lastUsed = t
}

func (p *projectNavigator) maybeLoad(ctx kitectx.Context, s git.Storage, maxFileSize int64, maxFiles int) {
	if !p.shouldLoad() {
		return
	}

	ignoreOpts := ignore.Options{
		Root:            p.opts.Root,
		IgnorePatterns:  []string{ignore.HiddenDirectoriesPattern},
		IgnoreFilenames: []localpath.Relative{ignore.GitIgnoreFilename, ignore.KiteIgnoreFilename},
	}
	recommendOpts := p.opts
	recommendOpts.MaxFiles = maxFiles
	recommendOpts.MaxFileSize = maxFileSize
	newState := p.load(ctx, s, ignoreOpts, recommendOpts)

	p.m.Lock()
	defer p.m.Unlock()

	p.state = newState
}

func (p *projectNavigator) shouldLoad() bool {
	p.m.Lock()
	defer p.m.Unlock()

	if p.state.status == Inactive {
		p.state.status = InProgress
		return true
	}

	rebuild, err := p.shouldRebuild()
	if err != nil {
		p.state = projectState{
			status: Failed,
			err:    err,
		}
		return false
	}
	if rebuild {
		p.state = projectState{status: InProgress}
		return true
	}
	return false
}

func (p *projectNavigator) shouldRebuild() (bool, error) {
	if p.state.status == Active {
		return p.state.recommender.ShouldRebuild()
	}
	if p.state.status == Failed && p.state.err == recommend.ErrOpenedTooManyFiles {
		return p.state.ignorer.ShouldRebuild()
	}
	return false, nil
}

// FileIterator yields file recommendations.
// It is not safe to call from multiple go routines.
type FileIterator struct {
	Next func(numFiles int) ([]recommend.File, error)
}

func (p *projectNavigator) navigate(ctx kitectx.Context, request recommend.Request) (FileIterator, error) {
	state := p.getState()
	switch state.status {
	case Inactive:
		return FileIterator{}, ErrShouldLoad
	case InProgress:
		return FileIterator{}, errWasInProgress
	case Failed, IgnorerFailed:
		return FileIterator{}, state.err
	}

	shouldRebuild, err := p.shouldRebuild()
	if err != nil {
		return FileIterator{}, err
	}
	if shouldRebuild {
		return FileIterator{}, ErrShouldLoad
	}

	files, err := state.recommender.Recommend(ctx, request)
	if err != nil {
		return FileIterator{}, err
	}
	var idx int

	return FileIterator{
		Next: func(numFiles int) ([]recommend.File, error) {
			if idx == len(files) {
				return nil, ErrEmptyIterator
			}
			if numFiles <= 0 {
				return nil, errInvalidNumFiles
			}
			hi := idx + numFiles
			if hi > len(files) {
				hi = len(files)
			}
			blockRequest := recommend.BlockRequest{
				Request:      request,
				InspectFiles: files[idx:hi],
			}
			idx = hi
			return state.recommender.RecommendBlocks(ctx, blockRequest)
		},
	}, nil
}

func load(ctx kitectx.Context, s git.Storage, ignoreOpts ignore.Options, recOpts recommend.Options) projectState {
	i, err := ignore.New(ignoreOpts)
	if err != nil {
		return projectState{
			status: IgnorerFailed,
			err:    err,
		}
	}
	r, err := recommend.NewRecommender(ctx, recOpts, i, s)
	if err != nil {
		return projectState{
			// Keep the ignorer to have access to ShouldRebuild
			ignorer: i,
			status:  Failed,
			err:     err,
		}
	}
	return projectState{
		recommender: r,
		status:      Active,
	}
}
