package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/kiteco/kiteco/kite-go/summarize/data"
	"github.com/kiteco/kiteco/kite-go/summarize/filter"
	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/kite-golib/githubcorpus"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/pipeline/transform"
	"github.com/kiteco/kiteco/kite-golib/status"
	"github.com/kiteco/kiteco/kite-golib/tarball"
	"github.com/kiteco/kiteco/kite-golib/text"
)

var extractCmd = cmdline.Command{
	Name:     "extract",
	Synopsis: "extract gh commits from gh dump",
	Args: &extractArgs{
		RunDB:               rundb.DefaultRunDB,
		Verbose:             true,
		MaxChangesPerCommit: 10,
		Role:                pipeline.Standalone,
	},
}

var (
	extractStats       = status.NewSection("gh-commits-extract")
	mergeCommitRatio   = extractStats.Ratio("mergeCommitRatio")
	skippedCommitCount = extractStats.Counter("skippedCommitCount")
	repoCount          = extractStats.Counter("repoCount")
	sampleCount        = extractStats.Counter("sampleCount")
	skippedRepoCount   = extractStats.Counter("skippedRepoCount")
)

func printExtractStats(start time.Time) string {
	var w bytes.Buffer
	fmt.Fprintf(&w, "Stats after %v:\n", time.Since(start))
	fmt.Fprintf(&w, "skippedRepoCount: %v\n", skippedRepoCount.GetValue())
	fmt.Fprintf(&w, "mergeCommitRatio: %v\n", mergeCommitRatio.Value())
	fmt.Fprintf(&w, "skippedCommitCount: %v\n", skippedCommitCount.GetValue())
	fmt.Fprintf(&w, "repoCount: %v\n", repoCount.GetValue())
	fmt.Fprintf(&w, "sampleCount: %v\n", sampleCount.GetValue())
	return w.String()
}

type extractArgs struct {
	Role                pipeline.Role
	Port                int
	Endpoints           []string
	Crawl               string
	Out                 string
	RunDB               string
	Verbose             bool
	MaxSamples          int64
	MaxChangesPerCommit int
	TmpDir              string
}

func (args *extractArgs) Handle() error {
	start := time.Now()
	fail(os.MkdirAll(args.TmpDir, os.ModePerm))

	var logger io.Writer
	if args.Verbose {
		logger = os.Stderr
	}

	go func() {
		for range time.Tick(10 * time.Minute) {
			fmt.Println(printExtractStats(start))
		}
	}()

	files, err := aggregator.ListDir(args.Crawl)
	fail(err)

	crawl := source.NewGHReposCrawl(source.DatasetOpts{
		Logger:       logger,
		NumGo:        1,
		NoCache:      true,
		PanicOnError: true,
		Quit:         source.CountQuiter(&sampleCount.Value, args.MaxSamples, 30*time.Second),
	}, maxRepoSizeCompressed, "gh-crawl", files...)

	extract := transform.NewMapChan("extract", func(s pipeline.Sample) chan pipeline.Sample {
		res := make(chan pipeline.Sample)
		go func() {
			defer close(res)
			extractCommits(args.MaxSamples, args.MaxChangesPerCommit, s, res)
		}()
		return res
	})

	write := newGitCommitsWriter(args.Out, "writer", aggregator.WriterOpts{
		Logger: logger,
		NumGo:  2,
		TmpDir: args.TmpDir,
	})

	pm := make(pipeline.ParentMap)
	pm.Chain(
		crawl,
		extract,
		write,
	)

	pipe := pipeline.Pipeline{
		Name:    "gh-commits-extract",
		Parents: pm,
		Sources: []pipeline.Source{crawl},
		ResultsFn: func(map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
			vals := []rundb.Result{
				{
					Name:  "stats",
					Value: printExtractStats(start),
				},
				{
					Name:  "crawl",
					Value: rundb.RenderS3DirLink(args.Crawl, args.Crawl),
				},
				{
					Name:  "results",
					Value: rundb.RenderS3DirLink(args.Out, args.Out),
				},
			}
			for _, val := range vals {
				fmt.Printf("%s: %v\n", val.Name, val.Value)
			}
			return vals
		},
	}

	engine, err := pipeline.NewEngine(pipe, pipeline.EngineOptions{
		NumWorkers:     runtime.NumCPU(),
		Role:           args.Role,
		RunName:        args.Crawl,
		RunDBPath:      args.RunDB,
		Port:           args.Port,
		ShardEndpoints: args.Endpoints,
	})
	fail(err)

	_, err = engine.Run()
	fail(err)
	return nil
}

func extractCommits(maxSamples int64, maxChangesPerCommit int, s pipeline.Sample, res chan pipeline.Sample) {
	done := func() bool {
		return maxSamples > 0 && sampleCount.GetValue() > maxSamples
	}
	if done() {
		return
	}

	repo := openRepo(s)
	defer os.RemoveAll(repo.TmpDir)
	if repo.Repo == nil {
		skippedRepoCount.Add(1)
		return
	}

	head, err := repo.Repo.Head()
	fail(err)

	commits, err := repo.Repo.Log(&git.LogOptions{
		From:  head.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	fail(err)

	fail(commits.ForEach(func(c *object.Commit) (err error) {
		if done() {
			return storer.ErrStop
		}
		defer func() {
			if err != nil {
				skippedCommitCount.Add(1)
				err = nil
			}
		}()

		if c.NumParents() != 1 {
			// skip merge commits
			mergeCommitRatio.Hit()
			return nil
		}
		mergeCommitRatio.Miss()

		parent, err := c.Parent(0)
		fail(err)

		patch := computePatch(parent, c, maxChangesPerCommit)
		if patch == nil {
			return errors.New("unable to compute patch")
		}

		msg, err := text.StandardizeEncoding(c.Message)
		if err != nil {
			return err
		}

		commitData := data.GitCommit{
			RepoOwner: repo.Owner,
			RepoName:  repo.Name,
			Message:   msg,
		}
		for _, fp := range patch.FilePatches() {
			if fp.IsBinary() {
				continue
			}

			frm, _ := fp.Files()
			frmPath, err := text.StandardizeEncoding(frm.Path())
			if err != nil {
				return err
			}

			commitFileData := data.GitCommitFile{Path: frmPath}
			for _, chunk := range fp.Chunks() {
				content, err := text.StandardizeEncoding(chunk.Content())
				if err != nil {
					return err
				}
				commitFileData.Chunks = append(commitFileData.Chunks, data.GitDiffChunk{
					Content: content,
					Type:    data.DiffOperation(chunk.Type()),
				})
			}
			commitData.Files = append(commitData.Files, commitFileData)
		}
		if len(commitData.Files) == 0 {
			return errors.New("no commit files")
		}

		res <- commitData
		sampleCount.Add(1)
		return nil
	}))

	repoCount.Add(1)
}

func computePatch(from, to *object.Commit, maxChangesPerCommit int) *object.Patch {
	// TODO: because of https://github.com/sergi/go-diff/issues/89
	// we have to first compute the set of files that changed,
	// then check to make sure none of the files are too large,
	// then compute the patch.
	ft, err := from.Tree()
	fail(err)
	tt, err := to.Tree()
	fail(err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	changes, err := object.DiffTreeWithOptions(ctx, ft, tt, &object.DiffTreeOptions{
		DetectRenames:    true,
		RenameScore:      60,
		RenameLimit:      uint(maxChangesPerCommit), // detecting renames can use a fair amount of memory so we set this to be a very low value
		OnlyExactRenames: false,
	})
	if err == object.ErrCanceled {
		return nil
	}
	fail(err)

	if maxChangesPerCommit > 0 && len(changes) > maxChangesPerCommit {
		return nil
	}
	for _, c := range changes {
		if c.From.Name != c.To.Name {
			// skip commits that contain renames, or additions or deletions (since in these cases the paths will be empty)
			return nil
		}

		fsz, err := ft.Size(c.From.Name)
		if err != nil || (maxFileSize > 0 && fsz > maxFileSize) {
			// non nil err means the file was not found in the repo, this can happen for forks that are embedded in a repo
			// e.g kiteco/atom-plugin/atom-snippets-fork
			return nil
		}

		tsz, err := tt.Size(c.To.Name)
		if err != nil || (maxFileSize > 0 && tsz > maxFileSize) {
			// non nil err means the file was not found in the repo, this can happen for forks that are embedded in a repo
			// e.g kiteco/atom-plugin/atom-snippets-fork
			return nil
		}

		f, err := ft.File(c.From.Name)
		if err != nil {
			// for some reason we can still get object.ErrFileNotFound here
			return nil
		}

		contents, err := f.Contents()
		fail(err)

		if filter.File(c.From.Name, []byte(contents)) {
			// skip commits that contain filtered files
			return nil
		}
	}

	patch, err := changes.Patch()
	fail(err)
	return patch
}

type repo struct {
	Owner  string
	Name   string
	Repo   *git.Repository
	TmpDir string
}

func openRepo(s pipeline.Sample) repo {
	ks := s.(pipeline.Keyed)

	// TODO(juan): ideally we could avoid having to write the repo to temporary disk storage,
	// but I was too lazy to figure out how to do this.
	owner, name, _, err := githubcorpus.ParseRepoCorpusFilename(path.Base(ks.Key))
	fail(err)

	tmpDir, err := ioutil.TempDir("", "")
	fail(err)

	fail(tarball.UnpackGzipBytes(tmpDir, ks.Sample.(sample.ByteSlice)))

	// repo is one dir in a directory with the repo name
	r, err := git.PlainOpen(filepath.Join(tmpDir, name))
	if err == git.ErrRepositoryNotExists {
		// occasionally we get weird repos that do no exist
		fmt.Printf("repo %s/%s does not exist, skipping (%s)\n", owner, name, ks.Key)
		return repo{}
	}
	fail(err)
	return repo{
		Owner:  owner,
		Name:   name,
		Repo:   r,
		TmpDir: tmpDir,
	}
}
