package ignore

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
)

const (
	// GitIgnoreFilename ...
	GitIgnoreFilename = ".gitignore"

	// KiteIgnoreFilename ...
	KiteIgnoreFilename = ".kite_code_nav_ignore"

	// HiddenDirectoriesPattern ...
	HiddenDirectoriesPattern = ".*/"

	maxIgnoreFilesnames = 2
	maxIgnoreFileSize   = 1e6
)

var errTooManyIgnoreFilenames = errors.New("too many ignore filenames")

// Ignorer determines which paths to ignore
type Ignorer interface {
	Ignore(pathname localpath.Absolute, isDir bool) bool
	ShouldRebuild() (bool, error)
}

type ignorer struct {
	opts       Options
	patterns   patternSet
	fileStates map[localpath.Relative]fileState
}

type fileState struct {
	exists  bool
	modTime time.Time
}

// Options ...
type Options struct {
	Root            localpath.Absolute
	IgnoreFilenames []localpath.Relative
	// IgnorePatterns are only used if no ignore files are found.
	IgnorePatterns []string
}

// New ...
func New(opts Options) (Ignorer, error) {
	return newIgnorer(opts)
}

func newIgnorer(opts Options) (ignorer, error) {
	if len(opts.IgnoreFilenames) > maxIgnoreFilesnames {
		return ignorer{}, errTooManyIgnoreFilenames
	}

	opts.Root = opts.Root.Clean()
	i := ignorer{
		opts: opts,
	}

	var err error
	i.fileStates, err = i.getFileStates()
	if err != nil {
		return ignorer{}, err
	}

	m := newMunger()

	for _, name := range opts.IgnoreFilenames {
		file, err := opts.Root.Join(name).Open()
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return ignorer{}, err
		}
		contents, err := ioutil.ReadAll(io.LimitReader(file, maxIgnoreFileSize))
		if err != nil {
			return ignorer{}, err
		}
		munged := m.mungePatterns(string(contents))
		patterns := parsePatterns(munged)
		i.patterns = append(i.patterns, patterns...)
	}

	if len(i.patterns) > 0 {
		return i, nil
	}

	munged := m.mungePatterns(strings.Join(opts.IgnorePatterns, "\n"))
	patterns := parsePatterns(munged)
	i.patterns = append(i.patterns, patterns...)

	return i, nil
}

// Assumes that parent directories have already been checked and should not be ignored.
func (i ignorer) Ignore(pathname localpath.Absolute, isDir bool) bool {
	if !strings.HasPrefix(string(pathname), string(i.opts.Root)) {
		// we clean `i.opts.Root` in `newIgnorer`, so it does not end with a separator.
		// whether or not `pathname` ends with a separator, we can check the prefix
		// to determine if `pathname` is under `i.opts.Root`.
		return true
	}
	rel, err := pathname.RelativeTo(i.opts.Root)
	if err != nil {
		return true
	}
	// don't ignore the root if there is an ignore pattern like ".*"
	if rel == "." {
		return false
	}
	gitFile := git.File(filepath.ToSlash(string(rel)))
	return i.patterns.ignore(gitFile, isDir)
}

// ShouldRebuild checks if the ignore files have been modified
func (i ignorer) ShouldRebuild() (bool, error) {
	current, err := i.getFileStates()
	if err != nil {
		return false, err
	}
	if len(current) != len(i.fileStates) {
		return true, nil
	}
	for _, name := range i.opts.IgnoreFilenames {
		if current[name] != i.fileStates[name] {
			return true, nil
		}
	}
	return false, nil
}

func (i ignorer) getFileStates() (map[localpath.Relative]fileState, error) {
	fileStates := make(map[localpath.Relative]fileState)
	for _, name := range i.opts.IgnoreFilenames {
		info, err := i.opts.Root.Join(name).Lstat()
		if os.IsNotExist(err) {
			fileStates[name] = fileState{}
			continue
		}
		if err != nil {
			return nil, err
		}
		fileStates[name] = fileState{
			exists:  true,
			modTime: info.ModTime(),
		}
	}
	return fileStates, nil
}
