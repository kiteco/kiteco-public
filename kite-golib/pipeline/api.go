package pipeline

// StartRequest is a request to start a pipeline run
type StartRequest struct {
	Shard               int
	TotalShards         int
	CoordinatorEndpoint string
}

// StatusResponse returns the status of a shard
type StatusResponse struct {
	State RunState
	Err   string
}

// ResultsResponse contains the final results of a shard
type ResultsResponse struct {
	// SerializedResults contains the JSON-serialized results for each aggregator
	SerializedResults map[string][]byte
}
