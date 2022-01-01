package distranking

import (
	"encoding/json"
	"io"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource/keytypes"
)

// Ranking maps distributions to a global ranking
type Ranking map[keytypes.Distribution]int

// New deserializes a new symbol index from an io.Reader
func New(r io.Reader) (Ranking, error) {
	type distAndRank struct {
		Distribution keytypes.Distribution
		Rank         int
	}

	var ranks []distAndRank
	err := json.NewDecoder(r).Decode(&ranks)
	if err != nil {
		return nil, err
	}

	rank := make(Ranking)
	for _, r := range ranks {
		rank[r.Distribution] = r.Rank
	}

	return rank, nil
}
