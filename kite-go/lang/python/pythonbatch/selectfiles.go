package pythonbatch

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/filters"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// maxParseErrors specifies the max number of parse errors to record
const maxParseErrors = 5

// SelectionOptions represents the criteria by which an ancestor directory is selected for indexing
type SelectionOptions struct {
	ProjectFileLimit int
	LibraryFileLimit int
	SizeLimit        int
	Parse            pythonparser.Options
}

// ErrNonAbsolutePath is returned when the Selector is passed non-absolute paths
type ErrNonAbsolutePath string

// Error implements error
func (e ErrNonAbsolutePath) Error() string {
	return fmt.Sprintf("received a non-absolute path: %s", string(e))
}

// GetInSameDir collects the names of those files that are descendents of the directory containing fpath.
// If fpath is not in the provided file listing, an error is returned.
// The results are returned in order of proximity to fpath, starting with fpath itself.
// The results also don't include any "library" paths (as per GetLibraryPrefix), possibly excepting fpath itself.
// TODO: Deprecate
// GetInSameDir is no longer used by file selection.
func GetInSameDir(ctx kitectx.Context, fpath string, files []*localfiles.File) ([]string, error) {
	ctx.CheckAbort()

	prefix := slashed(path.Dir(fpath))
	if strings.HasPrefix("/windows/", prefix) {
		prefix = strings.ToLower(prefix)
	}

	// the first element is reserved for fpath
	children := []string{""}
	var libCount int
	for _, file := range files {
		ctx.CheckAbort()

		if file.Name == fpath {
			children[0] = fpath
			continue
		}

		// don't include library paths
		if GetLibraryPrefix(file.Name) != "" {
			libCount++
			continue
		}

		if strings.HasPrefix(file.Name, prefix) {
			children = append(children, file.Name)
		}
	}
	log.Printf("Found %d of %d children in directory, %d library files", len(children), len(files), libCount)

	if children[0] == "" {
		return nil, errors.New("requested file path not found in dir")
	}

	// sort everything except the first element which is reserved for fpath
	// first by depth, then by lexicographic ordering (split on /)
	sort.Slice(children[1:], func(i, j int) bool {
		ctx.CheckAbort()

		iName := children[i+1]
		jName := children[j+1]
		if iDepth, jDepth := strings.Count(iName, "/"), strings.Count(jName, "/"); iDepth != jDepth {
			return iDepth < jDepth
		}

		var iPart, jPart string
		for len(iName) > 0 && len(jName) > 0 && iPart == jPart {
			if idx := strings.Index(iName, "/"); idx != -1 {
				iPart = iName[:idx]
				iName = iName[idx+1:]
			} else {
				iPart = iName
				iName = ""
			}
			if idx := strings.Index(jName, "/"); idx != -1 {
				jPart = jName[:idx]
				jName = jName[idx+1:]
			} else {
				jPart = jName
				jName = ""
			}
		}
		return iPart < jPart
	})

	return children, nil
}

// Selector encapsulates input & state for file selection; see Selector.Select
type Selector struct {
	// required inputs:
	Opts           SelectionOptions
	Getter         localcode.FileGetter
	StartPaths     []string
	Files          []*localfiles.File
	LibraryFiles   []string
	LibraryManager localcode.LibraryManager
	FileSystem     localcode.FileSystem
	Local          bool
	// optional input:
	Graph          pythonresource.Manager
	Logf           func(msg string, vals ...interface{})
	BuildDurations map[string]time.Duration
	ParseInfo      *localcode.ParseInfo
	parseErrors    errors.Errors

	// state:
	fileMap      map[string]*localfiles.File
	projectPaths []string
	libraryPaths []string

	work      []fileToAdd
	selection map[string]*SourceUnit
	missing   map[string]bool

	pathToEggs    map[string][]eggPath
	packageToEggs map[string]string
	zipReaders    map[string]*zip.Reader
	zipFiles      map[string][]byte
}

func (s *Selector) logf(msg string, vals ...interface{}) {
	if s.Logf == nil {
		return
	}
	s.Logf(msg, vals...)
}

// ErrFileLimitExceeded is returned by the WalkFunc when the project file and library limits
// have been reached as a signal to stop walking.
var ErrFileLimitExceeded = errors.New("project and library file limits exceeded")

// Select returns a map containing files selected via import analysis and a map of missing hashes
func (s Selector) Select(ctx kitectx.Context) (map[string]*SourceUnit, map[string]bool, error) {
	ctx.CheckAbort()

	if s.StartPaths == nil {
		return nil, nil, errors.Errorf("must have a start path")
	}

	s.pathToEggs = make(map[string][]eggPath)
	s.packageToEggs = make(map[string]string)
	s.zipReaders = make(map[string]*zip.Reader)
	s.zipFiles = make(map[string][]byte)

	if s.ParseInfo == nil {
		s.ParseInfo = &localcode.ParseInfo{}
	}
	if s.BuildDurations == nil {
		s.BuildDurations = make(map[string]time.Duration)
	}

	defer func() {
		if s.parseErrors != nil {
			for _, e := range s.parseErrors.Slice() {
				s.ParseInfo.ParseErrors = append(s.ParseInfo.ParseErrors, e.Error())
			}
		}
	}()

	if len(s.StartPaths) == 0 {
		// nothing to select
		return nil, nil, nil
	}

	if err := s.preprocessFiles(ctx); err != nil {
		return nil, nil, err
	}

	s.inferProjectAndLibraryPaths(ctx)

	s.selection = make(map[string]*SourceUnit)
	s.missing = make(map[string]bool)

	projectRemaining := s.Opts.ProjectFileLimit
	libraryRemaining := s.Opts.LibraryFileLimit

	// add start file
	startFile := s.StartPaths[0]
	if file, err := s.getFile(startFile); err == nil {
		s.work = append(s.work, fileToAdd{file: file, libraryPath: ""})
	}

	var children, libCount int
	initDirs := make(map[string]struct{})
	log.Printf("Starting walk of project: %s", path.Dir(startFile))
	ts := time.Now()
	err := s.FileSystem.Walk(ctx, path.Dir(startFile), func(filePath string, fi localcode.FileInfo, err error) error {
		ctx.CheckAbort()

		if err != nil {
			if err == localcode.ErrLibDir {
				libCount++
				// skip library directories
				return localcode.ErrSkipDir
			}
			return nil
		}

		if fi.IsDir {
			// check if dir has no init but parent does
			parent := path.Dir(filePath)
			if _, exists := initDirs[parent]; exists {
				if _, err := s.FileSystem.Stat(path.Join(filePath, "__init__.py")); err != nil {
					return localcode.ErrSkipDir
				}
			}
			// file selection only adds files, continue walking
			return nil
		}

		// stop adding files if limits exceeded
		if projectRemaining <= 0 && libraryRemaining <= 0 {
			return ErrFileLimitExceeded
		}

		if !strings.HasSuffix(filePath, ".py") {
			return nil
		}

		if strings.HasSuffix(filePath, "__init__.py") {
			initDirs[path.Dir(filePath)] = struct{}{}
		}

		// add imports for current file for processing
		s.addWork(ctx, &projectRemaining, &libraryRemaining)

		if file, err := s.getFile(filePath); err == nil {
			children++
			s.work = append(s.work, fileToAdd{file: file, libraryPath: ""})
		}

		return nil
	})
	// process remaining work
	s.addWork(ctx, &projectRemaining, &libraryRemaining)

	walkTime := time.Since(ts)
	log.Printf("Walk project took %s", walkTime)
	s.BuildDurations["walk_project"] = walkTime
	if err != nil && err != localcode.ErrSkipDir && err != ErrFileLimitExceeded {
		return nil, nil, err
	}

	for path, source := range s.selection {
		if source == nil {
			delete(s.selection, path)
		}
	}

	log.Printf("Found %d children in directory, %d library files", children, libCount)
	return s.selection, s.missing, nil
}

func (s *Selector) addWork(ctx kitectx.Context, projectRemaining, libraryRemaining *int) {
	for len(s.work) > 0 && (*projectRemaining > 0 || *libraryRemaining > 0) {
		ctx.CheckAbort()

		work := s.work[0]
		s.work = s.work[1:]

		if work.libraryPath == "" && *projectRemaining > 0 && s.addFile(ctx, work) {
			*projectRemaining--
		} else if work.libraryPath != "" && *libraryRemaining > 0 && s.addFile(ctx, work) {
			*libraryRemaining--
		}
	}
}

func (s *Selector) getFile(path string) (*localfiles.File, error) {
	return &localfiles.File{
		Name:          path,
		HashedContent: path,
	}, nil
}

// statFile should only be called when statEggs should also be called
func (s *Selector) statFile(pkgPath, pkgName, importPath string, suffixes []string) (*localfiles.File, error) {
	path := path.Join(pkgPath, pkgName, importPath)
	var err error
	for _, suffix := range suffixes {
		filePath := path + suffix
		if _, err = s.FileSystem.Stat(filePath); err == nil {
			return s.getFile(filePath)
		}
		if eggPath, err := s.statEggs(filePath, pkgPath, pkgName, importPath+suffix); err == nil {
			return s.getFile(eggPath)
		}
	}
	return nil, err
}

func (s *Selector) statEggs(fullPath, pkgPath, pkgName, importPath string) (string, error) {
	eggs := s.findEggs(pkgPath)
	pkgEgg := s.packageToEggs[path.Join(pkgPath, pkgName)]
	for _, egg := range eggs {
		if egg.isDir {
			name := path.Join(egg.filename, pkgName, importPath)
			if _, err := s.FileSystem.Stat(name); err != nil {
				return "", err
			}
			return name, nil
		}

		if pkgEgg != egg.filename {
			continue
		}
		reader, ok := s.zipReaders[egg.filename]
		if !ok {
			continue
		}
		name := path.Join(pkgName, importPath)
		for _, f := range reader.File {
			if f.FileHeader.Name == name {
				r, err := f.Open()
				if err != nil {
					log.Println(err)
					return "", err
				}
				defer r.Close()
				contents, err := ioutil.ReadAll(r)
				if err != nil {
					log.Println(err)
					return "", err
				}
				s.zipFiles[fullPath] = contents
				return fullPath, nil
			}
		}
	}
	return "", os.ErrNotExist
}

type eggPath struct {
	isDir    bool
	filename string
}

func (s *Selector) findEggs(dir string) []eggPath {
	if eggs, ok := s.pathToEggs[dir]; ok {
		return eggs
	}

	files, err := s.FileSystem.Glob(dir, "*.egg")
	if err != nil {
		return nil
	}

	var eggs []eggPath
	for _, filename := range files {
		if !strings.HasSuffix(filename, ".egg") {
			continue
		}
		if !path.IsAbs(filename) {
			filename = path.Join(dir, filename)
		}
		var isDir bool
		if info, err := os.Stat(filename); err == nil {
			if info.IsDir() {
				isDir = true
			} else {
				if err := s.addEggArchive(dir, filename); err != nil {
					log.Println(err)
					continue
				}
			}
			eggs = append(eggs, eggPath{
				filename: filename,
				isDir:    isDir,
			})
		}
	}
	s.pathToEggs[dir] = eggs
	return eggs
}

func (s *Selector) addEggArchive(dir, filename string) error {
	contents, err := s.Getter.Get(filename)
	if err != nil {
		return err
	}
	reader, err := zip.NewReader(bytes.NewReader(contents), int64(len(contents)))
	if err != nil {
		return err
	}
	s.zipReaders[filename] = reader
	for _, rf := range reader.File {
		entry := strings.Split(rf.FileHeader.Name, "/")
		if len(entry) != 2 {
			continue
		}
		if entry[1] != "__init__.py" {
			continue
		}
		s.packageToEggs[path.Join(dir, entry[0])] = filename
	}
	return nil
}

type fileToAdd struct {
	file *localfiles.File

	// invariant: (libraryPath == "" || strings.HasPrefix(slashed(file.Name), slashed(libraryPath)))
	// this invariant will be necessary to compute "pseudo-paths" for the Libraries SourceTree down the line
	libraryPath string
}

// addFile adds the given file, and queues up discovered dependencies, returning true if the file was added
func (s *Selector) addFile(ctx kitectx.Context, work fileToAdd) bool {
	ctx.CheckAbort()

	if _, ok := s.selection[work.file.Name]; ok {
		return false // file already added, so don't add it again
	}
	s.logf("adding %s", work.file.Name)

	// make sure we don't waste time trying again if if we encounter an error
	s.selection[work.file.Name] = nil

	var err error
	contents, exists := s.zipFiles[work.file.HashedContent]
	if !exists {
		contents, err = s.Getter.Get(work.file.HashedContent)
		if err != nil {
			s.missing[work.file.HashedContent] = work.libraryPath != ""
			s.logf(errors.Wrapf(err, "error retrieving file %s", work.file.Name).Error())
		}
	}
	if s.Opts.SizeLimit > 0 && len(contents) > s.Opts.SizeLimit {
		return false
	}

	var mod *pythonast.Module
	var imps []pythonstatic.ImportPath

	opts := s.Opts.Parse
	opts.ScanOptions.Label = work.file.Name
	ts := time.Now()
	err = kitectx.Background().WithTimeout(pythonparser.ParseTimeout, func(ctx kitectx.Context) (err error) {
		mod, err = pythonparser.Parse(ctx, contents, opts)
		return
	})
	parseTime := time.Since(ts)
	if err != nil {
		errMsg := errors.Wrapf(err, "error parsing file %s", work.file.Name)
		s.logf(errMsg.Error())
		if kitectx.IsDeadlineExceeded(err) {
			s.ParseInfo.ParseTimeouts++
		} else {
			s.ParseInfo.ParseFailures++
		}
		if s.parseErrors == nil || s.parseErrors.Len() < maxParseErrors {
			s.parseErrors = errors.Append(s.parseErrors, errMsg)
		}
	}
	if mod == nil {
		if err == nil {
			errMsg := errors.Errorf("nil module for file %s", work.file.Name)
			s.logf(errMsg.Error())
			s.ParseInfo.ParseFailures++
			if s.parseErrors == nil || s.parseErrors.Len() < maxParseErrors {
				s.parseErrors = errors.Append(s.parseErrors, errMsg)
			}
		}
		mod = &pythonast.Module{}
	} else {
		s.ParseInfo.ParseDurations = append(s.ParseInfo.ParseDurations, parseTime)
		imps = pythonstatic.FindImports(ctx, work.file.Name, mod)
	}

	s.selection[work.file.Name] = &SourceUnit{
		ASTBundle: pythonstatic.ASTBundle{
			AST:         mod,
			Imports:     imps,
			Path:        work.file.Name,
			LibraryPath: work.libraryPath,
			Windows:     strings.HasPrefix(work.file.Name, "/windows/"),
		},
		Hash:     work.file.HashedContent,
		Contents: contents,
	}

	// add work for all the dependencies of work.file
	for _, imp := range imps {
		if depFile, depLibraryPath := s.handleImport(ctx, imp, work.libraryPath); depFile != nil {
			s.work = append(s.work, fileToAdd{file: depFile, libraryPath: depLibraryPath})
		}
	}

	return true
}

// ErrHandleImport is used when handleImport receives a non-absolute path
var ErrHandleImport = errors.New("handleImport received a non-absolute path")

func (s *Selector) handlePathImport(ctx kitectx.Context, pkgPath string, imp pythonstatic.ImportPath) *localfiles.File {
	ctx.CheckAbort()

	var pkgName, importPath string
	if !imp.Path.Empty() {
		pkgName = imp.Path.Parts[0]
		if len(imp.Path.Parts) > 1 {
			importPath = path.Join(imp.Path.Parts[1:]...)
		}
	}

	suffixes := []string{".py", "/__init__.py"}

	if imp.Extract != "" {
		importPath = path.Join(importPath, imp.Extract)
		if file, err := s.statFile(pkgPath, pkgName, importPath, suffixes); err == nil {
			return file
		}
	}
	if file, err := s.statFile(pkgPath, pkgName, importPath, suffixes); err == nil {
		return file
	}

	return nil
}

// handleHierarchicalImport tries to resolve the given import from every ancestor directory of path, stopping after no __init__.py file is found
// when combined with handleRelativeImport, this is equivalent to the heuristic in SourceTree.sourcePackageSearch
// TODO(naman) better would be to attempt to infer the Python3 interpreter's working directory, since that is the only root that would produce a valid import
func (s *Selector) handleHierarchicalImport(ctx kitectx.Context, dir string, imp pythonstatic.ImportPath) *localfiles.File {
	for {
		// don't check path itself, since it's handled by the relative import check
		dir = path.Dir(dir)
		if file := s.handlePathImport(ctx, dir, imp); file != nil {
			return file
		}

		if _, err := s.FileSystem.Stat(path.Join(dir, "__init__.py")); err != nil || dir == "/" {
			// no __init__.py or already at root: stop going up the directory tree
			break
		}
	}
	return nil
}

func (s *Selector) handleRelativeImport(ctx kitectx.Context, imp pythonstatic.ImportPath, path, libraryPath string) *localfiles.File {
	ctx.CheckAbort()

	// If the import is from a library file, we must not include a file that is outside the corresponding
	// library directory (site-packages), since this will violates the `fileToAdd.libraryPath` invariant.
	if libraryPath == "" || strings.HasPrefix(slashed(path), slashed(libraryPath)) {
		if file := s.handlePathImport(ctx, path, imp); file != nil {
			return file
		}
	}
	return nil
}

func (s *Selector) handleImport(ctx kitectx.Context, imp pythonstatic.ImportPath, libraryPath string) (*localfiles.File, string) {
	ctx.CheckAbort()

	dir := path.Dir(imp.Origin)
	for i := 1; i < imp.RelativeDots; i++ {
		dir = path.Dir(dir)
	}

	if imp.RelativeDots > 0 {
		// handle explicit relative import
		if file := s.handleRelativeImport(ctx, imp, dir, libraryPath); file != nil {
			return file, libraryPath
		}
		return nil, ""
	}

	// handle implicit relative import
	if file := s.handleRelativeImport(ctx, imp, dir, libraryPath); file != nil {
		return file, libraryPath
	}

	// let project files take precedent over global values
	for _, p := range s.projectPaths {
		if file := s.handlePathImport(ctx, p, imp); file != nil {
			return file, ""
		}
	}

	// hierarchical import from project directory only if we're not in a library
	if libraryPath == "" {
		if file := s.handleHierarchicalImport(ctx, dir, imp); file != nil {
			return file, ""
		}
	}

	// but global values take precedence over locally installed libraries, since we know more about globals
	if s.Graph != nil && len(s.Graph.DistsForPkg(imp.Path.Head())) > 0 {
		// TODO(naman) should we do full traversal here?
		return nil, "" // global toplevel found
	}

	for _, p := range s.libraryPaths {
		if file := s.handlePathImport(ctx, p, imp); file != nil {
			return file, p
		}
	}

	return nil, ""
}

// preprocessFiles updates filenames if necessary, and computes s.fileMap
func (s *Selector) preprocessFiles(ctx kitectx.Context) error {
	s.fileMap = make(map[string]*localfiles.File)
	for _, file := range s.Files {
		ctx.CheckAbort()

		if !path.IsAbs(file.Name) {
			return ErrNonAbsolutePath(file.Name)
		}
		if l := lang.FromFilename(file.Name); l != lang.Python {
			continue
		}
		s.fileMap[file.Name] = file
	}

	return nil
}

// inferProjectAndLibraryPaths computes s.projectPaths and s.libraryPaths
func (s *Selector) inferProjectAndLibraryPaths(ctx kitectx.Context) {
	ctx.CheckAbort()

	s.projectPaths = findProjectDirs(ctx, s.StartPaths, s.FileSystem, s.logf)

	//TODO(hrysoula): how are library files updated? Files is not updated by watcher
	if s.LibraryManager == nil {
		s.libraryPaths = s.LibraryFiles
		return
	}
	ts := time.Now()
	s.LibraryManager.AddProject(s.projectPaths)
	addProjectTime := time.Since(ts)
	s.logf("add project took %s", addProjectTime)
	s.BuildDurations["add_project"] = addProjectTime
	s.libraryPaths = s.LibraryManager.Dirs()
}

// findProjectDirs collects project directories for a slice of start paths
// Because the return value is passed to s.LibraryManager.AddProject, the
// directories are filtered with filters.isFilteredDir.
func findProjectDirs(ctx kitectx.Context, startPaths []string, fs localcode.FileSystem, logf func(msg string, vals ...interface{})) []string {
	var projectPaths []string

	root := "/windows"
	if runtime.GOOS != "windows" {
		root = "/"
		// windows root is /windows/<volume>, which will be added below
		projectPaths = append(projectPaths, root)
	}

	seen := map[string]struct{}{root: struct{}{}}

	// TODO(naman) we could alternatively take the intersection over the
	// non-init-containing ancestors of StartPaths instead of the union as we do here.
	for _, start := range startPaths {
		ctx.CheckAbort()

		for dir := path.Dir(start); dir != root; dir = path.Dir(dir) {
			ctx.CheckAbort()

			if filters.IsFilteredDir(runtime.GOOS, dir) {
				continue
			}

			if _, ok := seen[dir]; ok {
				break
			}
			seen[dir] = struct{}{}

			_, err := fs.Stat(path.Join(dir, "__init__.py"))
			if err != nil && os.IsNotExist(err) {
				logf("project path: %s", dir)
				projectPaths = append(projectPaths, dir)
			}
		}
	}
	return projectPaths
}

// GetLibraryPrefix computes the prefix of path corresponding to the library directory (e.g. /.../site-packages)
// if no such prefix is found, it returns ""
func GetLibraryPrefix(filepath string) string {
	names := []string{
		"/site-packages/",
		"/dist-packages/",
	}
	for _, name := range names {
		idx := strings.Index(filepath, name)
		if idx != -1 {
			return path.Clean(filepath[:idx+len(name)])
		}
	}
	return ""
}

func slashed(filepath string) string {
	if !strings.HasSuffix(filepath, "/") {
		return filepath + "/"
	}
	return filepath
}
