package search

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/google/go-github/github"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

func loadPullMeta(dir string) (PullMeta, error) {
	path := filepath.Join(dir, "pull.json")
	reader, err := fileutil.NewReader(path)
	if err != nil {
		return PullMeta{}, err
	}
	defer reader.Close()
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return PullMeta{}, err
	}
	pr := new(github.PullRequest)
	err = json.Unmarshal(contents, pr)
	if err != nil {
		return PullMeta{}, err
	}
	return PullMeta{
		URL:    *pr.HTMLURL,
		Author: *pr.User.Login,
		Title:  *pr.Title,
		Body:   strings.TrimSpace(*pr.Body),
		Number: *pr.Number,
	}, nil
}

func loadCommentGroups(dir string, dataset []data) ([]CommentGroup, error) {
	grouper := make(map[string][]Comment)
	commentsDir := filepath.Join(dir, "comments")
	urls := make(map[string]string)
	loaded := make(map[string]bool)
	for _, d := range dataset {
		parent := filepath.Dir(d.path)
		if filepath.Dir(parent) != commentsDir {
			continue
		}
		if loaded[parent] {
			continue
		}
		comment, path, diff, err := loadComment(parent)
		if err != nil {
			return nil, err
		}
		grouper[diff] = append(grouper[diff], comment)
		urls[diff] = urlFromPath(path)
		loaded[parent] = true
	}
	var groups []CommentGroup
	for diff, comments := range grouper {
		group := CommentGroup{
			Diff:     diff,
			URL:      urls[diff],
			Comments: comments,
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func loadComment(path string) (Comment, string, string, error) {
	reader, err := fileutil.NewReader(filepath.Join(path, "comment.json"))
	if err != nil {
		return Comment{}, "", "", err
	}
	defer reader.Close()
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return Comment{}, "", "", err
	}
	var prComment github.PullRequestComment
	err = json.Unmarshal(contents, &prComment)
	if err != nil {
		return Comment{}, "", "", err
	}
	comment := Comment{
		Author:  *prComment.User.Login,
		Content: *prComment.Body,
	}
	return comment, *prComment.Path, *prComment.DiffHunk, nil
}

func loadFileDiffs(dir string, dataset []data) ([]FileDiff, error) {
	var fileDiffs []FileDiff
	fileDiffsDir := filepath.Join(dir, "files")
	loaded := make(map[string]bool)
	for _, d := range dataset {
		parent := filepath.Dir(d.path)
		if filepath.Dir(parent) != fileDiffsDir {
			continue
		}
		if loaded[parent] {
			continue
		}
		fileDiff, err := loadFileDiff(parent)
		if err != nil {
			return nil, err
		}
		fileDiffs = append(fileDiffs, fileDiff)
		loaded[parent] = true
	}
	return fileDiffs, nil
}

func loadFileDiff(path string) (FileDiff, error) {
	reader, err := fileutil.NewReader(filepath.Join(path, "diff.json"))
	if err != nil {
		return FileDiff{}, err
	}
	defer reader.Close()
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return FileDiff{}, err
	}
	var commitFile github.CommitFile
	err = json.Unmarshal(contents, &commitFile)
	if err != nil {
		return FileDiff{}, err
	}
	var diff string
	if commitFile.Patch != nil {
		diff = *commitFile.Patch
	}
	return FileDiff{
		URL:  urlFromPath(*commitFile.Filename),
		Diff: diff,
	}, nil
}

func urlFromPath(path string) string {
	return fileutil.Join("https://github.com/kiteco/kiteco/blob/master", filepath.ToSlash(path))
}
