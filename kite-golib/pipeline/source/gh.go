package source

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync/atomic"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/text"

	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
)

// NewGHReposCrawl returns pipeline.Keyed samples where the value is a byte slice which hold the full contents of a github
// repo directory (tarred and gziped) and the key is the name of the associated s3 file
func NewGHReposCrawl(opts DatasetOpts, maxRepoSizeCompressed int, name string, files ...string) pipeline.Source {
	return NewDataset(opts, name, ReadProcessFn(maxRepoSizeCompressed), files...)
}

// NewGHMetadata returns a source where the samples are the gh metadata
func NewGHMetadata(opts DatasetOpts, name string, data string) (pipeline.Source, error) {
	var files []string
	if strings.HasSuffix(data, ".json.gz") {
		// TODO: the new version of this will be a directory, this is a hack for backwards compatibility
		// but it takes 8 hours to re run so we can leave it for now. Once we update this is should point to a directory.
		files = []string{data}
	} else {
		var err error
		files, err = aggregator.ListDir(data)
		if err != nil {
			return nil, err
		}
	}

	return NewDataset(opts, name, JSONProcessFn(sample.GHRepoMetadata{}), files...), nil
}

const ghRepoKiteMetadataFile = "kite-repo-entry.json"

// RawGHCrawlOpts ...
type RawGHCrawlOpts struct {
	DatasetOpts
	// Skip takes a file name and file size (in bytes) and returns true if
	// the file should be skipped (not added to the resulting repo)
	Skip               func(string, int) bool
	MaxRepos           int
	MetaDataOnly       bool
	UTF8EncodeNames    bool
	UTF8EncodeContents bool
}

// DefaultRawGHOpts ...
var DefaultRawGHOpts = RawGHCrawlOpts{
	DatasetOpts: DefaultDatasetOpts,
}

// NewRawGHCrawl with the specified options using data from
// the provided directory.
func NewRawGHCrawl(opts RawGHCrawlOpts, name string, repos []string) *Dataset {
	var count int64
	f := func(name string, r io.Reader, recs chan<- RecordStopError) {
		defer close(recs)

		gz, err := gzip.NewReader(r)
		if err != nil {
			recs <- RecordStopError{
				Err: fmt.Errorf("error opening gzip reader: %v", err),
			}
			return
		}

		var repo sample.GHRepo
		var ignoreErr bool
		err = fileutil.ProcessTar(gz, func(path string, r io.Reader) error {
			contents, err := ioutil.ReadAll(r)
			if err != nil {
				return fmt.Errorf("error reading contents of %s: %v", path, err)
			}

			if strings.HasSuffix(path, ghRepoKiteMetadataFile) {
				if err := json.Unmarshal(contents, &repo.Meta); err != nil {
					return errors.Errorf("error decoding metadata file %s: %v", path, err)
				}
				if opts.MetaDataOnly {
					// return a non nil error so we exit out early
					// instead of iterating through the rest of the tar
					ignoreErr = true
					return errors.Errorf("found metadata")
				}
				return nil
			}

			if opts.UTF8EncodeNames {
				sPath, err := text.StandardizeEncoding(path)
				if err != nil {
					logf(opts.Logger, "unable to standardize filename '%s': %v", path, err)
					return nil
				}
				path = sPath
			}

			if opts.Skip != nil && opts.Skip(path, len(contents)) {
				return nil
			}

			if opts.UTF8EncodeContents {
				src, err := text.StandardizeEncoding(string(contents))
				if err != nil {
					logf(opts.Logger, "unable to standardize contents of '%s': %v", path, err)
					return nil
				}
				contents = []byte(src)
			}

			repo.Files = append(repo.Files, sample.GHFile{
				Name:     path,
				Contents: contents,
			})

			return nil
		})

		if err != nil && !ignoreErr {
			recs <- RecordStopError{
				Err: err,
			}
			return
		}

		if repo.Meta == (sample.GHRepoMetadata{}) {
			recs <- RecordStopError{
				Err: errors.Errorf("unable to find metadata for repo %s", name),
			}
			return
		}

		repo.Meta.Path = name

		recs <- RecordStopError{
			Record: pipeline.Record{
				Key:   fmt.Sprintf("%d", repo.Meta.ID),
				Value: repo,
			},
		}

		atomic.AddInt64(&count, 1)
		if opts.MaxRepos > 0 && atomic.LoadInt64(&count) >= int64(opts.MaxRepos) {
			recs <- RecordStopError{
				Stop: true,
			}
		}
	}

	return NewDataset(opts.DatasetOpts, name, f, repos...)
}
