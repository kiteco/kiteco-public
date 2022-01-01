package githubcorpus

import (
	"context"

	"github.com/google/go-github/github"
)

// Contents ...
type Contents interface {
	// GetFile takes filename, sha and returns contents of filename at the given sha
	GetFile(string, string) ([]byte, error)
	// GetCommit takes the sha of a commit and returns the full contents of the commit object
	GetCommit(string) (*github.RepositoryCommit, error)
}

type contents struct {
	client *github.Client
	owner  string
	name   string
	limit  func()
}

func (c contents) GetFile(sha, fn string) ([]byte, error) {
	c.limit()
	content, _, _, err := c.client.Repositories.GetContents(
		context.Background(), c.owner, c.name, fn,
		&github.RepositoryContentGetOptions{
			Ref: sha,
		},
	)
	if err != nil {
		return nil, err
	}

	cstr, err := content.GetContent()
	if err != nil {
		GetContentSuccessRate.Miss()
		return nil, err
	}
	GetContentSuccessRate.Hit()

	return []byte(cstr), nil
}

func (c contents) GetCommit(sha string) (*github.RepositoryCommit, error) {
	c.limit()
	content, _, err := c.client.Repositories.GetCommit(
		context.Background(), c.owner, c.name, sha,
	)
	if err != nil {
		GetCommitSuccessRate.Miss()
		return nil, err
	}
	GetCommitSuccessRate.Hit()

	return content, nil
}
