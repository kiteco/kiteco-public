package curation

// Suggestions contain query suggestions for a langauge identifier.
type Suggestions struct {
	Ident       string // eg. os, os.path, os.path.exists
	Language    string // "python" for all now
	Package     string // package for the Ident
	Source      string // "google" or "bing" for now
	Suggestions []string
}

// SuggestionScore represents a query suggestion and its score
type SuggestionScore struct {
	Query     string   // so post title or google's suggested query
	Tokens    []string // tokenized query
	Source    string   // it's either "so" or "google" for now
	Package   string   // the package the query refers to
	ViewCount int      // view count
	VoteCount int      // vote count
	URL       string   // url to the post
}

// ByScore implements the sort interface
type ByScore []*SuggestionScore

func (b ByScore) Len() int      { return len(b) }
func (b ByScore) Swap(i, j int) { b[j], b[i] = b[i], b[j] }
func (b ByScore) Less(i, j int) bool {
	if b[i].ViewCount < b[j].ViewCount {
		return true
	}
	if b[i].ViewCount == b[j].ViewCount {
		if b[i].VoteCount < b[j].VoteCount {
			return true
		}
	}
	return false
}
