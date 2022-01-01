package ranking

// Entry is the structure used to serialize features together
// with the query id and the ranking label of the code example.
type Entry struct {
	SnapshotID int64     `json:"snapshot_id"`
	QueryHash  string    `json:"query_id"`
	QueryText  string    `json:"query_text"`
	QueryCode  string    `json:"query_code"`
	Label      float64   `json:"label"`
	Features   []float64 `json:"features"`
}
