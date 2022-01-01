package bench

import (
	"encoding/gob"
	"os"
)

func init() {
	gob.Register([][]int32{})
	gob.Register([][2]int32{})
	gob.Register([][]float32{})
}

// FeedRecord contains the feeds/fetches that are input into a Tensorflow model.
type FeedRecord struct {
	Feeds   map[string]interface{}
	Fetches []string
}

// SaveFeedRecords to a gob file
func SaveFeedRecords(filename string, recs []FeedRecord) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := gob.NewEncoder(f).Encode(recs); err != nil {
		return err
	}
	return nil
}

// LoadFeedRecords from a gob file
func LoadFeedRecords(filename string) ([]FeedRecord, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var recs []FeedRecord
	if err := gob.NewDecoder(f).Decode(&recs); err != nil {
		return nil, err
	}
	return recs, nil
}
