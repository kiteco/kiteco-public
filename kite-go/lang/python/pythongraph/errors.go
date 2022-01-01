package pythongraph

import (
	"fmt"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

// TrainSampleErr stores information about the type of error that occured
// while building a particular train sample
type TrainSampleErr struct {
	Err    error
	Hash   string
	Symbol pythonresource.Symbol

	// ExprSubTaskType associated with the error if this was
	// from an infer expr train sample
	ExprSubTaskType ExprSubTaskType
}

// Error implements error
func (e TrainSampleErr) Error() string {
	if e.ExprSubTaskType != "" {
		return fmt.Sprintf("symbol: %s, hash: %s, subtask: %s, err: %v", e.Symbol, e.Hash, e.Err, e.ExprSubTaskType)
	}
	return fmt.Sprintf("symbol: %s, hash: %s, err: %v", e.Symbol, e.Hash, e.Err)
}

// TrainSampleErrs stores information about a set of training sample errors
type TrainSampleErrs []TrainSampleErr

// Error implements error
func (es TrainSampleErrs) Error() string {
	var parts []string
	for _, e := range es {
		parts = append(parts, e.Error())
	}
	return strings.Join(parts, ",")
}
