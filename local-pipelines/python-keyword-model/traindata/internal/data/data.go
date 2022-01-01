package data

import (
	"sync"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonkeyword"
)

// Record is a serialized row in the data set,
// these are direclty fed into the python training script as json.
type Record struct {
	Features        pythonkeyword.Features
	IsKeyword       bool
	KeywordCategory int // 1-30  and -1 corresponds to keywords ignored by the model (currently global and nonlocal)
	Literal         string
}

// ItemCounter stores count for each keywords when building a frequency table
type ItemCounter struct {
	Keywords     map[int]uint64 `json:"keywords"`
	Mutex        sync.Mutex     `json:"-"`
	Keyword      uint64         `json:"keyword"`
	Name         uint64         `json:"name"`
	FilesScanned uint64         `json:"files_scanned"`
}

// NewItemCounter build a new ItemCounter object with an initialized Keywords map and the number of file scanned set
func NewItemCounter(fileCount uint64) *ItemCounter {
	return &ItemCounter{Keywords: make(map[int]uint64),
		FilesScanned: fileCount}
}

// KeywordExample contains the required data to generate example for model-test
// If an example file output is specified during the train data generation, all items will be also put in this
// example file with a snippet and the expected keyword
type KeywordExample struct {
	CodeSnippet     string
	KeywordCategory int
}

// SampleTag implements pipeline.Sample
func (KeywordExample) SampleTag() {}

// SamplingRates contains all subsampling rate for each keywords and for name vs keyword balancing.
type SamplingRates struct {
	NameKeyword float64
	Keywords    map[int]float64
}

// SampleTag implements pipeline.Sample
func (Record) SampleTag() {}
