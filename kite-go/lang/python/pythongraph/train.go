package pythongraph

import "math/rand"

// TrainConfig bundles config for training
type TrainConfig struct {
	MaxHops      int
	Graph        GraphFeedConfig
	NumCorrupted int
}

// TrainParams bundles params for training
type TrainParams struct {
	ModelMeta
	Rand  *rand.Rand
	Saver Saver
}
