package pythoncuration

import (
	"github.com/kiteco/kiteco/kite-go/dynamicanalysis"
	"github.com/kiteco/kiteco/kite-go/ranking"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// Ranker wraps the ranking model and the featurer for the
// code example ranker for passive search.
type Ranker struct {
	model    *ranking.Ranker
	featurer *ExampleFeaturer
}

// NewRankerFromFile returns a pointer to a new Ranker object by loading
// both the ranking model and the featurer from a json file (loading
// the featurer from a JSON file is in PR #674).
func NewRankerFromFile(modelPath, featurerPath string) (*Ranker, error) {
	in, err := fileutil.NewCachedReader(modelPath)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	model, err := ranking.NewRankerFromJSON(in)
	if err != nil {
		return nil, err
	}

	fin, err := fileutil.NewCachedReader(featurerPath)
	if err != nil {
		return nil, err
	}
	defer fin.Close()

	featurer, err := NewExampleFeaturerFromGOB(fin)
	if err != nil {
		return nil, err
	}

	return &Ranker{
		model:    model,
		featurer: featurer,
	}, nil
}

// Features converts each code example into a feature vector.
func (pr *Ranker) Features(ident string, examples []*Snippet, references []*dynamicanalysis.ResolvedSnippet) []*ranking.DataPoint {
	var data []*ranking.DataPoint
	for i, eg := range examples {
		feats := pr.featurer.Features(ident, eg, references[i])
		data = append(data, &ranking.DataPoint{
			ID:       i,
			Features: feats,
		})
	}
	return data
}

// Rank ranks the given code examples for the given identifier name
// for passive search.
func (pr *Ranker) Rank(ident string, examples []*Snippet, references []*dynamicanalysis.ResolvedSnippet) []*Snippet {
	data := pr.Features(ident, examples, references)
	pr.model.Rank(data)
	var sortedExamples []*Snippet
	for _, d := range data {
		sortedExamples = append(sortedExamples, examples[d.ID])
	}
	return sortedExamples
}
