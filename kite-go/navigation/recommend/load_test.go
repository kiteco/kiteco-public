package recommend

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/require"
)

var (
	astPath     = testDir.Join("astgo.py")
	parserPath  = testDir.Join("parsergo.py")
	datagenPath = testDir.Join("datagensh.py")
)

type canUseFileTC struct {
	ignorePatterns []string
	path           localpath.Absolute
	size           int64
	maxFileSize    int64
	expected       bool
}

func TestCanUseFile(t *testing.T) {
	root := localpath.Absolute("/pi/rho")
	if runtime.GOOS == "windows" {
		root = localpath.Absolute("C:" + filepath.FromSlash("/pi/rho"))
	}
	tcs := []canUseFileTC{
		canUseFileTC{
			ignorePatterns: []string{".*", "*alpha*", "beta", "gamma"},
			path:           root.Join("delta", "epsilon.go"),
			size:           10,
			maxFileSize:    100,
			expected:       true,
		},
		canUseFileTC{
			ignorePatterns: []string{".*", "*alpha*", "beta", "gamma"},
			path:           root.Join("delta", "epsilon.js"),
			size:           10,
			maxFileSize:    100,
			expected:       true,
		},
		canUseFileTC{
			ignorePatterns: []string{".*", "*alpha*", "beta", "gamma"},
			path:           root.Join("delta", "epsilon.rs"),
			size:           10,
			maxFileSize:    100,
			expected:       false,
		},
		canUseFileTC{
			ignorePatterns: []string{".*", "*alpha*", "beta", "gamma"},
			path:           root.Join(".delta"),
			size:           10,
			maxFileSize:    100,
			expected:       false,
		},
		canUseFileTC{
			ignorePatterns: []string{".*", "*alpha*", "beta", "gamma"},
			path:           root.Join("delta", ".epsilon.go"),
			size:           10,
			maxFileSize:    100,
			expected:       false,
		},
		canUseFileTC{
			ignorePatterns: []string{".*", "*alpha*", "beta", "gamma"},
			path:           root.Join("delta", "epsilonalphaphi.go"),
			size:           10,
			maxFileSize:    100,
			expected:       false,
		},
		canUseFileTC{
			ignorePatterns: []string{".*", "*alpha*", "beta", "gamma"},
			path:           root.Join("beta"),
			size:           10,
			maxFileSize:    100,
			expected:       false,
		},
		canUseFileTC{
			ignorePatterns: []string{".*", "*alpha*", "beta", "gamma"},
			path:           root.Join("delta", "epsilon.go"),
			size:           110,
			maxFileSize:    100,
			expected:       false,
		},
	}
	for _, tc := range tcs {
		i, err := ignore.New(ignore.Options{
			Root:           root,
			IgnorePatterns: tc.ignorePatterns,
		})
		require.NoError(t, err)
		r := recommender{
			opts: Options{
				MaxFileSize: tc.maxFileSize,
			},
			ignorer: i,
		}
		actual := r.canUseFile(tc.path, tc.size)
		require.Equal(t, tc.expected, actual, tc.path)
	}
}

func TestLoadGraph(t *testing.T) {
	kiteco := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "kiteco", "kiteco")
	root, err := localpath.NewAbsolute(kiteco)
	require.NoError(t, err)
	s, err := git.NewStorage(git.StorageOptions{})
	require.NoError(t, err)
	r := recommender{
		opts: Options{
			UseCommits:           true,
			ComputedCommitsLimit: 100,
			Root:                 root,
			MaxFileSize:          1e6,
			MaxFiles:             1e5,
		},
		params: parameters{
			maxMatrixSize: 1e5,
		},
		gitStorage: s,
	}
	r.fileIndex = r.newFileIndex()

	g, err := r.loadGraph(kitectx.Background())
	require.NoError(t, err)
	require.NotZero(t, len(g.files))
	require.NotZero(t, len(g.editSize))

	var fileSizeSum int
	for _, edits := range g.files {
		fileSizeSum += len(edits)
	}
	var editSizeSum uint32
	for _, editSize := range g.editSize {
		editSizeSum += editSize
	}
	require.True(t, uint32(fileSizeSum) == editSizeSum)
	require.NotNil(t, g.editScores)
	require.NotZero(t, g.totalEditScore)
	require.Equal(t, len(g.files), len(g.editScores))
}

func TestLoadVectorizer(t *testing.T) {
	i, err := ignore.New(ignore.Options{
		Root:           testDir,
		IgnorePatterns: []string{"*sh.py"},
	})
	require.NoError(t, err)
	r := recommender{
		opts: Options{
			Root:        testDir,
			MaxFileSize: 1e6,
			MaxFiles:    1e5,
		},
		ignorer: i,
	}
	r.fileOpener = r.newFileOpener()
	r.fileIndex = r.newFileIndex()
	err = r.loadVectorizer(kitectx.Background())
	require.NoError(t, err)
	require.Contains(t, r.vectorizer.idf, newShingle([]rune("subli")))
	require.NotContains(t, r.vectorizer.idf, newShingle([]rune("human")))

	astID, err := r.fileIndex.toID(astPath)
	require.NoError(t, err)
	require.Contains(t, r.vectorizer.vectorSet.data, astID)
	require.NotZero(t, r.vectorizer.vectorSet.data[astID].norm)
	require.NotZero(t, len(r.vectorizer.vectorSet.data[astID].coords))

	datagenID, err := r.fileIndex.toID(datagenPath)
	require.NoError(t, err)
	require.NotContains(t, r.vectorizer.vectorSet.data, datagenID)
}

func TestRefreshVectorSetModifiedFile(t *testing.T) {
	i, err := ignore.New(ignore.Options{Root: testDir})
	require.NoError(t, err)
	r := recommender{
		opts: Options{
			Root:        testDir,
			MaxFileSize: 1e6,
			MaxFiles:    1e5,
		},
		ignorer: i,
	}
	r.fileOpener = r.newFileOpener()
	r.fileIndex = r.newFileIndex()
	err = r.loadVectorizer(kitectx.Background())
	require.NoError(t, err)

	astID, err := r.fileIndex.toID(astPath)
	require.NoError(t, err)
	parserID, err := r.fileIndex.toID(parserPath)
	require.NoError(t, err)
	require.NotZero(t, r.vectorizer.vectorSet.data[astID].norm)
	require.NotZero(t, r.vectorizer.vectorSet.data[parserID].norm)

	r.vectorizer.vectorSet.data[astID] = shingleVector{
		modTime: r.vectorizer.vectorSet.data[astID].modTime.Add(-time.Minute),
	}
	r.vectorizer.vectorSet.data[parserID] = shingleVector{
		modTime: r.vectorizer.vectorSet.data[parserID].modTime,
	}
	require.Zero(t, r.vectorizer.vectorSet.data[astID].norm)
	require.Zero(t, r.vectorizer.vectorSet.data[parserID].norm)

	numRefreshedFiles, err := r.refreshVectorSet(kitectx.Background())
	require.NoError(t, err)
	require.Equal(t, 1, numRefreshedFiles)
	require.NotZero(t, r.vectorizer.vectorSet.data[astID].norm)
	require.Zero(t, r.vectorizer.vectorSet.data[parserID].norm)
}

func TestRefreshVectorSetNewFile(t *testing.T) {
	i, err := ignore.New(ignore.Options{Root: testDir})
	require.NoError(t, err)
	r := recommender{
		opts: Options{
			Root:        testDir,
			MaxFileSize: 1e6,
			MaxFiles:    1e5,
		},
		ignorer: i,
	}
	r.fileOpener = r.newFileOpener()
	r.fileIndex = r.newFileIndex()
	err = r.loadVectorizer(kitectx.Background())
	require.NoError(t, err)

	astID, err := r.fileIndex.toID(astPath)
	require.NoError(t, err)
	parserID, err := r.fileIndex.toID(parserPath)
	require.NoError(t, err)
	require.NotZero(t, r.vectorizer.vectorSet.data[astID].norm)
	require.NotZero(t, r.vectorizer.vectorSet.data[parserID].norm)
	require.Contains(t, r.vectorizer.vectorSet.data, astID)
	require.Contains(t, r.vectorizer.vectorSet.data, parserID)

	delete(r.vectorizer.vectorSet.data, astID)
	r.vectorizer.watchDirs.data[testDir] = r.vectorizer.watchDirs.data[testDir].Add(-time.Second)
	require.Zero(t, r.vectorizer.vectorSet.data[astID].norm)
	require.NotZero(t, r.vectorizer.vectorSet.data[parserID].norm)
	require.NotContains(t, r.vectorizer.vectorSet.data, astID)
	require.Contains(t, r.vectorizer.vectorSet.data, parserID)

	numRefreshedFiles, err := r.refreshVectorSet(kitectx.Background())
	require.NoError(t, err)
	require.Equal(t, 1, numRefreshedFiles)
	require.NotZero(t, r.vectorizer.vectorSet.data[astID].norm)
	require.NotZero(t, r.vectorizer.vectorSet.data[parserID].norm)
	require.Contains(t, r.vectorizer.vectorSet.data, astID)
	require.Contains(t, r.vectorizer.vectorSet.data, parserID)
}

func TestRefreshVectorSetNewDir(t *testing.T) {
	i, err := ignore.New(ignore.Options{Root: testDir})
	require.NoError(t, err)
	r := recommender{
		opts: Options{
			Root:        testDir,
			MaxFileSize: 1e6,
			MaxFiles:    1e5,
		},
		ignorer: i,
	}
	r.fileOpener = r.newFileOpener()
	r.fileIndex = r.newFileIndex()
	err = r.loadVectorizer(kitectx.Background())
	require.NoError(t, err)

	topDir := testDir.Join("alpha")
	subDir := topDir.Join("beta")
	newCode := subDir.Join("gamma.py")
	newID, err := r.fileIndex.toID(newCode)
	require.NoError(t, err)
	require.NotZero(t, r.vectorizer.vectorSet.data[newID].norm)
	require.Contains(t, r.vectorizer.vectorSet.data, newID)
	require.NotContains(t, r.vectorizer.watchDirs.data, newID)
	require.NotContains(t, r.vectorizer.vectorSet.data, subDir)
	require.Contains(t, r.vectorizer.watchDirs.data, subDir)
	require.NotContains(t, r.vectorizer.vectorSet.data, topDir)
	require.Contains(t, r.vectorizer.watchDirs.data, topDir)

	delete(r.vectorizer.vectorSet.data, newID)
	delete(r.vectorizer.watchDirs.data, subDir)
	r.vectorizer.watchDirs.data[topDir] = r.vectorizer.watchDirs.data[topDir].Add(-time.Minute)
	require.Zero(t, r.vectorizer.vectorSet.data[newID].norm)
	require.NotContains(t, r.vectorizer.vectorSet.data, newID)
	require.NotContains(t, r.vectorizer.watchDirs.data, newID)
	require.NotContains(t, r.vectorizer.vectorSet.data, subDir)
	require.NotContains(t, r.vectorizer.watchDirs.data, subDir)
	require.NotContains(t, r.vectorizer.vectorSet.data, topDir)
	require.Contains(t, r.vectorizer.watchDirs.data, topDir)

	numRefreshedFiles, err := r.refreshVectorSet(kitectx.Background())
	require.NoError(t, err)
	require.Equal(t, 1, numRefreshedFiles)
	require.NotZero(t, r.vectorizer.vectorSet.data[newID].norm)
	require.Contains(t, r.vectorizer.vectorSet.data, newID)
	require.NotContains(t, r.vectorizer.watchDirs.data, newCode)
	require.NotContains(t, r.vectorizer.vectorSet.data, subDir)
	require.Contains(t, r.vectorizer.watchDirs.data, subDir)
	require.NotContains(t, r.vectorizer.vectorSet.data, topDir)
	require.Contains(t, r.vectorizer.watchDirs.data, topDir)
}

func TestRefreshVectorSetDeletedFile(t *testing.T) {
	i, err := ignore.New(ignore.Options{Root: testDir})
	require.NoError(t, err)
	r := recommender{
		opts: Options{
			Root:        testDir,
			MaxFileSize: 1e6,
			MaxFiles:    1e5,
		},
		ignorer: i,
	}
	r.fileOpener = r.newFileOpener()
	r.fileIndex = r.newFileIndex()
	err = r.loadVectorizer(kitectx.Background())
	require.NoError(t, err)

	parserID, err := r.fileIndex.toID(parserPath)
	require.NoError(t, err)
	deletedPath, err := r.fileIndex.toID(testDir.Join("kite-go", "navigation", "recommend", "deleted.go"))
	require.NoError(t, err)
	require.NotContains(t, r.vectorizer.vectorSet.data, deletedPath)

	r.vectorizer.vectorSet.data[deletedPath] = shingleVector{
		modTime: r.vectorizer.vectorSet.data[parserID].modTime,
	}
	require.Contains(t, r.vectorizer.vectorSet.data, deletedPath)

	numRefreshedFiles, err := r.refreshVectorSet(kitectx.Background())
	require.NoError(t, err)
	require.Equal(t, 0, numRefreshedFiles)
	require.NotContains(t, r.vectorizer.vectorSet.data, deletedPath)
}

func TestRefreshVectorSetDeletedDir(t *testing.T) {
	i, err := ignore.New(ignore.Options{Root: testDir})
	require.NoError(t, err)
	r := recommender{
		opts: Options{
			Root:        testDir,
			MaxFileSize: 1e6,
			MaxFiles:    1e5,
		},
		ignorer: i,
	}
	r.fileOpener = r.newFileOpener()
	r.fileIndex = r.newFileIndex()
	err = r.loadVectorizer(kitectx.Background())
	require.NoError(t, err)

	deletedDir := testDir.Join("kite-go", "navigation", "deleted")
	deletedSubDir := deletedDir.Join("sub")
	deletedPath, err := r.fileIndex.toID(deletedSubDir.Join("path.go"))
	require.NoError(t, err)
	require.NotContains(t, r.vectorizer.vectorSet.data, deletedPath)
	require.NotContains(t, r.vectorizer.watchDirs.data, deletedPath)
	require.NotContains(t, r.vectorizer.vectorSet.data, deletedSubDir)
	require.NotContains(t, r.vectorizer.watchDirs.data, deletedSubDir)
	require.NotContains(t, r.vectorizer.vectorSet.data, deletedDir)
	require.NotContains(t, r.vectorizer.watchDirs.data, deletedDir)

	r.vectorizer.watchDirs.data[deletedDir] = time.Time{}
	r.vectorizer.watchDirs.data[deletedSubDir] = time.Time{}
	r.vectorizer.vectorSet.data[deletedPath] = shingleVector{}
	require.Contains(t, r.vectorizer.vectorSet.data, deletedPath)
	require.NotContains(t, r.vectorizer.watchDirs.data, deletedPath)
	require.NotContains(t, r.vectorizer.vectorSet.data, deletedSubDir)
	require.Contains(t, r.vectorizer.watchDirs.data, deletedSubDir)
	require.NotContains(t, r.vectorizer.vectorSet.data, deletedDir)
	require.Contains(t, r.vectorizer.watchDirs.data, deletedDir)

	numRefreshedFiles, err := r.refreshVectorSet(kitectx.Background())
	require.NoError(t, err)
	require.Equal(t, 0, numRefreshedFiles)
	require.NotContains(t, r.vectorizer.vectorSet.data, deletedPath)
	require.NotContains(t, r.vectorizer.watchDirs.data, deletedPath)
	require.NotContains(t, r.vectorizer.vectorSet.data, deletedSubDir)
	require.NotContains(t, r.vectorizer.watchDirs.data, deletedSubDir)
	require.NotContains(t, r.vectorizer.vectorSet.data, deletedDir)
	require.NotContains(t, r.vectorizer.watchDirs.data, deletedDir)
}

func TestFileOpener(t *testing.T) {
	r := recommender{
		opts: Options{
			MaxFiles: 3,
		},
		params: parameters{
			maxFileOpensPerSecond: 5,
		},
	}
	r.fileOpener = r.newFileOpener()
	paths := []localpath.Absolute{
		testDir.Join("astgo.py"),
		testDir.Join("datagensh.py"),
		testDir.Join("logscss.py"),
		testDir.Join("trainpy.py"),
	}

	prev := time.Now()
	for i := 0; i < 3; i++ {
		f, err := r.fileOpener.open(paths[i])
		require.NotNil(t, f)
		require.NoError(t, err)
	}
	require.True(t, time.Since(prev) >= 590*time.Millisecond)

	prev = time.Now()
	f, err := r.fileOpener.open(paths[3])
	require.Nil(t, f)
	require.Error(t, err)
	require.True(t, time.Since(prev) <= 210*time.Millisecond)

	r.fileOpener.releaseMax()
	prev = time.Now()
	f, err = r.fileOpener.open(paths[3])
	require.NotNil(t, f)
	require.NoError(t, err)
	require.True(t, time.Since(prev) >= 190*time.Millisecond)
}

func TestRead(t *testing.T) {
	r := recommender{
		opts: Options{
			MaxFiles:    10,
			MaxFileSize: 1e6,
		},
		params: parameters{
			maxFileOpensPerSecond: 1000,
		},
	}
	r.fileOpener = r.newFileOpener()

	contents, err := r.read(astPath)
	require.NoError(t, err)
	require.Equal(t, "package pythonast", regexp.MustCompile("\n|\r\n").Split(string(contents), 3)[1])

	r.opts.MaxFileSize = 15
	contents, err = r.read(astPath)
	require.NoError(t, err)

	// depends on \n vs \r\n
	expected := []string{"package pyt", "package py"}

	actual := regexp.MustCompile("\n|\r\n").Split(string(contents), 3)[1]
	require.Contains(t, expected, actual)
}

func TestGetCommits(t *testing.T) {
	kiteco := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "kiteco", "kiteco")
	i, err := ignore.New(ignore.Options{
		Root: localpath.Absolute(kiteco),
	})
	require.NoError(t, err)
	s, err := git.NewStorage(git.StorageOptions{})
	require.NoError(t, err)
	r := recommender{
		opts: Options{
			Root:                 localpath.Absolute(kiteco),
			ComputedCommitsLimit: git.DefaultComputedCommitsLimit,
		},
		params: parameters{
			maxMatrixSize: 1000,
		},
		ignorer:    i,
		gitStorage: s,
	}

	commitFiles, numEdits, err := r.getCommits(kitectx.Background())
	require.NoError(t, err)
	require.NotZero(t, numEdits)
	seen := make(map[git.File]bool)
	for file := range commitFiles {
		seen[file] = true
	}
	var hits int
	err = filepath.Walk(kiteco, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			require.NotContains(t, seen, path)
			return nil
		}
		rel, err := filepath.Rel(kiteco, path)
		require.NoError(t, err)
		gitFile := git.File(filepath.ToSlash(rel))
		if seen[gitFile] {
			hits++
		}
		return nil
	})
	require.NoError(t, err)
	require.True(t, hits >= 5)
}
