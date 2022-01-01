package ranking

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
)

// BinaryClassifier is a binary classifier (could be an SVM classifier or a Logistic Regression classifier ect)
type BinaryClassifier struct {
	ScorerType string
	Scorer     Scorer
}

// PredictProba returns the probability of feat to be classified as class 1 (v.s. 0) given the model.
func (c *BinaryClassifier) PredictProba(feat []float64) float64 {
	return c.Scorer.Evaluate(feat)
}

// NewBinaryClassifierFromJSON loads a classifer from JSON.
func NewBinaryClassifierFromJSON(r io.Reader) (*BinaryClassifier, error) {
	var intermediate struct {
		Scorer     json.RawMessage
		ScorerType string
	}

	decoder := json.NewDecoder(r)
	err := decoder.Decode(&intermediate)
	if err != nil {
		return nil, err
	}
	classifier := &BinaryClassifier{
		ScorerType: intermediate.ScorerType,
	}

	switch classifier.ScorerType {
	case "logistic_regression":
		var multiclass struct {
			Bias  []float64
			Coefs [][]float64
		}

		err := json.Unmarshal(intermediate.Scorer, &multiclass)
		if err != nil {
			return nil, err
		}

		if len(multiclass.Bias) == 0 {
			return nil, fmt.Errorf("length of Bias is 0")
		}
		if len(multiclass.Coefs) == 0 {
			return nil, fmt.Errorf("length of coefficients is 0")
		}
		classifier.Scorer = &LogisticRegression{
			Bias:  multiclass.Bias[0],
			Coefs: multiclass.Coefs[0],
		}
	}
	return classifier, nil
}

// LogisticRegression represents a binary logistic regression classifier
type LogisticRegression struct {
	Bias  float64
	Coefs []float64
}

// Print prints coefficients of a logistic regression classifier.
func (l *LogisticRegression) Print() {
	log.Println("Bias", l.Bias)
	log.Println("Coefs", l.Coefs)
}

// Evaluate returns the probability of the feature vector to be classified as class 1 (v.s. 0) given
// the model.
func (l *LogisticRegression) Evaluate(feats []float64) float64 {
	if len(feats) != len(l.Coefs) {
		log.Fatalln("feature length is not equal to length of coefs", len(feats), len(l.Coefs))
	}
	var score float64
	for i := range l.Coefs {
		score += feats[i] * l.Coefs[i]
	}
	score += l.Bias
	return 1 / (1 + math.Exp(-score))
}
