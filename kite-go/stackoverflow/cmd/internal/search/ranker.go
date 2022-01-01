package search

import (
	"encoding/gob"
	"io"
	"sort"

	"github.com/kiteco/kiteco/kite-go/ranking"
	"github.com/kiteco/kiteco/kite-go/stackoverflow"
	"github.com/kiteco/kiteco/kite-golib/tfidf"
)

// Ranker implements Ranker interface.
type Ranker struct {
	featurers  Featurers
	scorer     ranking.Scorer
	normalizer ranking.Normalizer
}

// NewRanker initializes a new Ranker using the data in the readers.
func NewRanker(modelReader, idfsReader io.Reader) (*Ranker, error) {
	decoder := gob.NewDecoder(idfsReader)
	var idfs map[string]*tfidf.IDFCounter
	err := decoder.Decode(&idfs)
	if err != nil {
		return nil, err
	}
	featurers, err := NewFeaturers(idfs)
	if err != nil {
		return nil, err
	}
	scorer, normalizer, err := ranking.NewScorerNormalizerFromJSON(modelReader)
	if err != nil {
		return nil, err
	}
	return &Ranker{
		featurers:  featurers,
		scorer:     scorer,
		normalizer: *normalizer,
	}, nil
}

type byScore []PageWithScore

func (bs byScore) Len() int {
	return len(bs)
}

func (bs byScore) Swap(i, j int) {
	bs[i], bs[j] = bs[j], bs[i]
}

func (bs byScore) Less(i, j int) bool {
	return bs[i].Score < bs[j].Score
}

// PageWithScore encapsulates a StackOverflowPage and any score
// data that was computed in the ranking process.
type PageWithScore struct {
	Page  *stackoverflow.StackOverflowPage
	Score float64
}

// Rank ranks the given pages relative to provided query.
func (r *Ranker) Rank(query string, pages []*stackoverflow.StackOverflowPage) {
	data := make([]PageWithScore, len(pages))
	for i, page := range pages {
		features := r.featurers.Features(query, Document{
			Page: page,
		})
		features = r.normalizer.Normalize(features)
		data[i] = PageWithScore{
			Page:  page,
			Score: r.scorer.Evaluate(features),
		}
	}
	sort.Sort(sort.Reverse(byScore(data)))
	for i := range pages {
		pages[i] = data[i].Page
	}
}
