package search

import (
	"regexp"
	"sort"
	"strings"
)

type appraiser struct {
	re   regexp.Regexp
	opts Options
}

func newAppraiser(query string, opts Options) (appraiser, error) {
	re, err := regexp.Compile(query)
	if err != nil {
		return appraiser{}, err
	}
	return appraiser{
		re:   *re,
		opts: opts,
	}, nil
}

func (a appraiser) match(content string) bool {
	if a.opts.CaseSensitive {
		return a.re.MatchString(content)
	}
	return a.re.MatchString(strings.ToLower(content))
}

func (a appraiser) rankPulls(pulls []Pull) {
	sort.Slice(pulls, func(i, j int) bool {
		return a.scorePull(pulls[i]) > a.scorePull(pulls[j])
	})
}

func (a appraiser) rankFiles(files []File) {
	sort.Slice(files, func(i, j int) bool {
		return a.scoreFile(files[i]) > a.scoreFile(files[j])
	})
}

func (a appraiser) rankCommentGroups(commentGroups []CommentGroup) {
	sort.Slice(commentGroups, func(i, j int) bool {
		return a.scoreCommentGroup(commentGroups[i]) > a.scoreCommentGroup(commentGroups[j])
	})
}

func (a appraiser) rankFileDiffs(fileDiffs []FileDiff) {
	sort.Slice(fileDiffs, func(i, j int) bool {
		return a.scoreFileDiff(fileDiffs[i]) > a.scoreFileDiff(fileDiffs[j])
	})
}

func (a appraiser) scorePull(pull Pull) int {
	return pull.Meta.Number
}

func (a appraiser) scoreFile(file File) int {
	distinctLines := make(map[string]bool)
	for _, line := range file.Lines {
		distinctLines[line.Content] = true
	}
	if a.match(file.URL) {
		return len(distinctLines) + 1
	}
	return len(distinctLines)
}

func (a appraiser) scoreFileDiff(fileDiff FileDiff) int {
	file := File{
		URL: fileDiff.URL,
	}
	raw := strings.Split(fileDiff.Diff, "\n")
	for i, content := range raw {
		if !a.match(content) {
			continue
		}
		line := Line{
			Number:  i,
			Content: content,
		}
		file.Lines = append(file.Lines, line)
	}
	return a.scoreFile(file)
}

func (a appraiser) scoreCommentGroup(commentGroup CommentGroup) int {
	file := File{}
	raw := strings.Split(commentGroup.Diff, "\n")
	for _, comment := range commentGroup.Comments {
		raw = append(raw, strings.Split(comment.Content, "\n")...)
	}
	for i, content := range raw {
		if !a.match(content) {
			continue
		}
		line := Line{
			Number:  i,
			Content: content,
		}
		file.Lines = append(file.Lines, line)
	}
	return a.scoreFile(file)
}

func (a appraiser) sampleFileDiffs(fileDiffs []FileDiff) {
	for i, fileDiff := range fileDiffs {
		fileDiffs[i] = FileDiff{
			URL:  fileDiff.URL,
			Diff: a.getRelevantBlocks(fileDiff.Diff),
		}
	}
}

func (a appraiser) getRelevantBlocks(content string) string {
	re := regexp.MustCompile("\n[+-| ]?\n")
	replaced := re.ReplaceAllString(content, "\n\n")
	blocks := strings.Split(replaced, "\n\n")
	var sampledBlocks []string
	for _, block := range blocks {
		if a.match(block) {
			sampledBlocks = append(sampledBlocks, block)
		}
	}
	return strings.Join(sampledBlocks, "\n\n")
}
