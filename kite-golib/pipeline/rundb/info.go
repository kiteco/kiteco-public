package rundb

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

// Status describes the status of the given run.
type Status string

const (
	// StatusUnknown is the default status
	StatusUnknown Status = ""
	// StatusStarted is set when the pipeline has started running.
	StatusStarted Status = "started"
	// StatusFinished is set when the pipeline has successfully finished.
	StatusFinished Status = "finished"
	// StatusError is set when the pipeline has errored out before it could successfully finish.
	StatusError Status = "error"
)

const (
	maxErrorSamples = 10 // The maximum number of error samples for a feed
)

var (
	rng = rand.New(rand.NewSource(3))
)

// Result describes a named piece of data that is saved inline with the run info for displaying.
type Result struct {
	Name  string
	Value interface{}
	// Aggregator optionally contains the name of the pipeline aggregator used to generate the result
	Aggregator string
}

// RunInfo describes a pipeline run, containing the results as well as some metadata.
type RunInfo struct {
	runDB     RunDB
	artifacts []string
	childRuns []ChildRun

	Pipeline  string
	Name      string
	CreatedAt time.Time

	// Params represents the parameters used to create the pipeline.
	Params map[string]interface{}

	GitCommitHash string
	GitBranch     string

	// FeedStats contains stats for each feed by name.
	FeedStats map[string]FeedStats
	Results   []Result

	Error         string
	Status        Status
	StatusUpdated time.Time
}

// ChildRun represents a run that's a child of the given run
type ChildRun struct {
	RelativePath string
	Info         RunInfo
}

// NewRunInfo creates a new RunInfo, using the current time as the timestamp.
func NewRunInfo(runDB RunDB, pipeline string, name string) RunInfo {
	hash, branch := getGitHashAndBranch()

	return RunInfo{
		runDB:         runDB,
		Pipeline:      pipeline,
		Name:          name,
		CreatedAt:     time.Now().UTC(),
		GitCommitHash: hash,
		GitBranch:     branch,
	}
}

// NewRunInfoFromPath retrieves a run, which is in the format of s3://<bucket name>/path/to/<pipeline>_<timestamp>
func NewRunInfoFromPath(s3Path string) (RunInfo, error) {
	_, err := awsutil.ValidateURI(s3Path)
	if err != nil {
		return RunInfo{}, fmt.Errorf("could not parse path (%s): %v", s3Path, err)
	}
	if strings.HasSuffix(s3Path, "/") {
		return RunInfo{}, fmt.Errorf("cannot have trailing slash in run '%s'", s3Path)
	}

	runDBDir := parentDir(s3Path)

	rdb, err := NewRunDB(runDBDir)
	if err != nil {
		return RunInfo{}, err
	}

	infoPath := fmt.Sprintf("%s/%s", s3Path, runInfoFilename)

	f, err := awsutil.NewCachedS3Reader(infoPath)
	if err != nil {
		return RunInfo{}, fmt.Errorf("error opening %s: %v", infoPath, err)
	}

	var info RunInfo
	if err := json.NewDecoder(f).Decode(&info); err != nil {
		return RunInfo{}, fmt.Errorf("error decoding %s: %v", infoPath, err)
	}
	info.runDB = rdb

	info.artifacts, info.childRuns, err = info.findArtifactsAndChildren()
	if err != nil {
		return RunInfo{}, fmt.Errorf(
			"error getting artifacts and children of run with path '%s': %v", s3Path, err)
	}

	return info, nil
}

// Artifacts represent files that are associated with the run and stored in S3, represented as paths
// relative to the run's directory.
func (r RunInfo) Artifacts() []string {
	return r.artifacts
}

// ChildRuns list all the runs that are children of the given run
func (r *RunInfo) ChildRuns() []ChildRun {
	return r.childRuns
}

// SetStatus updates the status of the RunInfo
func (r *RunInfo) SetStatus(s Status) {
	r.Status = s
	r.StatusUpdated = time.Now().UTC()
}

// S3Path of where the run is located
func (r RunInfo) S3Path() string {
	return fmt.Sprintf("%s/%s", r.runDB.S3Dir(), r.pathRelativeToRunDB())
}

func (r RunInfo) pathRelativeToRunDB() string {
	return fmt.Sprintf("%s_%s", r.CreatedAt.Format(time.RFC3339), r.Pipeline)
}

// full S3 path of a dependent file given the relative name
func (r RunInfo) dependentPath(rel string) string {
	return fmt.Sprintf("%s/%s", r.S3Path(), rel)
}

// Save the given run
func (r RunInfo) Save() error {
	return r.runDB.SaveRun(r)
}

// getGitHashAndBranch attempts to get the git commit hash and branch of the current git repo by making calls to the
// git CLI, or checking environment variables if that fails. Returns blank strings if they cannot be found.
func getGitHashAndBranch() (hash string, branch string) {
	hashCmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := hashCmd.Output()
	if err != nil {
		log.Printf("error getting git hash via CLI: %v", err)
		log.Printf("using $GIT_HASH var")
		hash = os.Getenv("GIT_HASH")
	} else {
		hash = strings.TrimSpace(string(out))
	}

	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err = branchCmd.Output()
	if err != nil {
		log.Printf("error getting git branch via CLI: %v", err)
		log.Printf("using $GIT_BRANCH var")
		branch = os.Getenv("GIT_BRANCH")
	} else {
		branch = strings.TrimSpace(string(out))
	}

	return
}

// findArtifactsAndChildren finds the artifacts and child runs that are associated with the given run.
func (r RunInfo) findArtifactsAndChildren() ([]string, []ChildRun, error) {
	keys, err := awsutil.S3ListObjects(s3Region, r.runDB.bucket,
		fmt.Sprintf("%s/%s/", r.runDB.dir, r.pathRelativeToRunDB()))
	if err != nil {
		return nil, nil, err
	}

	var artifacts []string
	var childRunDirs []string

	for _, k := range keys {
		path := fmt.Sprintf("s3://%s/%s", r.runDB.bucket, k)
		if path == r.dependentPath(doneFilename) || path == r.dependentPath(runInfoFilename) {
			continue
		}
		// find the path of the artifact relative to the path of the run
		relPath := strings.Replace(path, r.S3Path()+"/", "", 1)
		artifacts = append(artifacts, relPath)

		// if we see a run-info.json file, we assume it's part of a child run
		if strings.HasSuffix(path, runInfoFilename) {
			childRunDirs = append(childRunDirs, parentDir(relPath))
		}
	}

	// filter out the artifacts that belong to child runs
	var filteredArtifacts []string
	for _, art := range artifacts {
		var found bool
		for _, cr := range childRunDirs {
			if strings.HasPrefix(art, cr) {
				found = true
				break
			}
		}
		if !found {
			filteredArtifacts = append(filteredArtifacts, art)
		}
	}

	childRuns := make([]ChildRun, 0, len(childRunDirs))
	for _, relDir := range childRunDirs {
		absDir := fmt.Sprintf("%s/%s", r.S3Path(), relDir)

		runDBDir := parentDir(absDir)
		rdb, err := NewRunDB(runDBDir)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid RunDB dir '%s': %v", runDBDir, err)
		}
		ri, err := rdb.getRunInfoFromPath(fmt.Sprintf("%s/%s", rdb.dir, relDir), keys)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting run info from dir %s: %v", absDir, err)
		}

		childRuns = append(childRuns, ChildRun{
			RelativePath: relDir,
			Info:         ri,
		})
	}

	return filteredArtifacts, childRuns, nil
}

func parentDir(path string) string {
	parts := strings.Split(path, "/")
	return strings.Join(parts[:len(parts)-1], "/")
}
