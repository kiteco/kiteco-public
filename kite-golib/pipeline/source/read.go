package source

import "github.com/kiteco/kiteco/kite-golib/pipeline"

// ReadAll is a convenience function that will read all entries
// from a newly initialized source.
// NOTE:
//  - this is meant as a convenience function for when clients want to
//    use a source outside of the context of a pipeline.
//  - this should NOT be used in conjunction with a pipeline.
//  - clients should NOT call s.ForShard before calling this.
func ReadAll(s pipeline.Source) ([]pipeline.Record, error) {
	ss, err := s.ForShard(0, 1)
	if err != nil {
		return nil, err
	}
	var recs []pipeline.Record
	for r := ss.SourceOut(); (r != pipeline.Record{}); r = ss.SourceOut() {
		recs = append(recs, r)
	}
	return recs, nil
}
