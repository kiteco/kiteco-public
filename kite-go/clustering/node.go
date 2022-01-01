package clustering

import (
	"log"
	"math"
	"reflect"
)

// Node represents a data point.
// An implementation of Node must implement the following functions.
type Node interface {
	// Values returns the feature values of the caller.
	Values() []float64
	// ID returns the ID of a node.
	ID() uint64
}

// nodes represent a set of data points.
type nodes []Node

// Len returns the # of data points in the set.
func (ns nodes) Len() int {
	return len(ns)
}

// Values is needed for using the "github.com/biogo/cluster/kmeans" library.
func (ns nodes) Values(i int) []float64 {
	return ns[i].Values()
}

// Distance computes the euclidean distance between two nodes.
func Distance(n1, n2 Node) float64 {
	f1 := n1.Values()
	f2 := n2.Values()
	if len(f1) != len(f2) {
		log.Fatalf("Can't compute distance between two vectors of different dimensions.\n")
	}
	var sum float64
	for i := range f1 {
		sum += (f1[i] - f2[i]) * (f1[i] - f2[i])
	}
	return math.Sqrt(sum)
}

// Equal checks if two nodes have the same feature values.
func Equal(n1, n2 Node) bool {
	return reflect.DeepEqual(n1.Values(), n2.Values())
}
