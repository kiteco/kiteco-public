package githubcorpus

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/google/go-github/github"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"golang.org/x/oauth2"
)

// APICorpus uses the live github API to crawl pull requests
type APICorpus struct {
	client *github.Client
	opts   APIScanOptions

	lastRequest time.Time
	delay       time.Duration
	requests    int
}

// APIScanOptions ...
type APIScanOptions struct {
	State string
	Base  string

	MaxPages int
	PerPage  int

	IncludeCommits     bool
	IncludeCommitFiles bool
}

// NewAPIPullRequestCorpus ...
func NewAPIPullRequestCorpus(opts APIScanOptions) (*APICorpus, error) {
	if !opts.IncludeCommitFiles && !opts.IncludeCommits {
		return nil, errors.New("atleast one of IncludeCommitFiles or IncludeCommits must be true")
	}
	if opts.State == "" {
		return nil, errors.New("pr state must be set")
	}
	if opts.Base == "" {
		opts.Base = "master"
	}

	client := github.NewClient(
		oauth2.NewClient(context.Background(),
			oauth2.StaticTokenSource(
				&oauth2.Token{
					AccessToken: os.Getenv("GITHUB_AUTH_TOKEN"),
				},
			),
		),
	)
	return &APICorpus{
		client:      client,
		opts:        opts,
		lastRequest: time.Now(),
	}, nil
}

// Scan implements Corpus
func (r *APICorpus) Scan(f PullRequestFunc) error {
	return errors.Wrapf(nil, "scan not implemented")
}

// ScanRepo implements Corpus
func (r *APICorpus) ScanRepo(owner, name string, f PullRequestFunc) error {
	err := r.paginator(func(listOpts *github.ListOptions) (*github.Response, bool, error) {
		if r.opts.MaxPages > 0 && listOpts.Page > r.opts.MaxPages {
			return nil, false, nil
		}

		pullOpts := &github.PullRequestListOptions{
			State:       r.opts.State,
			Base:        r.opts.Base,
			Direction:   "desc",
			ListOptions: *listOpts,
		}

		prs, resp, err := r.client.PullRequests.List(context.Background(), owner, name, pullOpts)
		if err != nil {
			return nil, false, errors.Wrapf(err, "error getting pull requests for %s/%s, page=%d",
				owner, name, listOpts.Page)
		}

		for _, pr := range prs {

			var commitFiles []*github.CommitFile
			if r.opts.IncludeCommitFiles {
				commitFiles, err = r.getCommitFiles(owner, name, pr)
				if err != nil {
					return nil, false, errors.Wrapf(err, "error getting commit files for %s/%s, page=%d",
						owner, name, listOpts.Page)
				}
			}

			var commits []*github.RepositoryCommit
			if r.opts.IncludeCommits {
				commits, err = r.getCommits(owner, name, pr)
				if err != nil {
					return nil, false, errors.Wrapf(err, "error getting commits for pr %s/%s#%d",
						owner, name, pr.GetNumber())
				}
			}

			bundle := PullRequestBundle{
				Owner:       owner,
				Repo:        name,
				PullRequest: pr,
				CommitFiles: commitFiles,
				Commits:     commits,
			}

			next := f(bundle, contents{
				owner:  owner,
				name:   name,
				client: r.client,
				limit:  r.limit,
			})
			if !next {
				return resp, false, ErrStoppedEarly
			}
		}

		return resp, len(prs) > 0, nil
	})

	return err
}

// --

func (r *APICorpus) getCommitFiles(owner, name string, pr *github.PullRequest) ([]*github.CommitFile, error) {
	var commitFiles []*github.CommitFile
	err := r.paginator(func(listOpts *github.ListOptions) (*github.Response, bool, error) {
		if listOpts.Page > r.opts.MaxPages {
			return nil, false, nil
		}

		files, resp, err := r.client.PullRequests.ListFiles(context.Background(), owner, name, pr.GetNumber(), listOpts)
		if err != nil {
			return nil, false, errors.Wrapf(err, "error getting files for pr %s/%s#%d",
				owner, name, pr.GetNumber())
		}

		commitFiles = append(commitFiles, files...)
		return resp, len(files) > 0, nil
	})

	if err != nil {
		return nil, err
	}
	return commitFiles, nil
}

func (r *APICorpus) getCommits(owner, name string, pr *github.PullRequest) ([]*github.RepositoryCommit, error) {
	var commits []*github.RepositoryCommit
	err := r.paginator(func(listOpts *github.ListOptions) (*github.Response, bool, error) {
		if listOpts.Page > r.opts.MaxPages {
			return nil, false, nil
		}

		cs, resp, err := r.client.PullRequests.ListCommits(context.Background(), owner, name, pr.GetNumber(), listOpts)
		if err != nil {
			return nil, false, errors.Wrapf(err, "error getting commits for pr %s/%s#%d",
				owner, name, pr.GetNumber())
		}

		commits = append(commits, cs...)
		return resp, len(cs) > 0, nil
	})

	if err != nil {
		return nil, err
	}

	return commits, nil
}

type paginationFunc func(*github.ListOptions) (*github.Response, bool, error)

func (r *APICorpus) paginator(f paginationFunc) error {
	listOpts := &github.ListOptions{
		Page:    1,
		PerPage: r.opts.PerPage,
	}

	lastPage := 10 // some initial value, will be updated on first response
	for listOpts.Page < lastPage {
		r.limit() // its assumed f will hit the API once

		resp, fetchNext, err := f(listOpts)
		if err != nil {
			return err
		}

		if !fetchNext {
			break
		}

		lastPage = resp.LastPage
		listOpts.Page = resp.NextPage
	}
	return nil
}

var recomputeDelay = 100

func (r *APICorpus) limit() {
	if r.requests%recomputeDelay == 0 {
		r.delay = computeDelay(r.client)
	}

	waited := time.Since(r.lastRequest)
	if waited < r.delay {
		time.Sleep(r.delay)
	}

	r.lastRequest = time.Now()
	r.requests++
}

// --

func computeDelay(client *github.Client) time.Duration {
	rateLimit, _, err := client.RateLimits(context.Background())
	if err != nil {
		return time.Second
	}

	perHour := rateLimit.GetCore().Remaining
	timeRemaining := rateLimit.GetCore().Reset.Sub(time.Now())
	if perHour == 0 {
		return timeRemaining
	}
	delay := time.Duration(int64(timeRemaining) / int64(perHour))
	log.Printf("==== %d requests remaining, over %s, delay set to %s", perHour, timeRemaining, delay)
	return delay
}
