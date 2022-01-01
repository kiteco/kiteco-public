package inspectorapi

// ExprListing describes an AST expression along with associated information.
type ExprListing struct {
	ExprType   string `json:"expr_type"`
	Begin      int64  `json:"begin"`
	End        int64  `json:"end"`
	ResolvesTo string `json:"resolves_to"`
}

// ExprDetail contains information about a specific expression encountered.
type ExprDetail struct {
	Name          string      `json:"name"`
	Cursor        int64       `json:"cursor"`
	ExprType      string      `json:"expr_type"`
	Begin         int64       `json:"begin"`
	End           int64       `json:"end"`
	ResolvedValue ValueDetail `json:"resolved_value"`
}

// ValueDetail describes information about a specific pythontype.Value.
type ValueDetail struct {
	Repr          string        `json:"repr"`
	Kind          string        `json:"kind"`
	Type          string        `json:"type"`
	Constituents  []ValueDetail `json:"constituents"` // if the value is a union, then this holds the constituents.
	Address       string        `json:"address"`
	GlobalType    string        `json:"global_type"`
	CanonicalName string        `json:"canonical_name"`
}
