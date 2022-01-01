package pythondocs

import "github.com/kiteco/kiteco/kite-golib/status"

var (
	section = status.NewSection("lang/python/pythondocs")

	docsCacheRatio       = section.Ratio("LRU hit ratio")
	docsDiskmapRatio     = section.Ratio("Diskmap hit ratio")
	docsDiskmapMatchRate = section.Ratio("Diskmap/in-memory match rate")
	docsIndexRatio       = section.Ratio("In-memory hit ratio")

	docsDiskmapDuration = section.SampleDuration("Diskmap duration")
	docsEntityDuration  = section.SampleDuration("Corpus.Entity")
)
