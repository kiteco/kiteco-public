package pythonbatch

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fileSet map[string]string

// tempDirRoot is used by the tests as root temp dir,
// because the OS tempdir is a filtered directory
var tempDirRoot, _ = os.UserHomeDir()

func (g fileSet) Get(key string) ([]byte, error) {
	val, ok := g[key]
	if ok {
		return []byte(val), nil
	}
	fs := localcode.LocalFileSystem{}
	return fs.Get(key)
}

func doTest(t testing.TB, getter fileSet, libDirs []string, opts SelectionOptions, root string, expected map[string]string) {

	var err error
	fs := localcode.LocalFileSystem{}
	ctx := kitectx.Background()
	var files []*localfiles.File
	err = fs.Walk(ctx, filepath.Dir(root), func(path string, fi localcode.FileInfo, err error) error {
		if err != nil {
			if fi.IsDir {
				return localcode.ErrSkipDir
			}
			return nil
		}
		if !fi.IsDir {
			files = append(files, &localfiles.File{
				Name:          path,
				HashedContent: path,
			})
		}
		return nil
	})
	require.NoError(t, err)

	selected, _, err := Selector{
		StartPaths:   []string{root},
		Files:        files,
		Getter:       getter,
		Opts:         opts,
		Logf:         t.Logf,
		LibraryFiles: libDirs,
		FileSystem:   fs,
		Local:        true,
	}.Select(ctx)
	require.NoError(t, err)

	actual := make(map[string]string)
	for path, source := range selected {
		actual[path] = source.ASTBundle.LibraryPath
	}

	require.Equal(t, expected, actual)
}

func testSetup(t testing.TB, tempdir, root string, inputs fileSet, outputs map[string]string) (string, fileSet, []string, map[string]string) {
	// update paths to be rooted at temporary directory
	root = filepath.Join(tempdir, root)

	files := make(map[string]string)
	libPaths := make(map[string]bool)
	for p, contents := range inputs {
		// create files
		tempfile := filepath.Join(tempdir, p)
		dirPath := filepath.Dir(tempfile)
		err := os.MkdirAll(dirPath, os.ModePerm)
		require.NoError(t, err)
		err = ioutil.WriteFile(tempfile, []byte(inputs[p]), os.ModePerm)
		require.NoError(t, err)
		files[tempfile] = contents
		// check if library file
		prefix := GetLibraryPrefix(tempfile)
		if prefix != "" {
			libPaths[prefix] = true
		}
	}

	var libDirs []string
	for lib := range libPaths {
		libDirs = append(libDirs, lib)
	}

	expected := make(map[string]string)
	for p, libPath := range outputs {
		tempfile := filepath.Join(tempdir, p)
		if libPath != "" {
			libPath = filepath.Join(tempdir, libPath)
		}
		expected[tempfile] = libPath
	}

	return root, files, libDirs, expected
}

func TestSelectFiles_All(t *testing.T) {
	tempdir, err := ioutil.TempDir(tempDirRoot, "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tempdir) // clean up

	root := "/project_root/foo/bar/src.py"
	files := fileSet{
		"/project_root/foo/bar/src.py":      "from baz import src",   // 0. /project_root/foo/bar/baz/src.py
		"/project_root/foo/bar/baz/src.py":  "import bar",            // 1. /project_root/bar.py
		"/project_root/bar.py":              "import foo",            // 2. /project_root/foo/__init__.py
		"/project_root/foo/__init__.py":     "import bar",            // 3. /project_root/foo/bar/__init__.py
		"/project_root/foo/bar/__init__.py": "from ..baz import src", // 4. /project_root/foo/baz/src.py
		"/project_root/foo/baz/src.py":      "import baz",            // 5. /my/site-packages/baz/__init__.py
		"/my/site-packages/baz/__init__.py": "import src",            // 6. /my/site-packages/baz/src.py
		"/my/site-packages/baz/src.py":      "",                      // 7.
	}
	expected := make(map[string]string)
	for path := range files {
		expected[path] = GetLibraryPrefix(path)
	}
	root, files, libDirs, expected := testSetup(t, tempdir, root, files, expected)
	doTest(t, files, libDirs, DefaultOptions.PathSelection, root, expected)
}

func TestSelectFiles_ProjectLimit(t *testing.T) {
	tempdir, err := ioutil.TempDir(tempDirRoot, "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tempdir) // clean up

	root := "/project_root/foo/bar/src.py"
	files := fileSet{
		"/project_root/foo/bar/src.py":      "from baz import src",   // 0. /project_root/foo/bar/baz/src.py
		"/project_root/foo/bar/baz/src.py":  "import bar",            // 1. /project_root/bar.py
		"/project_root/bar.py":              "import foo",            // 2. /project_root/foo/__init__.py
		"/project_root/foo/__init__.py":     "import bar",            // 3. /project_root/foo/bar/__init__.py
		"/project_root/foo/bar/__init__.py": "from ..baz import src", // 4. /project_root/foo/baz/src.py
		"/project_root/foo/baz/src.py":      "import baz",            // 5. /my/site-packages/baz/__init__.py
		"/my/site-packages/baz/__init__.py": "import src",            // 6. /my/site-packages/baz/src.py
		"/my/site-packages/baz/src.py":      "",                      // 7.
	}
	expected := map[string]string{
		"/project_root/foo/bar/src.py":     "",
		"/project_root/foo/bar/baz/src.py": "",
	}
	root, files, libDirs, expected := testSetup(t, tempdir, root, files, expected)
	opts := DefaultOptions.PathSelection
	opts.ProjectFileLimit = 2
	doTest(t, files, libDirs, opts, root, expected)
}

func TestSelectFiles_LibraryLimit(t *testing.T) {
	tempdir, err := ioutil.TempDir(tempDirRoot, "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tempdir) // clean up

	root := "/project_root/foo/bar/src.py"
	files := fileSet{
		"/project_root/foo/bar/src.py":      "from baz import src",   // 0. /project_root/foo/bar/baz/src.py
		"/project_root/foo/bar/baz/src.py":  "import bar",            // 1. /project_root/bar.py
		"/project_root/bar.py":              "import foo",            // 2. /project_root/foo/__init__.py
		"/project_root/foo/__init__.py":     "import bar",            // 3. /project_root/foo/bar/__init__.py
		"/project_root/foo/bar/__init__.py": "from ..baz import src", // 4. /project_root/foo/baz/src.py
		"/project_root/foo/baz/src.py":      "import baz",            // 5. /my/site-packages/baz/__init__.py
		"/my/site-packages/baz/__init__.py": "import src",            // 6. /my/site-packages/baz/src.py
		"/my/site-packages/baz/src.py":      "",                      // 7.
	}
	expected := map[string]string{
		"/project_root/foo/bar/src.py":      "",
		"/project_root/foo/bar/baz/src.py":  "",
		"/project_root/bar.py":              "",
		"/project_root/foo/__init__.py":     "",
		"/project_root/foo/bar/__init__.py": "",
		"/project_root/foo/baz/src.py":      "",
		"/my/site-packages/baz/__init__.py": "/my/site-packages",
	}
	root, files, libDirs, expected := testSetup(t, tempdir, root, files, expected)
	opts := DefaultOptions.PathSelection
	opts.LibraryFileLimit = 1
	doTest(t, files, libDirs, opts, root, expected)
}

func TestSelectFiles_Some(t *testing.T) {
	tempdir, err := ioutil.TempDir(tempDirRoot, "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tempdir) // clean up

	root := "/project_root/foo/bar/src.py"
	files := fileSet{
		"/project_root/foo/bar/src.py":          "",           // 0.
		"/project_root/foo/bar/baz/src.py":      "import bar", // 1. /project_root/bar.py
		"/project_root/bar.py":                  "import foo", // 2. /project_root/foo/__init__.py
		"/project_root/foo/__init__.py":         "import baz", // 3. /my/site-packages/baz/__init__.py
		"/my/site-packages/baz/__init__.py":     "",           // 4.
		"/project_root/foo/bar/__init__.py":     "",           // 5.
		"/project_root/foo/bar/baz/__init__.py": "",           // 6.
		"/project_root/foo/baz/src.py":          "import baz",
		"/my/site-packages/baz/src.py":          "",
	}
	expected := map[string]string{
		"/project_root/foo/bar/src.py":          "",
		"/project_root/foo/bar/baz/src.py":      "",
		"/project_root/bar.py":                  "",
		"/project_root/foo/__init__.py":         "",
		"/project_root/foo/bar/__init__.py":     "",
		"/project_root/foo/bar/baz/__init__.py": "",
		"/my/site-packages/baz/__init__.py":     "/my/site-packages",
	}
	root, files, libDirs, expected := testSetup(t, tempdir, root, files, expected)
	opts := DefaultOptions.PathSelection
	doTest(t, files, libDirs, opts, root, expected)
}

func TestSelectFiles_BreadthFirst(t *testing.T) {
	tempdir, err := ioutil.TempDir(tempDirRoot, "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tempdir) // clean up

	root := "/project_root/foo/bar/src.py"
	files := fileSet{
		"/project_root/foo/bar/src.py":  "import foo\nimport bar\nimport baz",
		"/project_root/foo/bar/foo.py":  "import car",
		"/project_root/foo/bar/bar.py":  "import car",
		"/project_root/foo/bar/baz.py":  "import car",
		"/project_root/car/__init__.py": "from . import src",
		"/project_root/car/src.py":      "import car",
	}
	expected := map[string]string{
		"/project_root/foo/bar/src.py":  "",
		"/project_root/foo/bar/foo.py":  "",
		"/project_root/foo/bar/bar.py":  "",
		"/project_root/foo/bar/baz.py":  "",
		"/project_root/car/__init__.py": "",
	}
	root, files, libDirs, expected := testSetup(t, tempdir, root, files, expected)
	opts := DefaultOptions.PathSelection
	opts.ProjectFileLimit = 5
	doTest(t, files, libDirs, opts, root, expected)
}

func TestSelectFiles_InSitePackages(t *testing.T) {
	tempdir, err := ioutil.TempDir(tempDirRoot, "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tempdir) // clean up

	root := "/my/site-packages/foo/bar.py"
	files := fileSet{
		"/my/site-packages/foo/bar.py":        "import baz",
		"/my/site-packages/foo/baz.py":        "import car",
		"/your/site-packages/car/__init__.py": "",
	}
	expected := map[string]string{
		"/my/site-packages/foo/bar.py":        "",
		"/my/site-packages/foo/baz.py":        "",
		"/your/site-packages/car/__init__.py": "/your/site-packages",
	}
	root, files, libDirs, expected := testSetup(t, tempdir, root, files, expected)
	doTest(t, files, libDirs, DefaultOptions.PathSelection, root, expected)
}

func TestSelectFiles_EggPackage(t *testing.T) {
	tempdir, err := ioutil.TempDir(tempDirRoot, "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tempdir) // clean up

	// create egg package
	eggDir := fmt.Sprintf("%s/%s", tempdir, "egg_test/site-packages")
	_, err = exec.Command("./create_egg.sh", eggDir).Output()
	require.NoError(t, err)

	libDirs := []string{eggDir}

	root := "/project_root/foo/bar/src.py"
	files := fileSet{
		"/project_root/foo/bar/src.py": "import foo\nimport mymath\nfrom mymath import multiply",
		"/project_root/foo/bar/foo.py": "from mymath import adv\nfrom mymath.adv import fib",
	}
	expected := map[string]string{
		"/project_root/foo/bar/src.py":                   "",
		"/project_root/foo/bar/foo.py":                   "",
		"/egg_test/site-packages/mymath/__init__.py":     "egg_test/site-packages",
		"/egg_test/site-packages/mymath/multiply.py":     "egg_test/site-packages",
		"/egg_test/site-packages/mymath/adv/__init__.py": "egg_test/site-packages",
		"/egg_test/site-packages/mymath/adv/fib.py":      "egg_test/site-packages",
	}
	root, files, _, expected = testSetup(t, tempdir, root, files, expected)
	opts := DefaultOptions.PathSelection
	opts.ProjectFileLimit = 10
	doTest(t, files, libDirs, opts, root, expected)
}

func TestSelectFiles_ExtractedEggPackage(t *testing.T) {
	tempdir, err := ioutil.TempDir(tempDirRoot, "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tempdir) // clean up

	// create egg package
	eggDir := fmt.Sprintf("%s/%s", tempdir, "egg_test/site-packages")
	_, err = exec.Command("./create_egg.sh", eggDir).Output()
	require.NoError(t, err)

	libDirs := []string{eggDir}

	var archive string
	for _, lib := range libDirs {
		files, err := filepath.Glob(filepath.Join(lib, "*.egg"))
		require.NoError(t, err)
		for _, f := range files {
			archive = filepath.Base(f)
			err := Unzip(f)
			require.NoError(t, err)
		}
	}

	root := "/project_root/foo/bar/src.py"
	files := fileSet{
		"/project_root/foo/bar/src.py": "import foo\nimport mymath\nfrom mymath import multiply",
		"/project_root/foo/bar/foo.py": "from mymath import adv\nfrom mymath.adv import fib",
	}
	expected := map[string]string{
		"/project_root/foo/bar/src.py":                                   "",
		"/project_root/foo/bar/foo.py":                                   "",
		"/egg_test/site-packages/" + archive + "/mymath/__init__.py":     "egg_test/site-packages",
		"/egg_test/site-packages/" + archive + "/mymath/multiply.py":     "egg_test/site-packages",
		"/egg_test/site-packages/" + archive + "/mymath/adv/__init__.py": "egg_test/site-packages",
		"/egg_test/site-packages/" + archive + "/mymath/adv/fib.py":      "egg_test/site-packages",
	}
	root, files, _, expected = testSetup(t, tempdir, root, files, expected)
	opts := DefaultOptions.PathSelection
	opts.ProjectFileLimit = 10
	doTest(t, files, libDirs, opts, root, expected)
}

func TestSelectFiles_NoInit(t *testing.T) {
	tempdir, err := ioutil.TempDir(tempDirRoot, "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tempdir) // clean up

	root := "/project_root/foo/src.py"
	files := fileSet{
		"/project_root/foo/src.py":                      "",
		"/project_root/foo/bar/src.py":                  "",
		"/project_root/foo/car/__init__.py":             "",
		"/project_root/foo/car/src.py":                  "",
		"/project_root/foo/car/bar/__init__.py":         "",
		"/project_root/foo/car/bar/end/__init__.py":     "",
		"/project_root/foo/car/bar/end/zoo/src.py":      "",
		"/project_root/foo/car/bar/end/moo/src.py":      "",
		"/project_root/foo/car/bar/end/moo/__init__.py": "",
	}
	expected := map[string]string{
		"/project_root/foo/src.py":                      "",
		"/project_root/foo/bar/src.py":                  "",
		"/project_root/foo/car/__init__.py":             "",
		"/project_root/foo/car/src.py":                  "",
		"/project_root/foo/car/bar/__init__.py":         "",
		"/project_root/foo/car/bar/end/__init__.py":     "",
		"/project_root/foo/car/bar/end/moo/src.py":      "",
		"/project_root/foo/car/bar/end/moo/__init__.py": "",
	}
	root, files, libDirs, expected := testSetup(t, tempdir, root, files, expected)
	opts := DefaultOptions.PathSelection
	doTest(t, files, libDirs, opts, root, expected)
}

func TestSelectFiles_Error(t *testing.T) {
	selection, _, err := Selector{
		StartPaths: []string{"abc/def.py"},
		Files:      nil,
		Getter:     nil,
		Opts:       SelectionOptions{},
		FileSystem: localcode.LocalFileSystem{},
		Local:      true,
	}.Select(kitectx.Background())
	require.NoError(t, err)
	// start path does not exist, so it is skipped (no error)
	assert.Len(t, selection, 0)
	_, _, err = Selector{
		StartPaths: []string{"/abc.py"},
		Files:      []*localfiles.File{&localfiles.File{Name: "/abc.py"}, &localfiles.File{Name: "relative.py"}},
		Getter:     nil,
		Opts:       SelectionOptions{},
		FileSystem: localcode.LocalFileSystem{},
		Local:      true,
	}.Select(kitectx.Background())
	assert.NotNil(t, err)
}

func TestSelectFiles_TopLevelSkipDir(t *testing.T) {
	tempdir, err := ioutil.TempDir(tempDirRoot, "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tempdir) // clean up

	root := "/project_root/foo/bar/src.py"
	files := fileSet{
		"/project_root/foo/bar/src.py":      "",           // 0.
		"/project_root/foo/bar/baz/src.py":  "import bar", // 1. /project_root/bar.py
		"/project_root/bar.py":              "import foo", // 2. /project_root/foo/__init__.py
		"/project_root/foo/__init__.py":     "import baz", // 3. /my/site-packages/baz/__init__.py
		"/my/site-packages/baz/__init__.py": "",           // 4.
		"/project_root/foo/bar/__init__.py": "",           // 5.
		"/project_root/foo/baz/src.py":      "import baz",
		"/my/site-packages/baz/src.py":      "",
	}
	root, files, libDirs, _ := testSetup(t, tempdir, root, files, nil)
	includeFn := func(path string, info localcode.FileInfo) bool {
		return false
	}
	selected, _, err := Selector{
		StartPaths:   []string{root},
		Files:        nil,
		Getter:       files,
		Opts:         DefaultOptions.PathSelection,
		Logf:         t.Logf,
		LibraryFiles: libDirs,
		FileSystem:   localcode.LocalFileSystem{Include: includeFn},
		Local:        true,
	}.Select(kitectx.Background())
	require.NoError(t, err)
	assert.Len(t, selected, 2)
}

func TestSelectFiles_NonPython(t *testing.T) {
	tempdir, err := ioutil.TempDir(tempDirRoot, "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tempdir) // clean up

	root := "/project_root/foo/bar/src.py"
	files := fileSet{
		"/project_root/foo/bar/src.py":     "from baz import src", // 0. /project_root/foo/bar/baz/src.py
		"/project_root/foo/bar/baz/src.py": "import bar",          // 1. /project_root/bar.py
		"/project_root/foo/bar/bad.go":     "package bar",         // 2.
	}
	expected := map[string]string{
		"/project_root/foo/bar/src.py":     "",
		"/project_root/foo/bar/baz/src.py": "",
	}
	root, files, libDirs, expected := testSetup(t, tempdir, root, files, expected)
	opts := DefaultOptions.PathSelection
	doTest(t, files, libDirs, opts, root, expected)
}

func TestGetInSameDir(t *testing.T) {
	fpath := "/my/working/file.py"
	files := []*localfiles.File{
		&localfiles.File{Name: "/my/working/aaa.py"},
		&localfiles.File{Name: "/my/working/a/b/file.py"},
		&localfiles.File{Name: "/my/working/a/program.py"},
		&localfiles.File{Name: "/my/working/a/file.py"},
		&localfiles.File{Name: "/my/working/b/file.py"},
		&localfiles.File{Name: "/another/file.py"},
		&localfiles.File{Name: "/my/other/file.py"},
		&localfiles.File{Name: "/my/working/file.py"},
	}
	expected := []string{
		"/my/working/file.py",
		"/my/working/aaa.py",
		"/my/working/a/file.py",
		"/my/working/a/program.py",
		"/my/working/b/file.py",
		"/my/working/a/b/file.py",
	}
	actual, err := GetInSameDir(kitectx.Background(), fpath, files)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

// https://github.com/kiteco/kiteco/issues/11947
func TestSelectFiles_FilteredProjectDir(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS only")
	}

	tempdir, err := ioutil.TempDir(tempDirRoot, "example")
	assert.NoError(t, err)
	defer os.RemoveAll(tempdir) // clean up

	libraryPath := filepath.Join(tempdir, "Library")
	projectDirPath := filepath.Join(libraryPath, "project-dir")
	err = os.MkdirAll(projectDirPath, 0700)
	require.NoError(t, err)

	projectDirs := findProjectDirs(kitectx.Background(), []string{projectDirPath}, localcode.LocalFileSystem{}, log.Printf)
	require.NotContainsf(t, projectDirs, projectDirPath, "A filtered dir must not returned as project path")
	require.NotContainsf(t, projectDirs, libraryPath, "A filtered dir must not returned as project path")
}

// Unzip unzips the input zip file in the same location as the original
//
// Source: SO link: /questions/20357223/easy-way-to-unzip-file-with-golang
func Unzip(src string) error {
	dest := src
	tmpSrc := src + ".tmp"
	if err := os.Rename(src, tmpSrc); err != nil {
		return err
	}
	defer os.Remove(tmpSrc)

	r, err := zip.OpenReader(tmpSrc)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, os.ModePerm)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, os.ModePerm)
		} else {
			os.MkdirAll(filepath.Dir(path), os.ModePerm)
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}
