package main

import (
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonmetrics"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/aggregator"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
)

const (
	allProjectKey = "all_projects_"
)

func newMapAggregator(name string, projectName string) pipeline.Aggregator {
	newFn := func() sample.Addable { return make(sample.StatsMap) }

	inFn := func(s pipeline.Sample) sample.Addable {
		ref := s.(pythonmetrics.ReferenceComparison)
		key := getKey(ref.OnlineFields)

		sm := make(sample.StatsMap, 2)
		sm[allProjectKey+key] = sample.Stats{
			Count:   1,
			Sum:     1,
			Average: 1,
		}
		sm[projectName+"_"+key] = sample.Stats{
			Count:   1,
			Sum:     1,
			Average: 1,
		}
		return sm
	}
	return aggregator.NewSumAggregator(name, newFn, inFn)
}

func getKey(ref pythonmetrics.ReferenceComparisonOnline) string {
	switch ref.IntelliJResolutionLevel {
	case pythonmetrics.Unknown:
		if ref.KiteResolutionLevel == 0 {
			return "BothUnknown"
		}
		return "KiteKnown/IntelliJUnknown"
	case pythonmetrics.DuckType:
		if ref.KiteResolutionLevel == 0 {
			return "KiteUnknown/IntelliJDuckType"
		}
		return "KiteKnown/IntelliJDuckType"
	case pythonmetrics.UnionType, pythonmetrics.Known:
		if ref.KiteResolutionLevel == 0 {
			return "KiteUnkown/IntelliJKnown"
		}
		return "BothKnown"
	}
	return "UnexpectedCase: IJ:" + ref.IntelliJResolutionLevel.String() + " Kite:" + ref.KiteResolutionLevel.String()
}
