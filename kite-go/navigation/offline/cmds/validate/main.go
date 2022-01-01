package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

	"github.com/alexflint/go-arg"
	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/kiteco/kiteco/kite-go/navigation/ignore"
	"github.com/kiteco/kiteco/kite-go/navigation/localpath"
	"github.com/kiteco/kiteco/kite-go/navigation/offline/validation"
)

func main() {
	args := struct {
		ReposPath            string
		ReadDir              string
		StatsPath            string
		RecordsPath          string
		MaxFileRecs          int
		MaxBlockRecs         int
		UseCommits           bool
		ComputedCommitsLimit int
		StoragePath          string
		KeepUnderscores      bool
		SkipLines            bool
	}{
		MaxFileRecs:          5,
		MaxBlockRecs:         3,
		ComputedCommitsLimit: git.DefaultComputedCommitsLimit,
		StoragePath: filepath.Join(
			os.Getenv("GOPATH"),
			"src", "github.com", "kiteco", "kiteco",
			"kite-go", "navigation", "offline", "git-cache.json",
		),
	}
	arg.MustParse(&args)

	readDir, err := localpath.NewAbsolute(args.ReadDir)
	if err != nil {
		log.Fatal(err)
	}

	repos, err := validation.ReadRepos(args.ReposPath)
	if err != nil {
		log.Fatal(err)
	}

	storageOpts := git.StorageOptions{
		UseDisk: true,
		Path:    args.StoragePath,
	}
	s, err := git.NewStorage(storageOpts)
	if err != nil {
		log.Fatal(err)
	}

	var results []result
	var m sync.Mutex
	var wg sync.WaitGroup

	for _, repo := range repos {
		wg.Add(1)
		go func(repo validation.Repo) {
			defer wg.Done()
			log.Printf("starting on %s/%s\n", repo.Owner, repo.Name)

			repoDir := readDir.Join(localpath.Relative(repo.Owner), localpath.Relative(repo.Name))
			opts := validation.Options{
				UseCommits:           args.UseCommits,
				PullsPath:            repoDir.Join("open"),
				Root:                 repoDir.Join("root"),
				MaxFileRecs:          args.MaxFileRecs,
				MaxBlockRecs:         args.MaxBlockRecs,
				IgnoreFilenames:      []localpath.Relative{ignore.GitIgnoreFilename},
				ComputedCommitsLimit: args.ComputedCommitsLimit,
				KeepUnderscores:      args.KeepUnderscores,
				SkipLines:            args.SkipLines,
				Storage:              s,
			}
			fileStats, lineStats, records := validation.Validate(opts)

			m.Lock()
			defer m.Unlock()
			results = append(results, result{
				file:    fileStats,
				line:    lineStats,
				records: records,
				label:   fmt.Sprintf("%s/%s", repo.Owner, repo.Name),
			})

			log.Printf("finished with %s/%s\n", repo.Owner, repo.Name)
		}(repo)
	}

	wg.Wait()

	sort.Slice(results, func(i, j int) bool { return results[i].label < results[j].label })

	var files, lines []validation.Stats
	for _, r := range results {
		files = append(files, r.file)
		lines = append(lines, r.line)
	}
	results = append(results, result{
		file:  validation.Mean(files),
		line:  validation.Mean(lines),
		label: "mean",
	})

	err = writeStats(args.StatsPath, results)
	if err != nil {
		log.Fatal(err)
	}
	err = writeRecords(args.RecordsPath, results)
	if err != nil {
		log.Fatal(err)
	}
}

type result struct {
	file    validation.Stats
	line    validation.Stats
	records []validation.Record
	label   string
}

func writeStats(path string, results []result) error {
	if path == "" {
		log.Println("path is empty, skipping writeStats")
		return nil
	}
	log.Println("writing stats")

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	header := []string{
		"label",
		"file_f1",
		"file_precision",
		"file_recall",
		"line_f1",
		"line_precision",
		"line_recall",
	}
	err = writer.Write(header)
	if err != nil {
		return err
	}

	for _, r := range results {
		row := []string{r.label}
		row = append(row, r.file.Strings()...)
		row = append(row, r.line.Strings()...)
		err := writer.Write(row)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeRecords(path string, results []result) error {
	if path == "" {
		log.Println("path is empty, skipping writeRecords")
		return nil
	}
	log.Println("writing records")

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	header := []string{
		"label",
		"base",
		"recommended",
		"score",
		"is_relevant",
	}
	err = writer.Write(header)
	if err != nil {
		return err
	}

	for _, r := range results {
		for _, record := range r.records {
			row := []string{
				r.label,
				record.Base,
				record.Recommended,
				strconv.FormatFloat(record.Score, 'f', 6, 64),
				strconv.FormatBool(record.IsRelevant),
			}
			err := writer.Write(row)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
