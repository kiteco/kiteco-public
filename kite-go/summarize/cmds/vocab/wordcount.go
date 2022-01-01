package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/summarize/data"
	"github.com/kiteco/kiteco/kite-go/summarize/encode"
	"github.com/kiteco/kiteco/kite-golib/cmdline"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/bpe"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/words"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/dependent"
	"github.com/kiteco/kiteco/kite-golib/pipeline/rundb"
	"github.com/kiteco/kiteco/kite-golib/pipeline/source"
	"github.com/kiteco/kiteco/kite-golib/status"
)

var (
	stats       = status.NewSection("vocab-wordcount")
	commitCount = stats.Counter("commitCount")
	fileCount   = stats.Counter("fileCount")
)

func printStats(start time.Time) string {
	var w bytes.Buffer
	fmt.Fprintf(&w, "Stats after %v:\n", time.Since(start))
	fmt.Fprintf(&w, "commitCount: %v\n", commitCount.GetValue())
	fmt.Fprintf(&w, "fileCount: %v\n", fileCount.GetValue())
	return w.String()
}

var wordCountCmd = cmdline.Command{
	Name:     "wordcount",
	Synopsis: "count words in corpus",
	Args: &wordCountArgs{
		Out:       "wordcounts.json",
		CacheRoot: "/data/kite",
		MinCount:  200,
		SplitsDir: "splits",
	},
}

type repoPathSeen struct {
	seen map[uint64]struct{}
	m    sync.Mutex
}

func newRepoPathSeen() *repoPathSeen {
	return &repoPathSeen{seen: make(map[uint64]struct{})}
}

func (s *repoPathSeen) Seen(owner, name, path string) bool {
	ss := strings.Join([]string{owner, name, path}, ":")
	h := spooky.Hash64([]byte(ss))

	s.m.Lock()
	defer s.m.Unlock()
	if _, ok := s.seen[h]; ok {
		return true
	}
	s.seen[h] = struct{}{}
	return false
}

type wordCountArgs struct {
	Out        string
	CacheRoot  string
	MinCount   int
	SplitsDir  string
	MaxCommits int64
}

func (args *wordCountArgs) Handle() error {
	start := time.Now()
	fail(os.MkdirAll(args.SplitsDir, os.ModePerm))

	go func() {
		for range time.Tick(10 * time.Minute) {
			fmt.Println(printStats(start))
		}
	}()

	commits, err := data.NewGitCommitsSource(source.DatasetOpts{
		NumGo:        2,
		NoCache:      true,
		PanicOnError: true,
		Quit:         source.CountQuiter(&commitCount.Value, args.MaxCommits, 30*time.Second),
	}, "commits", data.TrainDataset, data.ValidateDataset)
	fail(err)

	aggregator, err := words.NewAggregator(args.SplitsDir)
	fail(err)

	seen := newRepoPathSeen()
	count := dependent.NewFromFunc("wordcount", func(s pipeline.Sample) {
		commit := s.(data.GitCommit)

		counts := make(words.Counts)
		add := func(ws []string, ext string) {
			for _, w := range ws {
				counts.Hit(w, ext, 1)
			}
		}

		add(encode.Lex(commit.Message), "msg")
		for _, f := range commit.Files {
			if seen.Seen(commit.RepoOwner, commit.RepoName, f.Path) {
				continue
			}
			add(encode.Lex(f.Path), "path")
			for _, chunk := range f.Chunks {
				add(encode.Lex(chunk.Content), chunk.Type.String())
			}
			fileCount.Add(1)
		}

		commitCount.Add(1)
		aggregator.Add(counts)
	})

	pm := make(pipeline.ParentMap)
	pm.Chain(commits, count)

	pipe := pipeline.Pipeline{
		Name:    "vocab-wordcount",
		Parents: pm,
		Sources: []pipeline.Source{commits},
		ResultsFn: func(map[pipeline.Aggregator]pipeline.Sample) []rundb.Result {
			vals := []rundb.Result{
				rundb.Result{
					Name:  "stats",
					Value: printStats(start),
				},
				rundb.Result{
					Name:  "input",
					Value: rundb.RenderS3DirLink(data.RawGHCommitsCrawl, data.RawGHCommitsCrawl),
				},
				rundb.Result{
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
		NumWorkers: runtime.NumCPU() * 2,
	})
	fail(err)

	_, err = engine.Run()
	fail(err)

	fail(aggregator.Flush())

	wordcount, err := aggregator.Merge(args.MinCount)
	fail(err)

	normalized := wordcount.Normalized(args.MinCount)

	var wc []bpe.BuilderWordCount
	for word, count := range normalized {
		wBytes := []byte(word)
		if word != string(wBytes) {
			panic(fmt.Sprintf("bad times, original word %s != %s (string([]byte(word))), bytes %x", word, string(wBytes), wBytes))
		}
		wc = append(wc, bpe.BuilderWordCount{
			Word:  wBytes,
			Count: count,
		})
	}

	sort.Slice(wc, func(i, j int) bool {
		return wc[i].Count > wc[j].Count
	})

	f, err := os.Create(args.Out)
	fail(err)
	defer f.Close()

	err = json.NewEncoder(f).Encode(&wc)
	fail(err)

	return nil
}
