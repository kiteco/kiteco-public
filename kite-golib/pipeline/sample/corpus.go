package sample

import (
	"time"

	"github.com/kiteco/kiteco/kite-golib/pipeline"
)

// Corpus of files
type Corpus interface {
	pipeline.Sample
	// ID that can be used to identify the corpus
	ID() string
	// List all files in the corpus, in alphabetical order
	List() ([]FileInfo, error)
	Get(filename string) ([]byte, error)
}

// FileInfo contains metadata for a file in a Corpus
type FileInfo struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
