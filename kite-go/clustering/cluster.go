package clustering

import (
	"sort"
)

// centroidDistance wraps the members and the centroid of a cluster.
type centroidDistance struct {
	centroid Node
	members  []Node
}

func newCentroidDistance(centroid Node, members []Node) centroidDistance {
	return centroidDistance{
		centroid: centroid,
		members:  members,
	}
}

func (cd centroidDistance) Len() int { return len(cd.members) }
func (cd centroidDistance) Swap(i, j int) {
	cd.members[j], cd.members[i] = cd.members[i], cd.members[j]
}
func (cd centroidDistance) Less(i, j int) bool {
	return Distance(cd.centroid, cd.members[i]) < Distance(cd.centroid, cd.members[j])
}

// Cluster represents a cluster of DataPoints.
// centroid is the center of the cluster.
// members are the Nodes that belong to this cluster.
type Cluster struct {
	Members  []*BasicNode `json:"members"`
	Centroid *BasicNode   `json:"centroid"`
	Weight   float64      `json:"weight"`
	ID       uint64       `json:"id"`
}

// BasicNode is a struct that wraps a feature vector, which is represented as a slice of float64.
type BasicNode struct {
	UniqueID uint64    `json:"uniqueID,omitempty"`
	FeatVec  []float64 `json:"featVec,omitempty"`
}

// Values implements Node.
func (c *BasicNode) Values() []float64 {
	return c.FeatVec
}

// ID implements Node.
func (c *BasicNode) ID() uint64 {
	return c.UniqueID
}

// FindClosestMembers returns data points that are close to the centroid of the cluster.
func (c *Cluster) FindClosestMembers(n int) []Node {
	if n <= 0 {
		return nil
	}
	var nodes []Node
	for _, n := range c.Members {
		nodes = append(nodes, n)
	}
	cd := newCentroidDistance(c.Centroid, nodes)
	sort.Sort(cd)

	if n > len(cd.members) {
		return cd.members
	}
	return cd.members[:n]
}

// ClearMembers removes members of a cluster to allow storing a cluster efficiently.
func (c *Cluster) ClearMembers() {
	c.Members = nil
}
