package ranking

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"sort"

	"github.com/kiteco/kiteco/kite-golib/decisiontree"
)

// DataPoint abstracts a data point that is to be ranked by the model.
type DataPoint struct {
	ID       int       // index of this DataPoint
	Name     string    // name of this data point if any
	Score    float64   // score that this data point gets
	Features []float64 // features of this data point
}

// ByScore implements the sort interface, so that we can rank
// the data points by score.
type ByScore []*DataPoint

func (d ByScore) Len() int           { return len(d) }
func (d ByScore) Swap(i, j int)      { d[j], d[i] = d[i], d[j] }
func (d ByScore) Less(i, j int) bool { return d[i].Score < d[j].Score }

// Normalizer normalizes input data as follows: x = (x + Offset) * Scale.
type Normalizer struct {
	Offset []float64
	Scale  []float64
}

// Normalize normalizes the input feature vector by the normalizer's
// offset vector and scale vector.
func (n *Normalizer) Normalize(features []float64) []float64 {
	if len(features) != len(n.Offset) {
		log.Fatalln("feature length is not equal to length of offset", len(features), len(n.Offset))
	}
	if len(features) != len(n.Scale) {
		log.Fatalln("feature length is not equal to length of scale")
	}
	for i := 0; i < len(features); i++ {
		features[i] = (features[i] + n.Offset[i]) * n.Scale[i]
	}
	return features
}

// Print prints co-efficients of the normalizer.
func (n *Normalizer) Print() {
	log.Println("Offset:", n.Offset)
	log.Println("Scale:", n.Scale)
}

// Ranker contains the scorer that is used to compute scores for the documents
// to be ranked.
type Ranker struct {
	Scorer        Scorer
	ScorerType    string
	Normalizer    *Normalizer
	FeatureLabels []string
}

// Rank ranks the input data points using the scores that the ranker generates for each data point.
func (r *Ranker) Rank(data []*DataPoint) {
	for _, dp := range data {
		dp.Score = r.evaluate(dp.Features)
	}
	sort.Sort(sort.Reverse(ByScore(data)))
}

// Evaluate normalizes the feature vector and evaluate the score
// of the feature vector computed by using the linear model.
func (r *Ranker) evaluate(features []float64) float64 {
	features = r.Normalizer.Normalize(features)
	return r.Scorer.Evaluate(features)
}

// Scorer defines the functions that a scorer (for example, a linear scorer or a tree scorer) should implement.
// Evaluate computes the score of a feature vector using the scorer.
type Scorer interface {
	Evaluate([]float64) float64
	Print()
}

// LinearScorer represents a linear regressor
type LinearScorer struct {
	Weights []float64
}

// Print prints the weights of the linear model.
func (l *LinearScorer) Print() {
	log.Println("Weights:", l.Weights)
}

// Evaluate returns the inner product of the weights and the input feature vector.
func (l *LinearScorer) Evaluate(features []float64) float64 {
	if len(features) != len(l.Weights) {
		log.Fatal("feature length is not equal to lengh of offsets", len(features), len(l.Weights))
	}
	var score float64
	for i, f := range features {
		score += f * l.Weights[i]
	}
	return score
}

// A Kernel represents a Mercer kernel function.
type Kernel interface {
	Evaluate(a, b []float64) float64
}

// KernelScorer computes scores using Kernel.
type KernelScorer struct {
	Support [][]float64 // the regression vectors
	Coefs   []float64   // the coefficients for each regression vector
	Kernel  Kernel
}

// Print prints out coefficients of a kernel scorer.
func (s *KernelScorer) Print() {
	log.Println("Coefs:", s.Coefs)
}

// Evaluate maps a feature vector to a score.
func (s *KernelScorer) Evaluate(features []float64) float64 {
	if len(s.Support) > 0 && len(features) != len(s.Support[0]) {
		log.Fatalf("expected feature size %d but receieved %d\n", len(s.Support[0]), len(features))
	}

	var score float64
	for i, c := range s.Coefs {
		score += c * s.Kernel.Evaluate(s.Support[i], features)
	}
	return score
}

// NewKernelScorer returns a pointer to a new KernelScorer.
func NewKernelScorer(support [][]float64, coefs []float64, kernel Kernel) *KernelScorer {
	if len(support) != len(coefs) {
		log.Fatalf("Length mismatch: support=%d, coefs=%d\n", len(support), len(coefs))
	}
	return &KernelScorer{
		Support: support,
		Coefs:   coefs,
		Kernel:  kernel,
	}
}

// RbfKernel implements the radial basis function.
type RbfKernel struct {
	Gamma float64 // Coefficient in exponent (always negative)
}

// NewRbfKernelScorer returns a RbfKernelScorer
func NewRbfKernelScorer(support [][]float64, coefs []float64, gamma float64) *KernelScorer {
	return NewKernelScorer(support, coefs, &RbfKernel{gamma})
}

// Evaluate returns the result of RBF(a, b)
func (k *RbfKernel) Evaluate(a, b []float64) float64 {
	if len(a) != len(b) {
		log.Fatalf("Length mismatch: %d vs %d\n", len(a), len(b))
	}
	// Compute the squared euclidean distance between A and B
	var ssd float64
	for i := range a {
		ssd += (a[i] - b[i]) * (a[i] - b[i])
	}
	return math.Exp(k.Gamma * ssd)
}

// NewRankerFromJSON loads a ranker from json.
func NewRankerFromJSON(r io.Reader) (*Ranker, error) {
	var intermediate struct {
		Normalizer    *Normalizer
		Scorer        json.RawMessage
		ScorerType    string
		FeatureLabels []string
	}

	d := json.NewDecoder(r)
	err := d.Decode(&intermediate)
	if err != nil {
		return nil, fmt.Errorf("error decoding top-level json: %v", err)
	}

	ranker := &Ranker{
		Normalizer:    intermediate.Normalizer,
		ScorerType:    intermediate.ScorerType,
		FeatureLabels: intermediate.FeatureLabels,
	}

	switch intermediate.ScorerType {
	case "Linear":
		var scorer LinearScorer
		err := json.Unmarshal(intermediate.Scorer, &scorer)
		if err != nil {
			return nil, fmt.Errorf("error deserializing linear scorer: %v", err)
		}
		ranker.Scorer = &scorer

	case "RbfKernel":
		var params struct {
			Support [][]float64
			Coefs   []float64
			Gamma   float64
		}
		err := json.Unmarshal(intermediate.Scorer, &params)
		if err != nil {
			return nil, fmt.Errorf("error deserializing RBF kernel scorer: %v", err)
		}
		ranker.Scorer = NewRbfKernelScorer(params.Support, params.Coefs, params.Gamma)

	case "TreeEnsemble":
		ranker.Scorer, err = decisiontree.Load(bytes.NewBuffer(intermediate.Scorer))
		if err != nil {
			return nil, fmt.Errorf("error deserializing tree ensemble scorer: %v", err)
		}

	default:
		return nil, fmt.Errorf("found unknown scorer type '%s'", intermediate.ScorerType)
	}

	return ranker, nil
}

// NewScorerNormalizerFromJSON returns a Scorer and *Normalizer read from json.
func NewScorerNormalizerFromJSON(r io.Reader) (Scorer, *Normalizer, error) {
	var intermediate struct {
		Normalizer    *Normalizer
		Scorer        json.RawMessage
		ScorerType    string
		FeatureLabels []string
	}

	d := json.NewDecoder(r)
	err := d.Decode(&intermediate)
	if err != nil {
		return nil, nil, fmt.Errorf("error decoding top-level json: %v", err)
	}

	normalizer := intermediate.Normalizer
	var scorer Scorer

	switch intermediate.ScorerType {
	case "Linear":
		var scorerLin LinearScorer
		err := json.Unmarshal(intermediate.Scorer, &scorerLin)
		if err != nil {
			return nil, nil, fmt.Errorf("error deserializing linear scorer: %v", err)
		}
		scorer = &scorerLin

	case "RbfKernel":
		var params struct {
			Support [][]float64
			Coefs   []float64
			Gamma   float64
		}
		err := json.Unmarshal(intermediate.Scorer, &params)
		if err != nil {
			return nil, nil, fmt.Errorf("error deserializing RBF kernel scorer: %v", err)
		}
		scorer = NewRbfKernelScorer(params.Support, params.Coefs, params.Gamma)

	case "TreeEnsemble":
		scorer, err = decisiontree.Load(bytes.NewBuffer(intermediate.Scorer))
		if err != nil {
			return nil, nil, fmt.Errorf("error deserializing tree ensemble scorer: %v", err)
		}

	default:
		return nil, nil, fmt.Errorf("found unknown scorer type '%s'", intermediate.ScorerType)
	}
	return scorer, normalizer, nil
}
