package localtraining

import (
	"math"
	"math/rand"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
	"github.com/kiteco/kiteco/kite-golib/lexicalv0/bpe"
)

// NewVocab ...
type NewVocab struct {
	NewEntries    []string
	NewIDs        []int
	NewEmbeddings [][]float32
}

// VocabInit ...
type VocabInit string

const (
	// AverageParents ...
	AverageParents VocabInit = "average_parents"
	// MaximumParents ...
	MaximumParents VocabInit = "maximum_parents"
	// SameVarRandom ...
	SameVarRandom VocabInit = "same_variance_random"
)

// InitializeNewVocab builds a new vocab with embeddings
func InitializeNewVocab(originalEncoder, newEncoder *lexicalv0.FileEncoder, vocabInit VocabInit, originalEmbd [][]float32, mergedPairs []bpe.MergedPair) (NewVocab, error) {
	embdSize := len(originalEmbd[0])
	newEmbd := make([][]float32, newEncoder.Size())

	// Add empty embeddings for any mismatch in size. Currently, the only known case
	// of this happening is when the model was trained w/out a SEP token, but the
	// new encoder now expects it.
	for len(originalEmbd) < originalEncoder.Size() {
		originalEmbd = append(originalEmbd, make([]float32, embdSize))
	}

	// Copy all old embeddings over
	for i := 0; i < originalEncoder.Size(); i++ {
		if newEncoder.IsLexical(i) {
			newEmbd[i] = originalEmbd[i]
			continue
		}
		word := originalEncoder.IDToString[i]
		nID := newEncoder.StringToID[word]
		newEmbd[nID] = originalEmbd[i]
	}

	// Initialize new embeddings
	var nIDs []int
	var newEntries []string
	switch vocabInit {
	case AverageParents:
		for _, l := range mergedPairs {
			nID, aEmbd, bEmbd := getParentsEmbd(newEncoder, l, newEmbd)
			if len(aEmbd) != len(bEmbd) {
				return NewVocab{}, errors.New("embedding lengths different for parents")
			}
			nIDs = append(nIDs, nID)
			newEntries = append(newEntries, l.Joined)
			newEmbd[nID] = average(aEmbd, bEmbd)
		}
	case MaximumParents:
		for _, l := range mergedPairs {
			nID, aEmbd, bEmbd := getParentsEmbd(newEncoder, l, newEmbd)
			if len(aEmbd) != len(bEmbd) {
				return NewVocab{}, errors.New("embedding lengths different for parents")
			}
			nIDs = append(nIDs, nID)
			newEntries = append(newEntries, l.Joined)
			newEmbd[nID] = maximum(aEmbd, bEmbd)
		}
	case SameVarRandom:
		originalVariance := featureVariance(originalEmbd)
		for _, l := range mergedPairs {
			nWord := l.Joined
			nID := newEncoder.StringToID[nWord]
			nIDs = append(nIDs, nID)
			newEntries = append(newEntries, nWord)
			newEmbd[nID] = uniformWithVar(originalVariance)
		}
	}

	// Sanity check
	for i, e := range newEmbd {
		if len(e) == 0 {
			return NewVocab{}, errors.New("no embedding for %s", newEncoder.IDToString[i])
		}
	}

	return NewVocab{
		NewEntries:    newEntries,
		NewIDs:        nIDs,
		NewEmbeddings: newEmbd,
	}, nil
}

func getParentsEmbd(encoder *lexicalv0.FileEncoder, l bpe.MergedPair, newEmbd [][]float32) (int, []float32, []float32) {
	nWord, a, b := l.Joined, l.Parent1, l.Parent2
	nID := encoder.StringToID[nWord]
	aID, bID := encoder.StringToID[a], encoder.StringToID[b]
	aEmbd, bEmbd := newEmbd[aID], newEmbd[bID]
	return nID, aEmbd, bEmbd
}

func average(a, b []float32) []float32 {
	var mean []float32
	for i := 0; i < len(a); i++ {
		mean = append(mean, (a[i]+b[i])/2)
	}
	return mean
}

func maximum(a, b []float32) []float32 {
	var maxes []float32
	for i := 0; i < len(a); i++ {
		max := a[i]
		if b[i] > a[i] {
			max = b[i]
		}
		maxes = append(maxes, max)
	}
	return maxes
}

func uniformWithVar(variance []float32) []float32 {
	var vector []float32
	for _, v := range variance {
		vector = append(vector, sampleUniformWithVar(v))
	}
	return vector
}

func sampleUniformWithVar(variance float32) float32 {
	// Uniform distribution in [-a, a] has variance a^2/3
	// Get a from the variance first, then generate sample
	a := math.Sqrt(float64(3 * variance))
	return float32(2*a*rand.Float64() - a)
}

func featureVariance(originalEmbeddings [][]float32) []float32 {
	// compute variance for each column (feature dimension)
	var means, sqs []float32
	for row, vals := range originalEmbeddings {
		if row == 0 {
			means = make([]float32, len(vals))
			sqs = make([]float32, len(vals))
		}
		for col, val := range vals {
			means[col] += val
			sqs[col] += val * val
		}
	}

	invNumRows := 1. / float32(len(originalEmbeddings))
	for col := 0; col < len(means); col++ {
		mean := means[col] * invNumRows
		variance := (sqs[col] * invNumRows) - (mean * mean)
		means[col] = variance
	}

	return means
}
