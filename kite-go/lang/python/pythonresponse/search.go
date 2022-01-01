package pythonresponse

// SearchResults contains the results of an active search request. It contains
// the results of querying the user's local codebase as well as the global
// import graph.
type SearchResults struct {
	LocalResults  interface{} `json:"local_results"`
	GlobalResults interface{} `json:"global_results"`
}
