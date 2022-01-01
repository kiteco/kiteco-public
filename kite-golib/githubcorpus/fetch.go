package githubcorpus

import (
	"path/filepath"

	"github.com/google/go-github/github"
)

// FetchCommits populates the full commit for each input commit along with the content for the base of each file modified in each commit.
// - dataFiles may be nil, otherwise it is modified.
func (r *APICorpus) FetchCommits(pr *github.PullRequest, commits []*github.RepositoryCommit, dataFiles map[string][]byte) ([]*github.RepositoryCommit, map[string][]byte) {
	owner := pr.GetBase().GetUser().GetLogin()
	name := pr.GetBase().GetRepo().GetName()

	if dataFiles == nil {
		dataFiles = make(map[string][]byte)
	}

	contents := contents{
		owner:  owner,
		name:   name,
		client: r.client,
		limit:  r.limit,
	}

	var fullCommits []*github.RepositoryCommit
	for _, c := range commits {
		cc, err := contents.GetCommit(c.GetSHA())
		if err != nil {
			continue
		}

		if len(cc.Files) == 0 {
			continue
		}

		if len(cc.Parents) != 1 {
			MergeCommitRatio.Hit()
			// skip merge commits
			continue
		}
		MergeCommitRatio.Miss()
		parent := cc.Parents[0]

		var files []github.CommitFile
		for _, f := range cc.Files {
			key := contentsKey(parent.GetSHA(), f.GetFilename())
			if _, ok := dataFiles[key]; ok {
				files = append(files, f)
				continue
			}

			buf, err := GetSourceState(cc, f.GetFilename(), contents)
			if err != nil {
				continue
			}

			files = append(files, f)
			dataFiles[key] = buf
		}
		cc.Files = files

		if len(cc.Files) == 0 {
			FullCommitSuccessRate.Miss()
			continue
		}
		FullCommitSuccessRate.Hit()

		fullCommits = append(fullCommits, cc)
	}

	return fullCommits, dataFiles
}

// FetchDataFiles ...
func (r *APICorpus) FetchDataFiles(pr *github.PullRequest, files []*github.CommitFile) map[string][]byte {
	owner := pr.GetBase().GetUser().GetLogin()
	name := pr.GetBase().GetRepo().GetName()

	fetchContent := func(sha, fn string) ([]byte, error) {
		return contents{
			owner:  owner,
			name:   name,
			client: r.client,
			limit:  r.limit,
		}.GetFile(sha, fn)
	}

	baseFNs := make(map[string]bool)
	headFNs := make(map[string]bool)

	for _, f := range files {
		if _, ok := allowedExts[filepath.Ext(f.GetFilename())]; !ok {
			continue
		}
		switch f.GetStatus() {
		case "modified":
			headFNs[f.GetFilename()] = true
			baseFNs[f.GetFilename()] = true
		case "added":
			headFNs[f.GetFilename()] = true
		case "deleted":
			// Ignore deleted files for now?
		}
	}

	baseSHA, headSHA := BaseAndHeadSHA(pr)

	contents := make(map[string][]byte)
	for fn := range baseFNs {
		buf, err := fetchContent(baseSHA, fn)
		if err != nil {
			continue
		}
		key := contentsKey(baseSHA, fn)
		contents[key] = buf
	}
	for fn := range headFNs {
		buf, err := fetchContent(headSHA, fn)
		if err != nil {
			continue
		}
		key := contentsKey(headSHA, fn)
		contents[key] = buf
	}

	return contents
}

// --

// IsCode returns whether a filename is source code
func IsCode(fn string) bool {
	return allowedExts[filepath.Ext(fn)]
}

var allowedExts = map[string]bool{
	".js":    true,
	".jsx":   true,
	".vue":   true,
	".css":   true,
	".less":  true,
	".html":  true,
	".py":    true,
	".java":  true,
	".sh":    true,
	".cs":    true,
	".ts":    true,
	".tsx":   true,
	".php":   true,
	".c":     true,
	".cc":    true,
	".cpp":   true,
	".h":     true,
	".hpp":   true,
	".m":     true,
	".go":    true,
	".scala": true,
	".kt":    true,
	".rb":    true,
}
