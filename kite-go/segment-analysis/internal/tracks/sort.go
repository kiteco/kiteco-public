package tracks

import analytics "gopkg.in/segmentio/analytics-go.v3"

// ByTimestamp sorts track events chronologically
type ByTimestamp []*analytics.Track

func (b ByTimestamp) Len() int           { return len(b) }
func (b ByTimestamp) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByTimestamp) Less(i, j int) bool { return b[i].Timestamp.Before(b[j].Timestamp) }

// ByUserTimestamp sorts track events chronologically, grouped by user
type ByUserTimestamp []*analytics.Track

func (b ByUserTimestamp) Len() int      { return len(b) }
func (b ByUserTimestamp) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByUserTimestamp) Less(i, j int) bool {
	if b[i].UserId != b[j].UserId {
		return b[i].UserId < b[j].UserId
	}
	return b[i].Timestamp.Before(b[j].Timestamp)
}
