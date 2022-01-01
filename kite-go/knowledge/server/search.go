package knowledge

import (
	"net/http"
	"strings"

	"github.com/kiteco/kiteco/kite-go/knowledge/search"
)

// SearchDisplay ...
type SearchDisplay struct {
	Query    string
	HasPulls bool
	HasFiles bool
	Pulls    []search.Pull
	Files    []search.File
}

func (s *Server) runSearch(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")
	if query == "" {
		s.templates.Render(w, "search.html", SearchDisplay{})
		return
	}
	searchOpts := search.Options{
		PullsDir: s.app.paths.ClosedPullsPath,
	}
	results, err := search.Search(query, searchOpts)
	if err != nil {
		s.showError(w, err)
		return
	}
	display := makeSearchDisplay(query, results)
	html := "pulls.html"
	if r.FormValue("mode") == "Files" {
		html = "files.html"
	}
	err = s.templates.Render(w, html, display)
	if err != nil {
		s.showError(w, err)
	}
}

func makeSearchDisplay(query string, results search.Results) SearchDisplay {
	var pulls []search.Pull
	for i, pull := range results.Pulls {
		if i == maxPulls {
			break
		}
		pulls = append(pulls, newPullDisplay(pull))
	}
	var files []search.File
	for i, file := range results.Files {
		if i == maxFiles {
			break
		}
		files = append(files, newFileDisplay(file))
	}
	return SearchDisplay{
		Query:    query,
		HasPulls: len(pulls) > 0,
		HasFiles: len(files) > 0,
		Pulls:    pulls,
		Files:    files,
	}
}

func newPullDisplay(pull search.Pull) search.Pull {
	displayPull := pull
	displayPull.Meta.Body = formatBody(pull.Meta.Body)
	if len(pull.CommentGroups) > maxCommentGroups {
		displayPull.CommentGroups = pull.CommentGroups[:maxCommentGroups]
	}
	for i, group := range pull.CommentGroups {
		if len(group.Comments) > maxComments {
			displayPull.CommentGroups[i].Comments = group.Comments[:maxComments]
		}
	}
	if len(pull.FileDiffs) > maxFileDiffs {
		displayPull.FileDiffs = pull.FileDiffs[:maxFileDiffs]
	}
	return displayPull
}

func newFileDisplay(file search.File) search.File {
	displayFile := file
	if len(file.Lines) > maxLines {
		displayFile.Lines = file.Lines[:maxLines]
	}
	return displayFile
}

func formatBody(body string) string {
	var lines []string
	for _, line := range strings.Split(body, "\r\n") {
		lines = append(lines, truncate(line, maxLineLength)...)
	}
	return strings.Join(lines, "\r\n")
}

func truncate(line string, maxLength int) []string {
	if len(line) <= maxLength {
		return []string{line}
	}
	var truncated []string
	var block []string
	var blockSize int
	words := strings.Split(line, " ")
	for i, word := range words {
		block = append(block, word)
		blockSize += 1 + len(word)
		if i < len(words)-1 && blockSize < maxLength {
			continue
		}
		truncated = append(truncated, strings.Join(block, " "))
		block = nil
		blockSize = 0
	}
	return truncated
}
