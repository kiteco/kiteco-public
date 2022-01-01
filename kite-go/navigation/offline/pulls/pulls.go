package pulls

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const tries = 5

// Options for extracting pull request data
type Options struct {
	Owner    string
	Repo     string
	WriteDir string
	PRState  string
	PerPage  int
	NumPulls int
	Comments bool
}

// ExtractPulls retrieves pull request data and writes it
func ExtractPulls(opts Options) error {
	if opts.WriteDir == "" {
		return errors.New("writedir required")
	}
	if opts.NumPulls == 0 {
		return errors.New("numpulls required")
	}

	pages := opts.NumPulls / opts.PerPage

	log.Printf("Extracting pulls with options:")
	log.Printf("WriteDir: %s\n", opts.WriteDir)
	log.Printf("PRState: %s\n", opts.PRState)
	log.Printf("Pulls: %d\n", opts.NumPulls)
	log.Printf("PerPage: %d\n", opts.PerPage)
	log.Printf("Pages: %d\n", pages)

	ctx := context.Background()
	extractor, err := newPullRequestExtractor(ctx, opts)
	if err != nil {
		return err
	}
	for i := 1; i <= pages; i++ {
		log.Printf("extracting page number %d / %d\n", i, pages)
		err := extractor.extractRepo(ctx, i)
		if err != nil {
			return err
		}
	}
	return nil
}

type pullRequestExtractor struct {
	opts    Options
	client  *github.Client
	exists  map[int]bool
	limiter *limiter
}

type limiter struct {
	last time.Time
}

func newPullRequestExtractor(ctx context.Context, opts Options) (pullRequestExtractor, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_AUTH_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	exists, err := getExisting(opts.WriteDir)
	if err != nil {
		return pullRequestExtractor{}, err
	}
	return pullRequestExtractor{
		opts:    opts,
		client:  client,
		exists:  exists,
		limiter: &limiter{last: time.Now()},
	}, nil
}

func getExisting(writeDir string) (map[int]bool, error) {
	pulls, err := os.Open(writeDir)
	if err != nil {
		return nil, err
	}
	raw, err := pulls.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	exists := make(map[int]bool)
	for _, pull := range raw {
		number, err := strconv.Atoi(pull)
		if err != nil {
			return nil, err
		}
		exists[number] = true
	}
	return exists, nil
}

func (e pullRequestExtractor) extractRepo(ctx context.Context, pageNumber int) error {
	pulls, err := e.listPulls(ctx, pageNumber)
	if err != nil {
		return err
	}
	for _, pull := range pulls {
		err := e.extractPullRequest(ctx, pull)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e pullRequestExtractor) listPulls(ctx context.Context, pageNumber int) ([]*github.PullRequest, error) {
	e.pause()
	pullOpts := &github.PullRequestListOptions{
		State: e.opts.PRState,
		Base:  "master",
		ListOptions: github.ListOptions{
			Page:    pageNumber,
			PerPage: e.opts.PerPage,
		},
	}
	var pulls []*github.PullRequest
	var err error
	for i := 0; i < tries; i++ {
		pulls, _, err = e.client.PullRequests.List(ctx, e.opts.Owner, e.opts.Repo, pullOpts)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Minute)
			continue
		}
		return pulls, nil
	}
	return nil, err
}

func (e pullRequestExtractor) listComments(ctx context.Context, pullNumber int) ([]*github.PullRequestComment, error) {
	e.pause()
	var commentList []*github.PullRequestComment
	var err error
	for i := 0; i < tries; i++ {
		commentList, _, err = e.client.PullRequests.ListComments(ctx, e.opts.Owner, e.opts.Repo, pullNumber, nil)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Minute)
			continue
		}
		return commentList, nil
	}
	return nil, err
}

func (e pullRequestExtractor) listFiles(ctx context.Context, pullNumber int) ([]*github.CommitFile, error) {
	e.pause()
	var filesList []*github.CommitFile
	var err error
	for i := 0; i < tries; i++ {
		filesList, _, err = e.client.PullRequests.ListFiles(ctx, e.opts.Owner, e.opts.Repo, pullNumber, nil)
		if err != nil {
			log.Println(err)
			time.Sleep(time.Minute)
			continue
		}
		return filesList, nil
	}
	return nil, err
}

type commentData struct {
	json []byte
	body string
	diff string
}

type diffData struct {
	json  []byte
	patch string
}

func (e pullRequestExtractor) extractPullRequest(ctx context.Context, pull *github.PullRequest) error {
	if e.exists[*pull.Number] {
		log.Printf("skipping pull request %d (already exists)\n", *pull.Number)
		return nil
	}
	log.Printf("extracting pull request %d\n", *pull.Number)
	meta, err := json.MarshalIndent(pull, "", " ")
	if err != nil {
		return err
	}

	var comments []commentData
	if e.opts.Comments {
		commentList, err := e.listComments(ctx, *pull.Number)
		if err != nil {
			return err
		}
		for _, comment := range commentList {
			jsonComment, err := json.MarshalIndent(comment, "", " ")
			if err != nil {
				return err
			}
			comment := commentData{
				json: jsonComment,
				body: *comment.Body,
				diff: *comment.DiffHunk,
			}
			comments = append(comments, comment)
		}
	}

	filesList, err := e.listFiles(ctx, *pull.Number)
	if err != nil {
		return err
	}
	var diffs []diffData
	for _, file := range filesList {
		jsonDiff, err := json.MarshalIndent(file, "", " ")
		if err != nil {
			return err
		}
		var patch string
		if file.Patch != nil {
			patch = *file.Patch
		}
		diff := diffData{
			json:  jsonDiff,
			patch: patch,
		}
		diffs = append(diffs, diff)
	}

	dir := filepath.Join(e.opts.WriteDir, strconv.Itoa(*pull.Number))
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(dir, "pull.json"), meta, 0600)
	if err != nil {
		return err
	}

	if e.opts.Comments {
		commentsDir := filepath.Join(dir, "comments")
		if len(comments) > 0 {
			err := os.Mkdir(commentsDir, 0700)
			if err != nil {
				return err
			}
		}
		for id, comment := range comments {
			commentDir := filepath.Join(commentsDir, strconv.Itoa(id))
			err := os.Mkdir(commentDir, 0700)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(filepath.Join(commentDir, "comment.json"), comment.json, 0600)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(filepath.Join(commentDir, "body.txt"), []byte(comment.body), 0600)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(filepath.Join(commentDir, "diff.txt"), []byte(comment.diff), 0600)
			if err != nil {
				return err
			}
		}
	}

	filesDir := filepath.Join(dir, "files")
	if len(diffs) > 0 {
		err := os.Mkdir(filesDir, 0700)
		if err != nil {
			return err
		}
	}
	for id, diff := range diffs {
		fileDir := filepath.Join(filesDir, strconv.Itoa(id))
		err := os.Mkdir(fileDir, 0700)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(fileDir, "diff.json"), diff.json, 0600)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(fileDir, "patch.txt"), []byte(diff.patch), 0600)
		if err != nil {
			return err
		}
	}

	e.exists[*pull.Number] = true
	return nil
}

func (e pullRequestExtractor) pause() {
	// API allows 5000 requests per hour -> 1 request per 720 milliseconds
	// https://developer.github.com/v3/#rate-limiting
	time.Sleep(time.Until(e.limiter.last.Add(720 * time.Millisecond)))
	e.limiter.last = time.Now()
}
