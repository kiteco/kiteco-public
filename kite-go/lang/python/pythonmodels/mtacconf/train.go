package mtacconf

// TrainSample represents training data that will pass to a python model training program.
type TrainSample struct {
	Features Features `json:"features"`
	// Label is the index of the call completion that was ultimately chosen, -1 if none was.
	Label int             `json:"label"`
	Meta  TrainSampleMeta `json:"meta"`
}

// TrainSampleMeta contains information about the sample that is not used directly for training or inference,
// but is useful for debugging and/or visualization.
type TrainSampleMeta struct {
	Hash            string   `json:"hash"`
	Cursor          int64    `json:"cursor"`
	CompIdentifiers []string `json:"comp_identifiers"`
}
