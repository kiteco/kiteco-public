package main

// Metric is a single metric
type Metric struct {
	Name  []string // list of string that identify the metric; should only use alphanumeric and -_
	Value float64  // value of the metric
	Unit  string   // unit of the metric
}

// unitNames is a struct to store strings for metric units
type unitNames struct {
	count        string
	percent      string
	kilobytes    string
	megabytes    string
	gigabytes    string
	seconds      string
	milliseconds string
}

// use this for unit names
var units = unitNames{
	count:        "count",
	percent:      "p",
	kilobytes:    "KB",
	megabytes:    "MB",
	gigabytes:    "GB",
	seconds:      "s",
	milliseconds: "ms",
}

// helper function for generating metric names, mostly to modularize the logic
//
// note that the first argument is expected to be the output of either Node.name() or oneSlice()
func metricName(concattedSlice []string, sections ...string) []string {
	return append(concattedSlice, sections...)
}

// concat multiple string slices
func oneSlice(slices ...[]string) []string {
	var s []string
	for _, i := range slices {
		s = append(s, i...)
	}
	return s
}

// getMetrics calls metrics() on all nodes and compiles a list of Metrics
func getMetrics(nodes []Node) []*Metric {
	var metrics []*Metric

	for _, n := range nodes {
		metrics = append(metrics, n.metrics()...)
	}

	return metrics
}
