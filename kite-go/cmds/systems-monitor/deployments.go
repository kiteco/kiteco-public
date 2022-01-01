package main

import (
	"fmt"
	"log"

	"github.com/kiteco/kiteco/kite-golib/status"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

// deployment nodes such as user nodes, user muxes
type deployment struct {
	Region string // Region is the region name of the node
	Type   string // Type is the node type
	IP     string // IP is the node's IP
	Num    string // Num is an instance label for metric naming

	// data sources
	debug  debugVars
	sys    sysMetrics
	status status.Status
}

func (d *deployment) name() []string {
	return []string{d.Region, d.Type, d.Num}
}

// fetchSources fetches all sources for a deployment node in parallel
func (d *deployment) fetchSources() error {
	sourceFuncs := []workerpool.Job{
		func() error { return fetchMemUsage(d.IP, &d.sys) },
		func() error { return fetchDebug(d.IP, &d.debug) },
		func() error { return fetchStatus(d.IP, &d.status) },
	}

	// run source functions with workerpool
	pool := workerpool.New(len(sourceFuncs))
	defer pool.Stop()
	pool.Add(sourceFuncs)

	if err := pool.Wait(); err != nil {
		return fmt.Errorf("error(s) in worker pool while fetching sources for %s: %v", d.IP, err)
	}

	return nil

}

// define and return all metrics
func (d *deployment) metrics() []*Metric {
	mets := []*Metric{
		// system memory usage
		&Metric{
			Name:  metricName(d.name(), "sys", "mem", "usage"),
			Value: d.sys.memUsage * 100.0,
			Unit:  units.percent,
		},
		// user-node heap allocation
		&Metric{
			Name:  metricName(d.name(), "proc", "alloc"),
			Value: float64(d.debug.MemStats.Alloc) / 1000000000.0, // convert to gigabytes float
			Unit:  units.gigabytes,
		},
		// user-node heap objects
		&Metric{
			Name:  metricName(d.name(), "proc", "heap-obj"),
			Value: float64(d.debug.MemStats.HeapObjects),
			Unit:  units.count,
		},
		// user-node gc pause
		&Metric{
			Name:  metricName(d.name(), "proc", "gc-pause-total"),
			Value: float64(d.debug.MemStats.PauseTotalNs) / 1000000.0, // convert to milliseconds
			Unit:  units.milliseconds,
		},
	}
	// add status metrics
	mets = append(mets, d.statusMetrics()...)

	return mets
}

// converts a deployment's Status to Metrics
func (d *deployment) statusMetrics() []*Metric {
	var mets []*Metric

	for _, section := range d.status.Sections {
		// skip headlines section because they are duplicates
		// NOTE: there is a space in front of Headlines - see status/status.go for why
		if section.Name == " Headlines" {
			continue
		}

		for key, counter := range section.Counters {
			if counter.Headline || counter.Timeseries {
				mets = append(mets, &Metric{
					Name:  metricName(d.name(), "status", instName(key)),
					Value: float64(counter.Value),
					Unit:  units.count,
				})
			}
		}

		for key, ratio := range section.Ratios {
			if ratio.Headline || ratio.Timeseries {
				mets = append(mets, &Metric{
					Name:  metricName(d.name(), "status", instName(key)),
					Value: float64(ratio.Value()),
					Unit:  units.percent,
				})
			}
		}

		for key, breakdown := range section.Breakdowns {
			for cat, val := range breakdown.Value() {
				if breakdown.Headline || breakdown.Timeseries {
					mets = append(mets, &Metric{
						Name:  metricName(d.name(), "status", instName(key), instName(cat)),
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
						Name:  metricName(d.name(), "status", instName(key), sampleName(i)),
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
						Name:  metricName(d.name(), "status", instName(key), sampleName(i)),
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
						Name:  metricName(d.name(), "status", instName(key), sampleName(i)),
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

// helper to return a name for a sample percentile
func sampleName(i int) string {
	// refer to status.samplePercentiles for the values and ordering
	switch i {
	case 0:
		return "25th"
	case 1:
		return "50th"
	case 2:
		return "75th"
	case 3:
		return "95th"
	case 4:
		return "99th"
	default:
		log.Fatalf("invalid sample percentile")
	}
	return ""
}
