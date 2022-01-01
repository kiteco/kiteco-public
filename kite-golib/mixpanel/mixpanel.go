package mixpanel

// User reflects the json returned by the mixpanel JQL when searching for people
type User struct {
	//ignore time and last_seen as not useful currently and time format is nonstandard
	DistinctID string                 `json:"distinct_id"`
	Labels     []string               `json:"labels"`
	Properties map[string]interface{} `json:"properties"`
}
