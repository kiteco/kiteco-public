package servercontext

import (
	"fmt"
	"log"
	"time"

	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonanalyzer"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonbatch"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonenv"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonlocal"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonscanner"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonstatic"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/localfiles"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/pkg/errors"
)

var (
	// DefaultBucketsByRegion for getting the local files
	DefaultBucketsByRegion = map[string]string{
		"us-west-1": "kite-local-content",
		"us-west-2": "kite-local-content-us-west-2",
		"us-east-1": "kite-local-content-us-east-1",
		"eastus":    "kite-local-content-us-east-1",
		"westus2":   "kite-local-content",
	}
)

// Recreator can recreate a lang/python.Context from a callee tracking event.
type Recreator struct {
	Services        *python.Services
	BucketsByRegion map[string]string
}

// NewRecreator creates a Recreator given a map of AWS regions to S3 buckets that hold local files, and a directory
// used to cache the local files.
func NewRecreator(bucketsByRegion map[string]string) (*Recreator, error) {
	services, err := createServices()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create services")
	}
	return &Recreator{
		Services:        services,
		BucketsByRegion: bucketsByRegion,
	}, nil
}

// RecreateContext creates a lang.python.Context given inputs from a logged event.
func (c *Recreator) RecreateContext(track *pythontracking.Event, createLocalIndex bool) (*python.Context, error) {
	var localSource *pythonenv.SourceTree
	var localIndex *pythonlocal.SymbolIndex
	var pythonPaths map[string]struct{}
	if createLocalIndex {
		var err error
		localIndex, err = c.createLocalIndex(track)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create local index")
		}
		if localIndex != nil {
			localSource = localIndex.SourceTree
			pythonPaths = localIndex.PythonPaths
		}
	}

	filename := track.Filename
	if filename == "" {
		filename = "/src.py"
	}

	importer := pythonstatic.Importer{
		Path:        filename,
		PythonPaths: pythonPaths,
		Global:      c.Services.ResourceManager,
		Local:       localSource,
	}

	resolver := pythonanalyzer.NewResolverUsingImporter(importer, pythonanalyzer.Options{
		User:    track.UserID,
		Machine: track.MachineID,
		Path:    filename,
	})

	buffer := []byte(track.Buffer)
	incrLexer := pythonscanner.NewIncrementalFromBuffer(buffer, pythonscanner.Options{
		ScanComments: true,
		ScanNewLines: true,
	})

	return python.NewContext(kitectx.Background(), python.ContextInputs{
		User:            track.UserID,
		Machine:         track.MachineID,
		Buffer:          buffer,
		Cursor:          int64(track.Offset),
		FileName:        filename,
		Importer:        importer,
		Resolver:        resolver,
		IncrLexer:       incrLexer,
		LocalIndex:      localIndex,
		EventSource:     "recreator",
		ResolverTimeout: 10 * time.Second,
	})
}

func (c *Recreator) createLocalIndex(track *pythontracking.Event) (*pythonlocal.SymbolIndex, error) {
	// If the original event had no local index, don't create one.
	if track.ArtifactMeta.OriginatingFilename == "" {
		return nil, nil
	}

	var files []*localfiles.File
	var filenames []string
	for filename, hash := range track.ArtifactMeta.FileHashes {
		files = append(files, &localfiles.File{
			UserID:        track.UserID,
			Machine:       track.MachineID,
			Name:          filename,
			HashedContent: hash,
		})
		filenames = append(filenames, filename)
	}

	bucket, found := c.BucketsByRegion[track.Region]
	if !found {
		return nil, fmt.Errorf("no bucket found for region: %s", track.Region)
	}
	getter := NewLocalFileGetter(bucket, files)

	builderParams := localcode.BuilderParams{
		UserID:     track.UserID,
		MachineID:  track.MachineID,
		Filename:   track.ArtifactMeta.OriginatingFilename,
		FileSystem: newMemFS(files),
		Files:      files,
		FileGetter: getter,
	}
	res, err := c.Services.BuilderLoader.Build(kitectx.Background(), builderParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build artifact")
	}
	return res.LocalArtifact.(*pythonlocal.SymbolIndex), nil
}

// GetIndexedFiles returns a (local filename => file contents) map of the files in the local index.
func (c *Recreator) GetIndexedFiles(track *pythontracking.Event) (map[string]string, error) {
	indexedFiles := make(map[string]string)

	bucket, found := c.BucketsByRegion[track.Region]
	if !found {
		return nil, fmt.Errorf("no bucket found for region: %s", track.Region)
	}
	getter := NewLocalFileGetter(bucket, nil)

	var hashes [][]byte
	for _, hash := range track.ArtifactMeta.FileHashes {
		hashes = append(hashes, []byte(hash))
	}

	for filename, hash := range track.ArtifactMeta.FileHashes {
		contents, err := getter.GetHash(hash)
		if err != nil {
			indexedFiles[filename] = err.Error()
		} else {
			indexedFiles[filename] = string(contents)
		}
	}
	return indexedFiles, nil
}

func createServices() (*python.Services, error) {
	serviceOptions := python.DefaultServiceOptions

	log.Printf("Creating resource manager")
	resourceManager, errc := pythonresource.NewManager(pythonresource.DefaultOptions)
	if err := <-errc; err != nil {
		return nil, errors.Wrap(err, "failed to create resource manager")
	}

	builderLoader := &pythonbatch.BuilderLoader{
		Graph:   resourceManager,
		Options: pythonbatch.DefaultOptions,
	}

	return &python.Services{
		Options:         &serviceOptions,
		ResourceManager: resourceManager,
		BuilderLoader:   builderLoader,
	}, nil
}
