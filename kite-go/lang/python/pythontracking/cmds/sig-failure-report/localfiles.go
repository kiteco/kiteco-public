package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonast"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/localfiles"
)

type unresolvedMatchType string

const (
	noMatch             = ""
	indexedFile         = "indexed_file"           // An unresolved module matches the name of an indexed file.
	indexedFileNotFound = "indexed_file_not_found" // The indexed file could not be retrieved from storage.
	libraryFile         = "library_file"           // An unresolved module matches the name of a library file.
	laterLibraryFile    = "later_library_file"     // The timestamp of the library file is later than that of the log.
	localFile           = "local_file"             // An unresolved module matches the name of a local un-indexed file.
	laterLocalFile      = "later_local_file"       // The timestamp of the local file is later than that of the log.
)

type unresolvedMatch struct {
	Type       unresolvedMatchType `json:"type"`
	Filename   string              `json:"filename"`
	IsAncestor bool                `json:"is_ancestor"`
}

type unresolvedImport struct {
	Name    string            `json:"name"`
	Matches []unresolvedMatch `json:"matches"`
}

// UnresolvedFileAnalysis contains information about local files whose names match unresolved imported modules.
type UnresolvedFileAnalysis struct {
	Performed bool               `json:"performed"`
	Imports   []unresolvedImport `json:"imports"`
}

func doUnresolvedFileAnalysis(track *pythontracking.Event, timestamp time.Time, ctx *python.Context, manager *localfiles.FileManager) (UnresolvedFileAnalysis, error) {
	files, err := manager.List(ctx.User, ctx.Machine)
	if err != nil {
		return UnresolvedFileAnalysis{}, err
	}
	unresolvedModules := getUnresolvedImportedModules(ctx)

	analysis := UnresolvedFileAnalysis{Performed: true}

	for _, module := range unresolvedModules {
		importAnalysis := unresolvedImport{Name: module}
		for _, file := range files {
			matchType, containingDir := getUnresolvedImportMatch(module, file, track, timestamp, ctx)
			if matchType != noMatch {
				importAnalysis.Matches = append(importAnalysis.Matches, unresolvedMatch{
					Type:       matchType,
					Filename:   file.Name,
					IsAncestor: strings.HasPrefix(track.Filename, containingDir),
				})
			}
		}
		analysis.Imports = append(analysis.Imports, importAnalysis)
	}
	return analysis, nil
}

func getUnresolvedImportMatch(module string, file *localfiles.File, track *pythontracking.Event, timestamp time.Time, ctx *python.Context) (unresolvedMatchType, string) {
	timestampIsLater := file.CreatedAt.After(timestamp)
	containingDir := trimModuleSuffix(module, file.Name)

	if containingDir == "" {
		return noMatch, ""
	}

	if pythonbatch.GetLibraryPrefix(file.Name) != "" {
		if timestampIsLater {
			return laterLibraryFile, containingDir
		}
		return libraryFile, containingDir
	}

	var indexedHash string
	for indexedFilename, hash := range track.ArtifactMeta.FileHashes {
		if file.Name == indexedFilename {
			indexedHash = hash
			break
		}
	}
	if indexedHash != "" {
		var hashIsMissing bool
		if ctx.LocalIndex != nil {
			_, hashIsMissing = ctx.LocalIndex.ArtifactMetadata.MissingHashes[indexedHash]
		}
		if hashIsMissing {
			return indexedFileNotFound, containingDir
		}
		return indexedFile, containingDir
	}

	if timestampIsLater {
		return laterLocalFile, containingDir
	}

	return localFile, containingDir
}

// getUnresolvedImportedModules returns a list of unresolved modules imported in the context's buffer.
// e.g. if a file contains
//   from foo.bar import baz
//   import ham as spam
// ...then this will return ["foo.bar", "ham"] (assuming that those modules were not resolved).
func getUnresolvedImportedModules(ctx *python.Context) []string {
	importMap := make(map[string]struct{})
	pythonast.Inspect(ctx.AST, func(n pythonast.Node) bool {
		if n, ok := n.(*pythonast.DottedExpr); ok {
			if ref := ctx.Resolved.References[n.Names[0]]; ref == nil {
				importMap[n.Join()] = struct{}{}
			}
		}
		return true
	})
	var imports []string
	for i := range importMap {
		imports = append(imports, i)
	}
	return imports
}

func trimModuleSuffix(module string, filename string) string {
	withSlashes := strings.Replace(module, ".", "/", -1)

	if moduleSuffix := fmt.Sprintf("/%s.py", withSlashes); strings.HasSuffix(filename, moduleSuffix) {
		return strings.TrimSuffix(filename, moduleSuffix)
	}

	if packageSuffix := fmt.Sprintf("/%s/__init__.py", withSlashes); strings.HasSuffix(filename, packageSuffix) {
		return strings.TrimSuffix(filename, packageSuffix)
	}

	return ""
}
