package decisiontree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDepthOne(t *testing.T) {
	node := Node{
		FeatureIndex: 0,
		Threshold:    2.5,
		LeftChild:    0,
		LeftIsLeaf:   true,
		RightChild:   1,
		RightIsLeaf:  true,
	}
	tree := DecisionTree{
		Nodes:       []Node{node},
		Outputs:     []float64{-3., 11.},
		FeatureSize: 2,
		Depth:       1,
	}
	x1 := []float64{1., 0.}
	x2 := []float64{5., 0.}
	assert.Equal(t, 0, tree.Bin(x1), "")
	assert.Equal(t, -3., tree.Evaluate(x1), "")
	assert.Equal(t, 1, tree.Bin(x2), "")
	assert.Equal(t, 11., tree.Evaluate(x2), "")
}

func TestDepthTwo(t *testing.T) {
	root := Node{
		FeatureIndex: 0,
		Threshold:    2.5,
		LeftChild:    1,
		LeftIsLeaf:   false,
		RightChild:   2,
		RightIsLeaf:  false,
	}
	left := Node{
		FeatureIndex: 1,
		Threshold:    0.,
		LeftChild:    0,
		LeftIsLeaf:   true,
		RightChild:   1,
		RightIsLeaf:  true,
	}
	right := Node{
		FeatureIndex: 1,
		Threshold:    1.,
		LeftChild:    2,
		LeftIsLeaf:   true,
		RightChild:   3,
		RightIsLeaf:  true,
	}
	tree := DecisionTree{
		Nodes:       []Node{root, left, right},
		FeatureSize: 2,
		Depth:       2,
	}
	assert.Equal(t, 0, tree.Bin([]float64{1., -1.}), "")
	assert.Equal(t, 1, tree.Bin([]float64{1., 1.}), "")
	assert.Equal(t, 2, tree.Bin([]float64{5., -2.}), "")
	assert.Equal(t, 3, tree.Bin([]float64{5., 2.}), "")
}
