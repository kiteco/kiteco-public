package githubcorpus

import (
	"github.com/google/go-github/github"
	"github.com/kiteco/kiteco/kite-golib/errors"
)

// GetSourceState returns the source state of the file that the commit file modifies,
// GetDestintationState(commit, fn, contents) == Apply(commit.Path, GetSourceState(commit, fn, contents)).
// NOTE: an error is returned if the commit has more than one possible source state (e.g because it was a merge commit)
func GetSourceState(commit *github.RepositoryCommit, fn string, contents Contents) ([]byte, error) {
	if len(commit.Parents) != 1 {
		return nil, errors.New("unable to get source state from commit file, has %d parents and need exactly 1", len(commit.Parents))
	}

	parent := commit.Parents[0]
	return contents.GetFile(parent.GetSHA(), fn)
}

// GetDestinationState returns the dest state of the file that the commit file modifies,
// GetDestintationState(commit, fn, contents) == Apply(commit.Path, GetSourceState(commit, fn, contents)).
func GetDestinationState(commit *github.RepositoryCommit, fn string, contents Contents) ([]byte, error) {
	return contents.GetFile(commit.GetSHA(), fn)
}

// Apply the provided patch to the provided file contents
func Apply(patch string, buf []byte) ([]byte, error) {
	panic("not implemented yet")
}
