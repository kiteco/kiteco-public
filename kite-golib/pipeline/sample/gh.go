package sample

// GHRepoMetadataPath is a list of the metadata from all gh repos in the latest dump
// TODO: the new version of this will be a directory, but it takes 8 hours to re run so
// we can leave it for now. Once we update this is should point to a directory.
const GHRepoMetadataPath = "s3://kite-local-pipelines/gh-dump-metadata/2019-03-30_03-36-34-AM.json.gz"

// GHRepoMetadata for a crawled repo
type GHRepoMetadata struct {
	// Path to the repo contents on s3
	Path string
	// ID of the repo, internal to kite
	ID    int64
	Owner string
	Repo  string
	// ForkedFrom is the ID of the repo that this was forked from, -1
	// if the repo is unique
	ForkedFrom int64
	Forks      int
}

// SampleTag implements pipeline.Sample
func (GHRepoMetadata) SampleTag() {}

// GHFile represents a file in a repo crawled from github
type GHFile struct {
	Name     string
	Contents []byte
}

// SampleTag implements pipeline.Sample
func (GHFile) SampleTag() {}

// GHRepo represents a repo crawled from github
type GHRepo struct {
	Meta  GHRepoMetadata
	Files []GHFile
}

// SampleTag implements pipeline.Sample
func (GHRepo) SampleTag() {}
