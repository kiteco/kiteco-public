package data

// GHCommitFile groups the metadata for a file modified as part of a commit along with
// the content of the file, before the commit is applied.
// This data comes from the github api
type GHCommitFile struct {
	Patch       string
	BaseContent string
}

// GHCommit groups together the metadata for a commit along with CommitFiles
// This data comes from the github api
type GHCommit struct {
	Message string
	Files   []GHCommitFile
}

// GHPullRequest encapsulates the PR data we use for training
// This data comes from the github api
type GHPullRequest struct {
	RepoOwner string
	RepoName  string
	Number    int
	Title     string
	Body      string
	Commits   []GHCommit
}
