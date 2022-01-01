package main

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonparser"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"

	"io/ioutil"
	"os"
	"time"
)

const maxSizeBytes = 1000000
const maxParseInterval = 1 * time.Second
const maxResolvingInterval = 1 * time.Second

// TODO cloned from pythonbatch/selectfiles.go
func getFile(path string) (*localfiles.File, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return &localfiles.File{
		Name:          path,
		HashedContent: path,
	}, nil
}

func getFileList(rootFolder string, suffix string) []*localfiles.File {
	filelist, err := source.GetFilelist(rootFolder, source.NewFileExtensionPredicate(suffix), true)
	maybeQuit(err)

	var result []*localfiles.File
	for _, filename := range filelist {
		file, err := getFile(filename)
		maybeQuit(err)
		result = append(result, file)
	}
	return result
}

func addFile(file *localfiles.File, ctx kitectx.Context, libraryPath string) *pythonbatch.SourceUnit {
	content, err := ioutil.ReadFile(file.Name)
	maybeQuit(err)

	var mod *pythonast.Module
	err = ctx.WithTimeout(pythonparser.ParseTimeout, func(ctx kitectx.Context) error {
		mod, _ = pythonparser.Parse(ctx, content, parseOpts)
		return nil
	})
	maybeQuit(err)

	return &pythonbatch.SourceUnit{
		ASTBundle: pythonstatic.ASTBundle{
			AST:         mod,
			Path:        file.Name,
			Imports:     pythonstatic.FindImports(ctx, file.Name, mod),
			LibraryPath: libraryPath,
			Windows:     false,
		},
		Hash:     file.HashedContent,
		Contents: content,
	}
}

func buildBatchManager(rm pythonresource.Manager, project *projectDescription) *pythonbatch.BatchManager {
	bi := pythonbatch.BatchInputs{
		User:    1,
		Machine: "offline-analysis",
		Graph:   rm,
	}

	parseOpts = pythonparser.Options{
		ScanOptions: pythonscanner.Options{
			ScanComments: false,
			ScanNewLines: false,
		},
		ErrorMode:   pythonparser.Recover,
		Approximate: false,
	}

	ctx := kitectx.Background()
	options := pythonbatch.Options{
		Options: pythonstatic.DefaultOptions,
		PathSelection: pythonbatch.SelectionOptions{
			ProjectFileLimit: 1000,
			LibraryFileLimit: 1000,
			SizeLimit:        maxSizeBytes,
			Parse:            parseOpts,
		},
		Local: true,
	}

	manager := pythonbatch.NewBatchManager(ctx, bi, options, nil)
	if len(project.VirtualEnvPath) > 0 {
		librarySources := getFileList(project.getLibraryFolder(), ".py")
		for _, sourceFile := range librarySources {
			sourceUnit := addFile(sourceFile, ctx, pythonbatch.GetLibraryPrefix(sourceFile.Name))
			maybeQuit(manager.Add(sourceUnit))
		}
	}

	projectSources := getFileList(project.getSourceFolder(), ".py")
	for _, sourceFile := range projectSources {
		sourceUnit := addFile(sourceFile, ctx, "")
		maybeQuit(manager.Add(sourceUnit))
	}

	//TODO exclude venv folder if SourceSubfolder is empty
	return manager
}

func buildContext(targetProject *projectDescription, rm pythonresource.Manager) func() pythonstatic.Importer {
	batchManager := buildBatchManager(rm, targetProject)
	batch, err := batchManager.Build(kitectx.Background())
	maybeQuit(err)
	return func() pythonstatic.Importer {
		return pythonstatic.Importer{PythonPaths: batch.Assembly.PythonPaths, Global: rm, Local: batch.Assembly.Sources}
	}
}
