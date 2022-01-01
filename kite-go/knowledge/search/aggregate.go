package search

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type data struct {
	path       string
	lineNumber int
	content    string
}

func process(raw string) (data, bool, error) {
	parts := strings.Split(raw, ":")
	if len(parts) < 3 {
		return data{}, false, nil
	}

	if runtime.GOOS == "windows" {
		// Account for the first semicolon in C:\
		parts = []string{
			fmt.Sprintf("%s:%s", parts[0], parts[1]),
			parts[2],
			parts[3],
		}
	}

	lineNumber, err := strconv.Atoi(parts[1])
	if err != nil {
		return data{}, true, err
	}
	return data{
		path:       parts[0],
		lineNumber: lineNumber,
		content:    strings.Join(parts[2:], ":"),
	}, true, nil
}

type aggregator struct {
	pullsDir string
	files    map[string][]data
	pulls    map[string][]data
}

func newAggregator(pullsDir string) aggregator {
	return aggregator{
		pullsDir: pullsDir,
		files:    make(map[string][]data),
		pulls:    make(map[string][]data),
	}
}

func (a *aggregator) add(d data) error {
	if strings.HasPrefix(d.path, a.pullsDir) {
		dir, err := pullDir(d.path, a.pullsDir)
		if err != nil {
			return err
		}
		a.pulls[dir] = append(a.pulls[dir], d)
		return nil
	}
	a.files[d.path] = append(a.files[d.path], d)
	return nil
}

func pullDir(path, pullsDir string) (string, error) {
	if !strings.HasPrefix(path, pullsDir) {
		return "", errors.New("bad pull path")
	}
	pullDir := path
	parent := filepath.Dir(pullDir)
	for parent != pullsDir {
		pullDir, parent = parent, filepath.Dir(parent)
	}
	return pullDir, nil
}

func (a aggregator) aggregate() (Results, error) {
	files, err := a.aggregateFiles()
	if err != nil {
		return Results{}, err
	}
	pulls, err := a.aggregatePulls()
	if err != nil {
		return Results{}, err
	}
	return Results{
		Files: files,
		Pulls: pulls,
	}, nil
}

func (a aggregator) aggregateFiles() ([]File, error) {
	var files []File
	for path, dataset := range a.files {
		url, err := makeFileURL(path)
		if err != nil {
			return nil, err
		}
		file := File{
			URL:   url,
			Lines: makeLines(dataset),
		}
		files = append(files, file)
	}
	return files, nil
}

func makeFileURL(path string) (string, error) {
	var parts []string
	var suffix string
	if runtime.GOOS != "windows" {
		parts = strings.Split(path, "github.com/kiteco/kiteco/")
		if len(parts) != 2 {
			return "", errors.New("bad file path")
		}
		suffix = parts[1]
	} else {
		parts = strings.Split(path, "github.com\\kiteco\\kiteco\\")
		if len(parts) != 2 {
			return "", errors.New("bad file path")
		}
		suffix = filepath.ToSlash(parts[1])
	}
	return fmt.Sprintf("https://github.com/kiteco/kiteco/blob/master/%s", suffix), nil
}

func makeLines(dataset []data) []Line {
	var lines []Line
	for _, d := range dataset {
		line := Line{
			Number:  d.lineNumber,
			Content: d.content,
		}
		lines = append(lines, line)
	}
	return lines
}

func (a aggregator) aggregatePulls() ([]Pull, error) {
	var pulls []Pull
	for dir, dataset := range a.pulls {
		meta, err := loadPullMeta(dir)
		if err != nil {
			return nil, err
		}
		commentGroups, err := loadCommentGroups(dir, dataset)
		if err != nil {
			return nil, err
		}
		fileDiffs, err := loadFileDiffs(dir, dataset)
		if err != nil {
			return nil, err
		}
		pull := Pull{
			Meta:          meta,
			CommentGroups: commentGroups,
			FileDiffs:     fileDiffs,
		}
		pulls = append(pulls, pull)
	}
	return pulls, nil
}
