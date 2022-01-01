package main

import "github.com/kiteco/kiteco/kite-golib/status"

// deploymentGroup is a node that represents a group of deployments, typically in a region
type deploymentGroup struct {
	Region      string        // name of region
	deployments []*deployment // array of deployments for this region

	// data sources
	debug  debugVars
	sys    sysMetrics
	status status.Status
}

func (g *deploymentGroup) name() []string {
	return []string{g.Region, "aggregate"}
}

// fetchSources aggregates the data sources across its deployments
func (g *deploymentGroup) fetchSources() error {
	var statuses []*status.Status
	var debugs []*debugVars
	var sys []*sysMetrics

	for _, d := range g.deployments {
		statuses = append(statuses, &d.status)
		debugs = append(debugs, &d.debug)
		sys = append(sys, &d.sys)
	}
	// aggregate
	g.status = *status.Aggregate(statuses)
	g.debug = aggregateDebug(debugs)
	g.sys = aggregateSys(sys)

	return nil
}

func (g *deploymentGroup) metrics() []*Metric {
	mets := []*Metric{
		// system memory usage
		&Metric{
			Name:  metricName(g.name(), "sys", "mem", "usage"),
			Value: g.sys.memUsage * 100.0,
			Unit:  units.percent,
		},
		// user-node heap allocation
		&Metric{
			Name:  metricName(g.name(), "proc", "alloc"),
			Value: float64(g.debug.MemStats.Alloc) / 1000000000.0, // convert to gigabytes float
			Unit:  units.gigabytes,
		},
		// user-node heap objects
		&Metric{
			Name:  metricName(g.name(), "proc", "heap-obj"),
			Value: float64(g.debug.MemStats.HeapObjects),
			Unit:  units.count,
		},
		// user-node gc pause
		&Metric{
			Name:  metricName(g.name(), "proc", "gc-pause-total"),
			Value: float64(g.debug.MemStats.PauseTotalNs) / 1000000.0, // convert to milliseconds
			Unit:  units.milliseconds,
		},
	}
	// add status metrics
	mets = append(mets, g.statusMetrics()...)

	return mets
}

// converts a deployment's Status to Metrics
func (g *deploymentGroup) statusMetrics() []*Metric {
	var mets []*Metric

	for _, section := range g.status.Sections {
		// skip headlines section because they are duplicates
		// NOTE: there is a space in front of Headlines - see status/status.go for why
		if section.Name == " Headlines" {
			continue
		}

		for key, counter := range section.Counters {
			if counter.Headline || counter.Timeseries {
				mets = append(mets, &Metric{
					Name:  metricName(g.name(), "status", instName(key)),
					Value: float64(counter.Value),
					Unit:  units.count,
				})
			}
		}

		for key, ratio := range section.Ratios {
			if ratio.Headline || ratio.Timeseries {
				mets = append(mets, &Metric{
					Name:  metricName(g.name(), "status", instName(key)),
					Value: float64(ratio.Value()),
					Unit:  units.percent,
				})
			}
		}

		for key, breakdown := range section.Breakdowns {
			for cat, val := range breakdown.Value() {
				if breakdown.Headline || breakdown.Timeseries {
					mets = append(mets, &Metric{
						Name:  metricName(g.name(), "status", instName(key), instName(cat)),
						Value: float64(val),
						Unit:  units.percent,
					})
				}
			}
		}

		for key, sample := range section.SampleInt64s {
			for i, val := range sample.Values() {
				if sample.Headline || sample.Timeseries {
					mets = append(mets, &Metric{
						Name:  metricName(g.name(), "status", instName(key), sampleName(i)),
						Value: float64(val),
						Unit:  units.count,
					})
				}
			}
		}

		for key, sample := range section.SampleBytes {
			for i, val := range sample.Values() {
				if sample.Headline || sample.Timeseries {
					mets = append(mets, &Metric{
						Name:  metricName(g.name(), "status", instName(key), sampleName(i)),
						Value: float64(val) / 1000.0, // convert to kilobytes
						Unit:  units.kilobytes,
					})
				}
			}
		}

		for key, sample := range section.SampleDurations {
			for i, val := range sample.Values() {
				if sample.Headline || sample.Timeseries {
					mets = append(mets, &Metric{
						Name:  metricName(g.name(), "status", instName(key), sampleName(i)),
						Value: float64(val) / 1000000.0, // convert to milliseconds
						Unit:  units.milliseconds,
					})
				}
			}
		}

		//for key, counter := range section.CounterDistributions {
		//}

		//for key, counter := range section.RatioDistributions {
		//}

		//for key, counter := range section.BoolDistributions {
		//}

		//for key, counter := range section.DurationDistributions {
		//}
	}

	return mets
}
