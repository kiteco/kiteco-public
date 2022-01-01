package clustering

import (
	"fmt"
	"log"

	"github.com/biogo/cluster/kmeans"
)

// Algorithm is an interface for different types of clustering algorithms.
type Algorithm interface {
	Train(data []Node) []*Cluster
}

// KMeans implements Algorithm using the standard k-means clustering algorithm.
// K: # of clusters to be found
type KMeans struct {
	K int
}

// NewKMeans returns a pointer to a new KMeans struct.
func NewKMeans(k int) (*KMeans, error) {
	if k <= 0 {
		return nil, fmt.Errorf("k must be larger than 0 to instantiate a KMeans struct")
	}
	return &KMeans{
		K: k,
	}, nil
}

// Train is KMeans' implementation for Algorithm.
func (km *KMeans) Train(data nodes) []*Cluster {
	trainer, err := kmeans.New(data)
	if err != nil {
		log.Printf("error instantiating kmeans object: %s", err.Error())
		return nil
	}
	trainer.Seed(km.K)
	trainer.Cluster()

	N := float64(data.Len())
	var clusters []*Cluster

	for _, c := range trainer.Centers() {
		if len(c.Members()) == 0 {
			continue
		}
		members := make([]*BasicNode, len(c.Members()))
		for m, index := range c.Members() {
			members[m] = &BasicNode{
				FeatVec:  data[index].Values(),
				UniqueID: data[index].ID(),
			}
		}
		centroid := &BasicNode{
			FeatVec: c.V(),
		}
		cluster := &Cluster{
			Centroid: centroid,
			Members:  members,
			Weight:   float64(len(members)) / N,
			ID:       uint64(len(clusters)),
		}
		clusters = append(clusters, cluster)
	}
	return clusters
}
