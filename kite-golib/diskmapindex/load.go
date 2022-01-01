package diskmapindex

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/diskmap"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

func readCloser(path, cache string) (io.ReadCloser, error) {
	if awsutil.IsS3URI(path) {
		return awsutil.NewCachedS3ReaderWithOptions(awsutil.CachedReaderOptions{
			CacheRoot: cache,
			Logger:    os.Stdout,
		}, path)
	}
	return os.Open(path)
}

func downloadFile(path, cache string) (string, error) {
	if !awsutil.IsS3URI(path) {
		return path, nil
	}

	url, err := awsutil.ValidateURI(path)
	if err != nil {
		return "", fmt.Errorf("error validating uri %s: %v", path, err)
	}

	localPath := awsutil.CachePathAt(cache, url)

	err = checkLocal(path, localPath)
	if err == nil {
		return localPath, nil
	}
	log.Printf("error verifying local %s, will attempt download: %v", localPath, err)

	r, err := awsutil.NewCachedS3ReaderWithOptions(awsutil.CachedReaderOptions{
		CacheRoot: cache,
		Logger:    os.Stdout,
	}, path)

	if err != nil {
		return "", fmt.Errorf("error opening %s: %v", path, err)
	}
	defer r.Close()

	// read everything so that file is committed to the cache
	if _, err := io.Copy(ioutil.Discard, r); err != nil {
		return "", fmt.Errorf("error reading %s: %v", path, err)
	}
	return localPath, nil
}

// checkLocal returns nil if the correct file has already been downloaded locally.
func checkLocal(path, localPath string) error {
	s3Checksum, err := awsutil.ChecksumS3(path)
	if err != nil {
		return err
	}

	localChecksum, err := awsutil.SavedChecksum(localPath)
	if err != nil {
		return err
	}

	if !bytes.Equal(s3Checksum, localChecksum) {
		return fmt.Errorf("checksum mismatch: S3 = %s, local = %s", s3Checksum, localChecksum)
	}
	return nil
}

func decodeDiskMaps(names []string, cache string, decompress bool) ([]*diskmap.Map, error) {
	n := 1
	if os.Getenv("INSTANCE_ID") != "" {
		// Hack: crank up parallelism if we're running on a cluster, and use this env var
		// as a proxy for indicating we are on a cluster.
		n = runtime.NumCPU()
	}
	wp := workerpool.New(n)

	diskMaps := make([]*diskmap.Map, len(names))
	for idx, blockName := range names {
		localIdx := idx
		localBlockName := blockName
		wp.Add([]workerpool.Job{func() error {
			localPath, err := downloadFile(localBlockName, cache)
			if err != nil {
				return fmt.Errorf("error getting file %s: %v", localBlockName, err)
			}

			localNameDecomp := localPath
			if decompress {
				// decompress block
				localNameDecomp = fmt.Sprintf("%s-decompressed", localPath)
				if err := decompressFile(localPath, localNameDecomp); err != nil {
					return err
				}
			}

			dm, err := diskmap.NewMap(localNameDecomp)
			if err != nil {
				return fmt.Errorf("error creating diskmap at %s (%s remote): %v", localNameDecomp, localBlockName, err)
			}

			diskMaps[localIdx] = dm
			return nil
		}})
	}

	err := wp.Wait()
	if err != nil {
		return nil, err
	}

	wp.Stop()

	return diskMaps, nil
}

func decompressFile(path, decompPath string) error {
	// Skip if the file has already been decompressed
	donePath := fmt.Sprintf("%s-done", decompPath)
	_, err := os.Stat(donePath)
	if err == nil {
		return nil
	}

	f, err := os.Create(decompPath)
	if err != nil {
		return fmt.Errorf("error creating local decomp file %s: %v", decompPath, err)
	}
	defer f.Close()

	rdr, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening local block %s: %v", path, err)
	}
	defer rdr.Close()

	drdr, err := gzip.NewReader(rdr)
	if err != nil {
		return fmt.Errorf("error getting gzip reader for %s: %v", decompPath, err)
	}
	defer drdr.Close()

	if _, err := io.Copy(f, drdr); err != nil {
		return fmt.Errorf("error decompressing %s -> %s: %v", path, decompPath, err)
	}

	doneFile, err := os.Create(donePath)
	if err != nil {
		return fmt.Errorf("error creating done file %s: %v", donePath, err)
	}
	doneFile.Close()

	return nil
}
