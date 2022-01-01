package pythonbatch

import (
	"fmt"
	"log"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// BuilderLoader is a wrapper struct that contain a localcode.BuilderFunc and localcode.LoaderFunc
// for python, along with extra data structures needed to perform those tasks.
type BuilderLoader struct {
	Graph   pythonresource.Manager
	Options Options
}

// NewBuilderLoader loads a new resource manager & constructs a new BuilderLoader.
// If the ResourceManager is already loaded, you should construct the BuilderLoader via a literal and pass in these values.
func NewBuilderLoader(opts Options) (*BuilderLoader, error) {
	manager, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		return nil, err
	}

	return &BuilderLoader{
		Graph:   manager,
		Options: opts,
	}, nil
}

// Build implements localcode.BuilderFunc for python
// the caller is responsible for catch-all panic handling
func (b *BuilderLoader) Build(ctx kitectx.Context, params localcode.BuilderParams) (*localcode.BuilderResult, error) {
	ctx.CheckAbort()

	var (
		uid            = params.UserID
		machineID      = params.MachineID
		filename       = params.Filename
		files          = params.Files
		libFiles       = params.LibraryFiles
		libManager     = params.LibraryManager
		getter         = params.FileGetter
		filesystem     = params.FileSystem
		buildDurations = params.BuildDurations
		buildInfo      = params.BuildInfo
		parseInfo      = params.ParseInfo
	)

	if buildDurations == nil {
		buildDurations = make(map[string]time.Duration)
	}
	if buildInfo == nil {
		buildInfo = make(map[string]int)
	}

	logf := func(msg string, vals ...interface{}) {
		log.Printf("pythonlocal.Builder (%d, %s): %s", uid, machineID, fmt.Sprintf(msg, vals...))
	}

	logf("starting build with filename: %s", filename)

	// ensure current filename is included in the files
	var hasFile bool
	for _, file := range files {
		if file.Name == filename {
			hasFile = true
			break
		}
	}
	if !hasFile {
		files = append(files, &localfiles.File{
			Name:          filename,
			HashedContent: filename,
		})
	}

	// Select files to index
	ts := time.Now()
	sources, missing, err := Selector{
		StartPaths:     []string{filename},
		Files:          files,
		LibraryFiles:   libFiles,
		LibraryManager: libManager,
		Opts:           b.Options.PathSelection,
		Graph:          b.Graph,
		Getter:         getter,
		Logf:           logf,
		FileSystem:     filesystem,
		Local:          params.Local,
		BuildDurations: buildDurations,
		ParseInfo:      parseInfo,
	}.Select(ctx)
	if err != nil {
		logf("SelectFiles error: %v", err)
		return nil, err
	}
	buildInfo["selected_files"] = len(sources)
	if len(sources) == 0 {
		if len(missing) > 0 {
			err = fmt.Errorf("missing file hashes (missing %d, originally were %d)", len(missing), len(files))
			logf(err.Error())
			return nil, err
		}
		err = fmt.Errorf("no files selected (originally were %d)", len(files))
		logf(err.Error())
		return nil, err
	}

	logf("selected %d (of %d) files", len(sources), len(files))
	selectFilesTime := time.Since(ts)
	logf("file selection took %s", selectFilesTime)
	buildDurations["file_selection"] = selectFilesTime
	selectFilesDuration.RecordDuration(selectFilesTime)
	ts = time.Now()

	// Add each file to the batch
	bi := BatchInputs{
		User:    uid,
		Machine: machineID,
		Graph:   b.Graph,
	}
	manager := NewBatchManager(ctx, bi, b.Options, buildDurations)
	for _, source := range sources {
		manager.Add(source)
	}
	managerAddTime := time.Since(ts)
	logf("manager add took %s", managerAddTime)
	buildDurations["manager_add"] = managerAddTime
	managerAddDuration.RecordDuration(managerAddTime)

	var result *Batch
	buildFunc := func(ctx kitectx.Context) error {
		ctx.CheckAbort()

		var err error

		ts = time.Now()
		result, err = manager.Build(ctx)
		if err != nil {
			logf("error building batch: %v", err)
			return err
		}
		managerBuildTime := time.Since(ts)
		logf("manager build took %s", managerBuildTime)
		buildDurations["manager_build"] = managerBuildTime
		managerBuildDuration.RecordDuration(managerBuildTime)

		return nil
	}

	if b.Options.BuildTimeout != time.Duration(0) {
		err = ctx.WithTimeout(b.Options.BuildTimeout, buildFunc)
	} else {
		err = buildFunc(ctx)
	}
	if err != nil {
		if _, ok := err.(kitectx.ContextExpiredError); ok {
			logf("context abort error: %v", err)
		}
		return nil, err
	}

	ts = time.Now()

	logf("constructed %d definitions, %d documentation, %d method patterns and %d tokens",
		len(result.Definitions), len(result.Docs), len(result.Methods), len(result.InvertedIndex))

	fileHashes := make(map[string]string)
	libraryHashes := make(map[string]string)
	missingFileHashes := make(map[string]bool)
	missingLibraryHashes := make(map[string]bool)
	addedLibDirsMap := make(map[string]bool)
	for _, source := range sources {
		if source.ASTBundle.LibraryPath != "" {
			libraryHashes[source.ASTBundle.Path] = source.Hash
			addedLibDirsMap[source.ASTBundle.LibraryPath] = true
			libManager.MarkUsed(source.ASTBundle.LibraryPath)
		} else {
			fileHashes[source.ASTBundle.Path] = source.Hash
		}
	}
	for hash, library := range missing {
		if library {
			missingLibraryHashes[hash] = true
		} else {
			missingFileHashes[hash] = true
		}
	}

	var addedFiles, addedLibs []*localfiles.File
	for name, hash := range fileHashes {
		addedFiles = append(addedFiles, &localfiles.File{
			Name:          name,
			HashedContent: hash,
		})
	}
	for name, hash := range libraryHashes {
		addedLibs = append(addedLibs, &localfiles.File{
			Name:          name,
			HashedContent: hash,
		})
	}

	var addedLibDirs []string
	for d := range addedLibDirsMap {
		addedLibDirs = append(addedLibDirs, d)
	}

	artifactMetadata := pythonlocal.ArtifactMetadata{
		OriginatingFilename:  filename,
		FileHashes:           fileHashes,
		MissingHashes:        missingFileHashes,
		LibraryHashes:        libraryHashes,
		MissingLibraryHashes: missingLibraryHashes,
	}

	symbolIndex := &pythonlocal.SymbolIndex{
		PythonPaths:      result.Assembly.PythonPaths,
		SourceTree:       result.Assembly.Sources,
		ValuesCount:      result.ValuesCount,
		ArgSpecMap:       result.ArgSpecs,
		DocumentationMap: result.Docs,
		DefinitionMap:    result.Definitions,
		MethodMap:        result.Methods,
		ArtifactMetadata: artifactMetadata,
		LocalBuildTime:   time.Now(),
	}

	return &localcode.BuilderResult{
		Files:         addedFiles,
		LibraryFiles:  addedLibs,
		LibraryDirs:   addedLibDirs,
		MissingHashes: missing,
		LocalArtifact: symbolIndex,
	}, nil
}

// Load implements localcode.LoaderFunc for python
func (b *BuilderLoader) Load(getter localcode.Getter) (localcode.Cleanuper, error) {
	// deprecated
	return nil, nil
}
