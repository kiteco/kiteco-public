package callprob

// TrainSample represents training data that will pass to a python model training program.
type TrainSample struct {
	Features Features `json:"features"`
	// Label is the index of the call completion that was ultimately chosen, -1 if none was.
	Labels []int           `json:"labels"`
	Meta   TrainSampleMeta `json:"meta"`
}

// TrainSampleMeta contains information about the sample that is not used directly for training or inference,
// but is useful for debugging and/or visualization.
type TrainSampleMeta struct {
	Hash            string   `json:"hash"`
	Cursor          int64    `json:"cursor"`
	CompIdentifiers []string `json:"comp_identifiers"`
}

// FlatFeatures represents features for one training sample for filtering model
type FlatFeatures struct {
	Contextual ContextualFeatures `json:"contextual"`
	Comp       CompFeatures       `json:"comp"`
}

// FlatTrainSample represents one training samples with features, label and some additional metadata
type FlatTrainSample struct {
	Features FlatFeatures        `json:"features"`
	Label    bool                `json:"label"`
	Meta     FlatTrainSampleMeta `json:"meta"`
}

// FlatTrainSampleMeta contains data to help debugging training of the filtering model
type FlatTrainSampleMeta struct {
	Hash           string `json:"hash"`
	Cursor         int64  `json:"cursor"`
	CompIdentifier string `json:"comp_identifier"`
}
