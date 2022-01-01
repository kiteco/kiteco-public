package clustering

import (
	"math"
	"testing"
)

// TestCentroidDistanceSwap tests if Swap of centroidDistance swaps two nodes.
func TestCentroidDistanceSwap(t *testing.T) {
	dp1 := newBasicNodeWithLen(nil)
	dp2 := newBasicNodeWithLen([]int{1, 2, 3})

	centroid := newBasicNodeWithLen(nil)
	data := []Node{dp1, dp2}

	cd := newCentroidDistance(centroid, data)

	cd.Swap(0, 1)

	if !Equal(cd.members[0], dp2) {
		t.Errorf("Expected dp1 and dp2 to swap.\n")
	}
}

// TestCentroidLess tests if centroidDistance's implementation of Less is correct.
func TestCentroidDistanceSwapLess(t *testing.T) {
	dp1 := newBasicNodeWithLen(nil)
	dp2 := newBasicNodeWithLen([]int{1, 2, 3})

	centroid := newBasicNodeWithLen(nil)
	data := []Node{dp1, dp2}

	cd := newCentroidDistance(centroid, data)

	if !cd.Less(0, 1) {
		t.Errorf("Expected dp1 < dp2.\n")
	}
}

// TestNodesValues tests if Nodes.Values() correctly return the feature of the i-th node.
func TestNodesValues(t *testing.T) {
	dp1 := newBasicNodeWithLen(nil)
	dp2 := newBasicNodeWithLen([]int{1, 2, 3})

	data := []Node{dp1, dp2}
	nodes := nodes(data)

	node := nodes.Values(0)
	for _, v := range node {
		if v != 0 {
			t.Errorf("Expected nodes.Values(0) to return an all-zero vector.\n")
		}
	}
}

// TestDistance tests if Distance computes the distance of two nodes correctly.
func TestDistance(t *testing.T) {
	dp1 := newBasicNodeWithLen(nil)
	dp2 := newBasicNodeWithLen(nil)
	dp3 := newBasicNodeWithLen([]int{1, 2, 3})
	expVal := 0.0
	recVal := Distance(dp1, dp2)
	if math.Abs(expVal-recVal) > 1e-8 {
		t.Errorf("Expected distance between dp1 and dp2 is %f, but got %f.\n", expVal, recVal)
	}
	expVal = math.Sqrt(3.0)
	recVal = Distance(dp1, dp3)
	if math.Abs(expVal-recVal) > 1e-8 {
		t.Errorf("Expected distance between dp1 and dp3 is %f, but got %f.\n", expVal, recVal)
	}
}

// TestEqual tests if Equal functions correctly.
func TestEqual(t *testing.T) {
	dp1 := newBasicNodeWithLen(nil)
	dp2 := newBasicNodeWithLen(nil)
	dp3 := newBasicNodeWithLen([]int{1, 2, 3})

	if !Equal(dp1, dp2) {
		t.Errorf("Expected dp1 and dp2 to be equal.\n")
	}

	if Equal(dp1, dp3) {
		t.Errorf("Expected dp1 and dp3 to be unequal.\n")
	}
}
