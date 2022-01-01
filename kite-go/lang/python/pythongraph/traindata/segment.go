package traindata

// SegmentedIndicesFeed ...
type SegmentedIndicesFeed struct {
	SampleIDs []int32 `json:"sample_ids"`
	Indices   []int32 `json:"indices"`
}

// Append ss to the feed and return a new feed
func (s SegmentedIndicesFeed) Append(ss SegmentedIndicesFeed, sampleIDOffset, indicesOffset int32) SegmentedIndicesFeed {
	for i := 0; i < len(ss.SampleIDs); i++ {
		s.SampleIDs = append(s.SampleIDs, ss.SampleIDs[i]+sampleIDOffset)
		s.Indices = append(s.Indices, ss.Indices[i]+indicesOffset)
	}
	return s
}

// FeedDict ...
func (s SegmentedIndicesFeed) FeedDict(prefix string) map[string]interface{} {
	return map[string]interface{}{
		prefix + "_sample_ids": s.SampleIDs,
		prefix + "_indices":    s.Indices,
	}
}

// NewSegmentedIndicesFeed ...
func NewSegmentedIndicesFeed(idxs ...int32) SegmentedIndicesFeed {
	if len(idxs) == 0 {
		return SegmentedIndicesFeed{
			SampleIDs: []int32{},
			Indices:   []int32{},
		}
	}
	return SegmentedIndicesFeed{
		SampleIDs: make([]int32, len(idxs)),
		Indices:   idxs,
	}
}
