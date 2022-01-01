package clustering

import (
	"testing"
)

// TestFindClosestMembers tests if we can find the n-th closest members to the center of a cluster.
func TestFindClosestMembers(t *testing.T) {
	dp1 := newBasicNodeWithLen(nil)
	dp2 := newBasicNodeWithLen(nil)
	dp3 := newBasicNodeWithLen([]int{1, 2, 3, 1, 2, 3})
	dp4 := newBasicNodeWithLen([]int{1, 2, 3})

	data1 := []*BasicNode{dp1, dp2, dp3}

	constant := 2.0 / 3.0
	centerFeat := newBasicNodeWithLen(nil)
	centerFeat.FeatVec[1] = constant
	centerFeat.FeatVec[2] = constant
	centerFeat.FeatVec[3] = constant

	cluster1 := &Cluster{
		Members:  data1,
		Centroid: centerFeat,
	}

	closests := cluster1.FindClosestMembers(0)
	if closests != nil {
		t.Errorf("Expected closest to be nil.\n")
	}

	closests = cluster1.FindClosestMembers(1)
	if !Equal(closests[0], dp1) {
		t.Errorf("Closest member should be an all-zero vec.\n")
	}

	closests = cluster1.FindClosestMembers(5)
	if len(closests) != 3 {
		t.Errorf("Expected to get 3 closest members, but got %d.\n", len(closests))
	}

	data2 := []*BasicNode{dp3, dp4}

	constant = 1.5
	centerFeat = newBasicNodeWithLen(nil)
	centerFeat.FeatVec[1] = constant
	centerFeat.FeatVec[2] = constant
	centerFeat.FeatVec[3] = constant

	cluster2 := &Cluster{
		Members:  data2,
		Centroid: centerFeat,
	}

	closests = cluster2.FindClosestMembers(1)
	if !Equal(closests[0], dp3) {
		t.Errorf("Expected dp3 to be the closest data point to the center of cluster2.\n")
	}
}

// TestClearMembers tests if clearMembers clears the Members field of a cluster.
func TestClearMembers(t *testing.T) {
	dp1 := newBasicNodeWithLen(nil)
	dp2 := newBasicNodeWithLen(nil)

	data1 := []*BasicNode{dp1, dp2}
	cluster1 := &Cluster{
		Members: data1,
	}

	expVal := 2
	recVal := len(cluster1.Members)
	if expVal != recVal {
		t.Errorf("Expected cluster1 to have %d members, but got %d.\n", expVal, recVal)
	}

	cluster1.ClearMembers()
	expVal = 0
	recVal = len(cluster1.Members)
	if expVal != recVal {
		t.Errorf("Expected cluster1 to have %d members, but got %d.\n", expVal, recVal)
	}
}

func newBasicNodeWithLen(indices []int) *BasicNode {
	vec := make([]float64, 100)
	for _, i := range indices {
		vec[i] += 1.0
	}
	return &BasicNode{
		FeatVec: vec,
	}
}

// TestBasicNodeValues tests if BasicNode.Values returns featVec correctly.
func TestBasicNodeValues(t *testing.T) {
	f := []float64{0.0, 1.0, 0.0, 1.0, 0.0}
	n := &BasicNode{
		FeatVec: f,
	}
	feat := n.Values()
	expVal := len(feat)
	recVal := len(f)
	if expVal != recVal {
		t.Errorf("Expected n.Values() to have len %d, bug got %d.\n", expVal, recVal)
	}
	for i := range feat {
		if feat[i] != f[i] {
			t.Errorf("Expected n.Values() to be identical to f.\n")
		}
	}
}
