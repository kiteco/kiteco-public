package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/status"
)

// debugVars and memstats are for unmarshalling the /debug/vars json
type debugVars struct {
	MemStats runtime.MemStats `json:"memstats"`
}

// sysMetrics contains system metrics
type sysMetrics struct {
	memUsage float64 // current system memory usage percentage
}

// ## Helper functions for fetching different sources, used by fetchSources methods

// fetchMemUsage fetches system memory info by calling free over ssh, calculates the usage percentage,
// and sets the value for the given pointer
func fetchMemUsage(ip string, dest *sysMetrics) error {
	out, err := exec.Command("ssh", "-oStrictHostKeyChecking=no", ip, "free").Output()
	if err != nil {
		return fmt.Errorf("could not get memory usage: %v", err)
	}
	total, used, _ := parseFree(string(out))
	if provider == "azure" {
		total, used, _ = parseFreeAzure(string(out))
	}
	usage := float64(used) / float64(total)

	dest.memUsage = usage

	return nil
}

// parseFree parses the stdout output of the unix free command
func parseFree(out string) (int, int, int) {
	// raw values are in second line
	vals := strings.Split(out, "\n")[1]
	// values with buffer considered are in third line
	valsWithBuf := strings.Split(out, "\n")[2]

	// split tokens
	var memVals []string
	for _, v := range strings.Split(vals, " ") {
		if v != "" {
			memVals = append(memVals, v)
		}
	}

	var memValsWithBuf []string
	for _, v := range strings.Split(valsWithBuf, " ") {
		if v != "" {
			memValsWithBuf = append(memValsWithBuf, v)
		}
	}
	// convert to int - total is second in raw vals, used and free are third and fourth in buf vals
	total, err := strconv.Atoi(memVals[1])
	if err != nil {
		log.Fatalln(err)
	}
	used, err := strconv.Atoi(memValsWithBuf[2])
	if err != nil {
		log.Fatalln(err)
	}
	free, err := strconv.Atoi(memValsWithBuf[3])
	if err != nil {
		log.Fatalln(err)
	}

	return total, used, free
}

// parseFreeAzure is for parsing the slightly different free output on azure
func parseFreeAzure(out string) (int, int, int) {
	// raw values are in second line
	vals := strings.Split(out, "\n")[1]

	// split tokens
	var memVals []string
	for _, v := range strings.Split(vals, " ") {
		if v != "" {
			memVals = append(memVals, v)
		}
	}

	// convert to int - total is second in raw vals, followed by used, free, shared (unused),
	// buff/cache (unused), and available (which is the "real" free space)
	total, err := strconv.Atoi(memVals[1])
	if err != nil {
		log.Fatalln(err)
	}
	used, err := strconv.Atoi(memVals[2])
	if err != nil {
		log.Fatalln(err)
	}
	available, err := strconv.Atoi(memVals[6])
	if err != nil {
		log.Fatalln(err)
	}

	return total, used, available
}

// fetchDebug fetches select values from the /debug/vars endpoint
func fetchDebug(ip string, dest *debugVars) error {
	u, err := url.Parse(fmt.Sprintf("http://%s:9091/debug/vars", ip))
	if err != nil {
		return fmt.Errorf("invalid URL for debug vars: %v", err)
	}
	res, err := http.Get(u.String())
	if err != nil {
		return fmt.Errorf("error getting debug vars: %v", err)
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(dest); err != nil {
		return fmt.Errorf("error parsing debug vars for %s: %v", ip, err)
	}

	return nil
}

// fetchStatus fetches status object from the /d
func fetchStatus(ip string, dest *status.Status) error {
	u, err := url.Parse(fmt.Sprintf("http://%s:9091/", ip))
	if err != nil {
		return fmt.Errorf("invalid URL for status: %v", err)
	}

	s, err := status.Poll(u)
	if err != nil {
		return fmt.Errorf("error polling status: %v", err)
	}

	dest.ShallowCopy(s)

	return nil
}

// aggregates a list of debugVar structs
//
// NOTE: we don't aggregate all fields, just the ones we care about
func aggregateDebug(debugs []*debugVars) debugVars {
	var out debugVars

	// lists to keep values in
	var allocs uint64
	var heapobjs uint64
	var gcpauses uint64
	for _, d := range debugs {
		// sum values
		allocs += d.MemStats.Alloc
		heapobjs += d.MemStats.HeapObjects
		gcpauses += d.MemStats.PauseTotalNs
	}

	// take and store average
	l := float64(len(debugs))
	out.MemStats.Alloc = round(float64(allocs) / l)
	out.MemStats.HeapObjects = round(float64(heapobjs) / l)
	out.MemStats.PauseTotalNs = round(float64(allocs) / l)

	return out
}

// aggregate sysMetrics
func aggregateSys(sys []*sysMetrics) sysMetrics {
	var out sysMetrics

	var memUsages float64
	for _, s := range sys {
		// sum values
		memUsages += s.memUsage
	}
	// take and store average
	out.memUsage = memUsages / float64(len(sys))

	return out
}

// quick helper for rounding to uint64
func round(x float64) uint64 {
	return uint64(math.Floor(x + 0.5))
}
