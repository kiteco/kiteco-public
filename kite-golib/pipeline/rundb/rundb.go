package rundb

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

const (
	// DefaultRunDB for pipelines to dump data in
	DefaultRunDB = "s3://kite-data/run-db"
	// DefaultTestRunDB for pipelines to dump data in while testing
	DefaultTestRunDB = "s3://kite-data/run-db-test"
)

const (
	s3Region        = "us-west-1"
	runInfoFilename = "run-info.json"
	doneFilename    = "DONE"
)

// RunDB represents a directory in S3 in which run results can be stored.
type RunDB struct {
	bucket string
	dir    string
}

// NewRunDB for the given S3 directory
func NewRunDB(s3Dir string) (RunDB, error) {
	uri, err := awsutil.ValidateURI(s3Dir)
	if err != nil {
		return RunDB{}, fmt.Errorf("could not validate S3 directory '%s': %v", s3Dir, err)
	}

	// it's possible to coerce
	if strings.HasSuffix(s3Dir, "/") {
		return RunDB{}, fmt.Errorf("s3 directory '%s' cannot end with trailing slash", s3Dir)
	}

	bucket := uri.Host
	dir := uri.Path
	if strings.HasPrefix(dir, "/") {
		dir = strings.Replace(dir, "/", "", 1)
	}

	return RunDB{bucket: bucket, dir: dir}, nil
}

// SaveRun saves the given run, either creating a new entry for the run or updating an existing one.
func (r RunDB) SaveRun(info RunInfo) error {
	buf, err := json.Marshal(info)
	if err != nil {
		return err
	}

	path := info.dependentPath(runInfoFilename)
	if info.Status == StatusFinished {
		donePath := info.dependentPath(doneFilename)
		if err := r.putFile(donePath, []byte("Execution complete")); err != nil {
			return err
		}
	}

	if err := r.putFile(path, buf); err != nil {
		return err
	}

	log.Printf("saved run (status = %s) in %s", info.Status, path)

	return nil
}

// ListRuns in descending order of timestamp
func (r RunDB) ListRuns(parseContent bool) ([]RunInfo, error) {
	keys, err := awsutil.S3ListObjects(s3Region, r.bucket, r.dir)
	if err != nil {
		return nil, err
	}

	var runs []RunInfo

	for _, k := range keys {
		if strings.HasSuffix(k, runInfoFilename) {
			// We only want to list the runs that are the immediate children of the RunDB directory
			if parentDir(parentDir(k)) != r.dir {
				continue
			}

			var info RunInfo
			if parseContent {
				uri := fmt.Sprintf("s3://%s/%s", r.bucket, k)
				f, err := awsutil.NewCachedS3Reader(uri)
				if err != nil {
					return nil, fmt.Errorf("error reading %s: %v", uri, err)
				}
				if err := json.NewDecoder(f).Decode(&info); err != nil {
					return nil, fmt.Errorf("error decoding %s: %v", uri, err)
				}
			} else {
				info, err = r.getRunInfoFromPath(parentDir(k), keys)
				if err != nil {
					return nil, err
				}
			}
			runs = append(runs, info)
		}
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})

	return runs, nil
}

// getRunInfoFromPath gets the run info from the S3 path of the run
func (r RunDB) getRunInfoFromPath(s string, keyList []string) (RunInfo, error) {
	parts := strings.Split(s, "/")
	runName := parts[len(parts)-1]
	idx := strings.Index(runName, "_")
	if idx < 1 {
		return RunInfo{}, fmt.Errorf("expected run name (%s) to be of format <run name>_timestamp", runName)
	}
	timestamp := runName[:idx]
	layout := "2006-01-02T15:04:05Z"
	parsedTimestamp, err := time.Parse(layout, timestamp)
	if err != nil {
		return RunInfo{}, err
	}
	pipeline := runName[idx+1:]
	donePath := fmt.Sprintf("%s/%s", s, doneFilename)
	status := StatusUnknown
	if contains(keyList, donePath) {
		status = StatusFinished
	}
	return RunInfo{
		runDB:         r,
		Pipeline:      pipeline,
		Name:          pipeline,
		CreatedAt:     parsedTimestamp,
		Params:        nil,
		GitCommitHash: "",
		Status:        status,
		StatusUpdated: time.Time{},
	}, nil
}

func contains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}

// S3Dir returns the path for which the DB was configured.
func (r RunDB) S3Dir() string {
	return fmt.Sprintf("s3://%s/%s", r.bucket, r.dir)
}

func (r RunDB) putFile(path string, data []byte) error {
	f, err := fileutil.NewBufferedWriter(path)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return nil
}
