package search

import "github.com/kiteco/kiteco/kite-go/stackoverflow"

// Index is interface for an index for so pages.
type Index interface {
	Search(query string, st stackoverflow.SearchType, numResults int) ([]int64, error)
}
