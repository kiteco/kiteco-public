package response

// Documentation is the basic result structure returned for documentation results.
type Documentation struct {
	Type        string `json:"type"`
	Signature   string `json:"signature"`
	Description string `json:"description"`
}
