package codebase

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/stretchr/testify/require"

	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/recommend"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

var (
	testDirString = filepath.Join(
		os.Getenv("GOPATH"), "src", "github.com", "kiteco", "kiteco",
		"kite-go", "navigation", "offline", "testdata",
	)
	testDir           = localpath.Absolute(testDirString)
	lexPath           = filepath.Join(testDirString, "kite-go", "lang", "lexical", "lexicalcomplete", "api", "api.go")
	pyPath            = filepath.Join(testDirString, "kite-go", "lang", "python", "pythoncomplete", "api", "api.go")
	encPath           = filepath.Join(testDirString, "kite-golib", "lexicalv0", "encoder.go")
	trainPath         = filepath.Join(testDirString, "local-pipelines", "lexical", "train", "train.py")
	modelPath         = filepath.Join(testDirString, "kite-python", "kite_ml", "kite", "model", "model.py")
	readmePath        = filepath.Join(testDirString, "README.md")
	maxFileSize int64 = 1e6
	maxFiles    int   = 1e5
)

type pathTC struct {
	operatingSystem    string
	path               string
	expectedNormalized string
	expectedBlocked    error
}

func TestPath(t *testing.T) {
	tcs := []pathTC{
		pathTC{
			operatingSystem:    "darwin",
			path:               "/alpha/beta/Library/gamma/delta.py",
			expectedNormalized: "/alpha/beta/Library/gamma/delta.py",
			expectedBlocked:    ErrPathInFilteredDirectory,
		},
		pathTC{
			operatingSystem:    "darwin",
			path:               "/alpha/beta/.gamma/delta.py",
			expectedNormalized: "/alpha/beta/.gamma/delta.py",
			expectedBlocked:    nil,
		},
		pathTC{
			operatingSystem:    "darwin",
			path:               "/alpha/beta/.gamma/delta.rs",
			expectedNormalized: "/alpha/beta/.gamma/delta.rs",
			expectedBlocked:    ErrPathHasUnsupportedExtension,
		},
		pathTC{
			operatingSystem:    "darwin",
			path:               "/go/src/github.com/kiteco/kiteco/windows/client/KiteService/LibraryIO.cs",
			expectedNormalized: "/go/src/github.com/kiteco/kiteco/windows/client/KiteService/LibraryIO.cs",
			// nil would be better
			expectedBlocked: ErrPathInFilteredDirectory,
		},
		pathTC{
			operatingSystem:    "linux",
			path:               "/alpha/beta/.gamma/delta.py",
			expectedNormalized: "/alpha/beta/.gamma/delta.py",
			expectedBlocked:    ErrPathInFilteredDirectory,
		},
		pathTC{
			operatingSystem:    "linux",
			path:               "/alpha/beta/Library/gamma/delta.py",
			expectedNormalized: "/alpha/beta/Library/gamma/delta.py",
			expectedBlocked:    nil,
		},
		pathTC{
			operatingSystem:    "windows",
			path:               "C:\\alpha\\beta\\appdata\\gamma\\delta.py",
			expectedNormalized: "C:\\alpha\\beta\\appdata\\gamma\\delta.py",
			expectedBlocked:    ErrPathInFilteredDirectory,
		},
		pathTC{
			operatingSystem:    "windows",
			path:               "C:\\alpha\\beta\\Library\\gamma\\delta.py",
			expectedNormalized: "C:\\alpha\\beta\\Library\\gamma\\delta.py",
			expectedBlocked:    nil,
		},
		pathTC{
			operatingSystem:    "windows",
			path:               "c:\\alpha\\beta\\appdata\\gamma\\delta.py",
			expectedNormalized: "C:\\alpha\\beta\\appdata\\gamma\\delta.py",
			expectedBlocked:    ErrPathInFilteredDirectory,
		},
		pathTC{
			operatingSystem:    "windows",
			path:               "c:\\alpha\\beta\\Library\\gamma\\delta.py",
			expectedNormalized: "C:\\alpha\\beta\\Library\\gamma\\delta.py",
			expectedBlocked:    nil,
		},
		pathTC{
			operatingSystem:    "windows",
			path:               "d:\\alpha\\beta\\Library\\gamma\\delta.py",
			expectedNormalized: "D:\\alpha\\beta\\Library\\gamma\\delta.py",
			expectedBlocked:    nil,
		},
		pathTC{
			operatingSystem:    "windows",
			path:               "D:\\alpha\\beta\\Library\\gamma\\delta.py",
			expectedNormalized: "D:\\alpha\\beta\\Library\\gamma\\delta.py",
			expectedBlocked:    nil,
		},
		pathTC{
			operatingSystem:    "windows",
			path:               "cd:\\alpha\\beta\\Library\\gamma\\delta.py",
			expectedNormalized: "cd:\\alpha\\beta\\Library\\gamma\\delta.py",
			expectedBlocked:    nil,
		},
	}

	for _, tc := range tcs {
		normalized := normalize(tc.operatingSystem, tc.path)
		blocked := blockPath(tc.operatingSystem, localpath.Absolute(normalized))
		require.Equal(t, tc.expectedNormalized, normalized)
		require.Equal(t, tc.expectedBlocked, blocked)
	}
}

type getProjectRootTC struct {
	path          localpath.Absolute
	expectedPath  localpath.Absolute
	expectedError error
}

func TestGetProjectRoot(t *testing.T) {
	tcs := []getProjectRootTC{
		getProjectRootTC{
			path:         testDir.Join("alpha", "BETA", "gamma"),
			expectedPath: testDir.Join("alpha", "BETA"),
		},
		getProjectRootTC{
			path:         testDir.Join("DELTA", "sigma", "tau"),
			expectedPath: testDir.Join("DELTA"),
		},
		getProjectRootTC{
			path:          testDir.Join("DELTA"),
			expectedError: ErrPathNotInSupportedProject,
		},
		getProjectRootTC{
			path:         testDir.Join("epsilon", "PHI", "sigma", "tau"),
			expectedPath: testDir.Join("epsilon", "PHI"),
		},
		getProjectRootTC{
			path:         testDir.Join("EPSILON", "PHI", "sigma", "tau"),
			expectedPath: testDir.Join("EPSILON", "PHI"),
		},
		getProjectRootTC{
			path:          testDir.Join("beta", "gamma"),
			expectedError: ErrPathNotInSupportedProject,
		},
	}

	for _, tc := range tcs {
		n := Navigator{
			isProjectRoot: func(path localpath.Absolute) (bool, error) {
				if path.Dir() == path {
					return false, nil
				}
				base := filepath.Base(string(path))
				return base == strings.ToUpper(base), nil
			},
		}
		path, err := n.getProjectRoot(tc.path)
		require.Equal(t, tc.expectedError, err, tc.path)
		require.Equal(t, tc.expectedPath, path)
	}
}

type isProjectRootTC struct {
	path         localpath.Absolute
	expected     bool
	doesNotExist bool
}

func TestIsProjectRoot(t *testing.T) {
	kiteco := localpath.Absolute(os.Getenv("GOPATH")).Join("src", "github.com", "kiteco", "kiteco")
	tcs := []isProjectRootTC{
		isProjectRootTC{
			path:     kiteco,
			expected: true,
		},
		isProjectRootTC{
			path: kiteco.Join("kite-go"),
		},
		isProjectRootTC{
			path:         kiteco.Join("kite-haskell"),
			doesNotExist: true,
		},
	}
	for _, tc := range tcs {
		actual, err := isProjectRoot(tc.path)
		require.Equal(t, tc.doesNotExist, os.IsNotExist(err))
		require.Equal(t, tc.expected, actual)
	}
}

func TestProjectInfo(t *testing.T) {
	cache, err := lru.New(defaultMaxProjects)
	require.NoError(t, err)
	s, err := git.NewStorage(git.StorageOptions{})
	require.NoError(t, err)
	n := Navigator{
		isProjectRoot: func(path localpath.Absolute) (bool, error) {
			return path == testDir, nil
		},
		projects:   cache,
		gitStorage: s,
		load: func(ctx kitectx.Context, s git.Storage, ignoreOpts ignore.Options, recOpts recommend.Options) projectState {
			time.Sleep(150 * time.Millisecond)
			r := mockRecommender{
				files: []recommend.File{{Path: pyPath}},
			}
			return projectState{
				recommender: r,
				status:      Active,
			}
		},
		m:        new(sync.Mutex),
		indexing: make(chan struct{}, 1),
		term:     newTerminator(),
	}

	status, project, err := n.ProjectInfo(modelPath)
	require.Equal(t, ErrProjectNotLoaded, err)
	require.Equal(t, ProjectStatus(""), status)
	require.Equal(t, testDirString, project)

	n.MaybeLoad(lexPath, maxFileSize, maxFiles)
	status, project, err = n.ProjectInfo(modelPath)
	require.NoError(t, err)
	require.Equal(t, Active, status)
	require.Equal(t, testDirString, project)

	status, project, err = n.ProjectInfo(readmePath)
	require.Equal(t, ErrPathHasUnsupportedExtension, err)
	require.Equal(t, ProjectStatus(""), status)
	require.Equal(t, "", project)
}

func TestNavigateAndUnload(t *testing.T) {
	cache, err := lru.New(defaultMaxProjects)
	require.NoError(t, err)
	n := Navigator{
		isProjectRoot: func(path localpath.Absolute) (bool, error) {
			return path == testDir, nil
		},
		projects: cache,
		load: func(ctx kitectx.Context, s git.Storage, ignoreOpts ignore.Options, recOpts recommend.Options) projectState {
			time.Sleep(1500 * time.Millisecond)
			r := mockRecommender{
				files: []recommend.File{{Path: pyPath}},
			}
			return projectState{
				recommender: r,
				status:      Active,
			}
		},
		m:        new(sync.Mutex),
		indexing: make(chan struct{}, 1),
		term:     newTerminator(),
	}

	files, err := n.Navigate(recommend.Request{
		Location: recommend.Location{
			CurrentPath: lexPath,
		},
	})
	if err == ErrShouldLoad {
		go n.MaybeLoad(lexPath, maxFileSize, maxFiles)
	}

	time.Sleep(1000 * time.Millisecond)

	require.Equal(t, ErrShouldLoad, err)
	require.Equal(t, FileIterator{}, files)

	v, ok := n.projects.Get(testDir)
	require.True(t, ok)
	require.Equal(t, InProgress, v.(*projectNavigator).state.status)

	time.Sleep(1000 * time.Millisecond)

	iter, err := n.Navigate(recommend.Request{
		Location: recommend.Location{
			CurrentPath: lexPath,
		},
	})
	require.NoError(t, err)

	batch, err := iter.Next(1)
	require.NoError(t, err)
	require.Equal(t, []recommend.File{{Path: pyPath}}, batch)

	v, ok = n.projects.Get(testDir)
	require.True(t, ok)
	require.Equal(t, Active, v.(*projectNavigator).state.status)

	time.Sleep(time.Second)
	n.MaybeUnload(time.Second)
	require.False(t, n.projects.Contains(testDir))
}

func TestIndexAndNavigate(t *testing.T) {
	cache, err := lru.New(defaultMaxProjects)
	require.NoError(t, err)
	n := Navigator{
		isProjectRoot: func(path localpath.Absolute) (bool, error) {
			return path.Dir() == testDir, nil
		},
		projects: cache,
		load: func(ctx kitectx.Context, s git.Storage, ignoreOpts ignore.Options, recOpts recommend.Options) projectState {
			time.Sleep(150 * time.Millisecond)
			r := mockRecommender{
				files: []recommend.File{{Path: pyPath}},
			}
			return projectState{
				status:      Active,
				recommender: r,
			}
		},
		m:        new(sync.Mutex),
		indexing: make(chan struct{}, 1),
		term:     newTerminator(),
	}

	n.MaybeLoad(lexPath, maxFileSize, maxFiles)
	go n.MaybeLoad(encPath, maxFileSize, maxFiles)
	time.Sleep(time.Millisecond)
	before := time.Now()
	iter, err := n.Navigate(recommend.Request{
		Location: recommend.Location{
			CurrentPath: lexPath,
		},
	})

	require.True(t, time.Since(before) < 10*time.Millisecond)
	require.NoError(t, err)

	files, err := iter.Next(5)
	require.NoError(t, err)
	require.Equal(t, 1, len(files))
	require.Equal(t, []recommend.File{{Path: pyPath}}, files)
}

func TestIndexAndTerminate(t *testing.T) {
	cache, err := lru.New(defaultMaxProjects)
	require.NoError(t, err)
	n := Navigator{
		isProjectRoot: func(path localpath.Absolute) (bool, error) {
			return path.Dir() == testDir, nil
		},
		projects: cache,
		load: func(ctx kitectx.Context, s git.Storage, ignoreOpts ignore.Options, recOpts recommend.Options) projectState {
			for true {
				ctx.CheckAbort()
				time.Sleep(100 * time.Millisecond)
			}
			panic("infinite loop")
		},
		m:        new(sync.Mutex),
		indexing: make(chan struct{}, 1),
		term:     newTerminator(),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		n.MaybeLoad(encPath, maxFileSize, maxFiles)
	}()
	time.Sleep(250 * time.Millisecond)
	n.Terminate()
	wg.Wait()
}

func TestProjectsLRU(t *testing.T) {
	cache, err := lru.New(3)
	require.NoError(t, err)
	n := Navigator{
		isProjectRoot: func(path localpath.Absolute) (bool, error) {
			return path.Dir() == testDir, nil
		},
		projects: cache,
		load: func(ctx kitectx.Context, s git.Storage, ignoreOpts ignore.Options, recOpts recommend.Options) projectState {
			time.Sleep(150 * time.Millisecond)
			r := mockRecommender{
				files: []recommend.File{{Path: pyPath}},
			}
			return projectState{
				status:      Active,
				recommender: r,
			}
		},
		m:        new(sync.Mutex),
		indexing: make(chan struct{}, 1),
		term:     newTerminator(),
	}

	n.MaybeLoad(lexPath, maxFileSize, maxFiles)
	n.MaybeLoad(encPath, maxFileSize, maxFiles)
	n.MaybeLoad(modelPath, maxFileSize, maxFiles)
	n.MaybeLoad(trainPath, maxFileSize, maxFiles)

	require.Equal(t, 3, n.projects.Len())
	k, _, ok := n.projects.GetOldest()
	require.True(t, ok)
	require.Equal(t, testDir.Join("kite-golib"), k)

	require.True(t, n.projects.Contains(testDir.Join("kite-python")))
	require.True(t, n.projects.Contains(testDir.Join("local-pipelines")))
	require.False(t, n.projects.Contains(testDir.Join("kite-go")))

	_, err = n.Navigate(recommend.Request{
		Location: recommend.Location{
			CurrentPath: lexPath,
		},
	})
	require.Equal(t, ErrShouldLoad, err)
}

func TestRebuild(t *testing.T) {
	tcs := []projectState{
		projectState{
			status: Active,
			recommender: mockRecommender{
				files:         []recommend.File{{Path: lexPath}},
				shouldRebuild: true,
			},
		},
	}
	for _, tc := range tcs {
		cache, err := lru.New(defaultMaxProjects)
		require.NoError(t, err)
		n := Navigator{
			isProjectRoot: func(path localpath.Absolute) (bool, error) {
				return path == testDir, nil
			},
			projects: cache,
			m:        new(sync.Mutex),
			indexing: make(chan struct{}, 1),
			term:     newTerminator(),
		}
		n.projects.Add(testDir, &projectNavigator{
			state: tc,
			load: func(ctx kitectx.Context, s git.Storage, ignoreOpts ignore.Options, recOpts recommend.Options) projectState {
				time.Sleep(1000 * time.Millisecond)
				r := mockRecommender{
					files: []recommend.File{{Path: pyPath}},
				}
				return projectState{
					status:      Active,
					recommender: r,
				}
			},
			m: new(sync.Mutex),
		})

		_, err = n.Navigate(recommend.Request{
			Location: recommend.Location{
				CurrentPath: encPath,
			},
		})
		require.Equal(t, ErrShouldLoad, err)

		go n.MaybeLoad(encPath, maxFileSize, maxFiles)
		time.Sleep(500 * time.Millisecond)

		duringIter, err := n.Navigate(recommend.Request{
			Location: recommend.Location{
				CurrentPath: encPath,
			},
		})
		require.Error(t, errWasInProgress)
		require.Equal(t, FileIterator{}, duringIter)

		time.Sleep(1000 * time.Millisecond)
		afterIter, err := n.Navigate(recommend.Request{
			Location: recommend.Location{
				CurrentPath: encPath,
			},
		})
		require.NoError(t, err)

		afterFiles, err := afterIter.Next(5)
		require.NoError(t, err)
		require.Equal(t, []recommend.File{{Path: pyPath}}, afterFiles)
	}
}
