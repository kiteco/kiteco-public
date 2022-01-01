package source

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/pipeline"
)

// Filelist represents a slice of paths
type Filelist []string

// LocalFiles is a source which process a list of files. Each file will be a Sample for the pipeline
type LocalFiles struct {
	name    string
	records chan pipeline.Record
	logger  io.Writer
}

// NewFileExtensionPredicate returns a predicate to filter file based on their extension
func NewFileExtensionPredicate(extension string) func(string) bool {
	return func(path string) bool {
		return strings.HasSuffix(path, extension)
	}
}

// GetFilelist recursively scans `root` folder
// filter will be applied to every filename and the file will be kept when filter return true
func GetFilelist(root string, filter func(string) bool, excludeDirectory bool) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if excludeDirectory && info.IsDir() {
			return nil
		}
		if filter(path) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// NewLocalFiles reads from each provided file from the local filesystem
// DEPRECATED: just call NewDataset directly
func NewLocalFiles(name string, numGo int, filelist Filelist, logger io.Writer) *Dataset {
	return NewDataset(DatasetOpts{
		Logger: logger,
		NumGo:  numGo,
	}, name, ReadProcessFn(0), filelist...)
}
