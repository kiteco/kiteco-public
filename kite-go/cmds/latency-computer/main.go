package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	driverTimeIdx      = 0
	handleEventTimeIdx = 1
	completionsTimeIdx = 2
	totalTimeIdx       = 3
	roundTripTimeIdx   = 4
)

func main() {
	var doStatusCodes bool
	var doTextSizes bool
	var doNumCompletions bool
	var logFileStr string
	var outDir string

	flag.BoolVar(&doStatusCodes, "status", false, "aggregate with respect to StatusCodes")
	flag.BoolVar(&doTextSizes, "text", false, "aggregate with respect to the text size sent in the request")
	flag.BoolVar(&doNumCompletions, "completions", false, "aggreate with respect to the number of completions returned")
	flag.StringVar(&logFileStr, "file", "", "the file to parse and interpret")
	flag.StringVar(&outDir, "outDir", "", "the directory to write to")
	flag.Parse()

	if logFileStr == "" {
		log.Println("You need to input a log file to parse")
		return
	}

	if outDir == "" {
		log.Println("You need to input a directory to write to")
		return
	}

	logFile, err := os.Open(logFileStr)
	if err != nil {
		log.Printf("Error opening log file: %v\n", err)
		return
	}
	defer logFile.Close()

	logScanner := bufio.NewScanner(logFile)

	// scan initial header
	logScanner.Scan()

	// make data slice
	var data [][]int64

	for logScanner.Scan() {
		// end of csv
		if logScanner.Text() == "#######" {
			break
		}
		var trial []int64
		for _, str := range strings.Split(logScanner.Text(), ",") {
			datum, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				log.Printf("formatting error in the data csv: %v\n", err)
				return
			}
			trial = append(trial, datum)
		}
		data = append(data, trial)
	}

	// now we have the data
	// open file
	var logBuf bytes.Buffer

	if doStatusCodes {
		statusCodeIdx := 5
		logBuf.WriteString("\n\n---------\nStatus Codes\n---------\n\n")
		aggregate(&logBuf, data, statusCodeIdx, 0)
	}

	if doTextSizes {
		textSizeIdx := 6
		logBuf.WriteString("\n\n---------\nText Length Sent\n----------\n\n")
		aggregate(&logBuf, data, textSizeIdx, 15)
	}

	if doNumCompletions {
		numCompletionsIdx := 7
		logBuf.WriteString("\n\n---------\nCompletions Returned\n----------\n\n")
		aggregate(&logBuf, data, numCompletionsIdx, 10)
	}

	logBuf.WriteString("\n\n###############\nOverall Measures\n################\n\n")
	writeStatisticalMeasures(&logBuf, data)

	// write independent trial variables
	logBuf.WriteString("\n\n***PARAMETERS****\n\n")
	for logScanner.Scan() {
		logBuf.WriteString(logScanner.Text() + "\n")
	}

	curDate := time.Now().Format(time.RFC3339)
	logDir := path.Join(outDir, "completions-measures-"+curDate+".txt")

	err = ioutil.WriteFile(logDir, logBuf.Bytes(), os.ModePerm)
	if err != nil {
		log.Printf("Error writing results file: %v\n", err)
	}
}

func aggregate(b *bytes.Buffer, data [][]int64, trialIdx int, numBuckets int) {
	// each bucket is a subset of data
	// to get number of buckets, will need to sort data along trialIdx
	sort.SliceStable(data, func(i, j int) bool {
		return data[i][trialIdx] < data[j][trialIdx]
	})
	// then: get min and max and distance => get bucket size
	numTrials := len(data)
	dist := data[numTrials-1][trialIdx] - data[0][trialIdx]
	bucketSize := int64(1)
	if numBuckets > 0 {
		bucketSize = dist / int64(numBuckets)
	}
	// then: distribute into buckets and write bucket results
	curBucket := int64(0)
	b.WriteString(fmt.Sprintf("Bucket %d\n---\n", curBucket))
	var bucketData [][]int64
	for _, trial := range data {
		// get the right bucket
		for int64(curBucket)+bucketSize <= trial[trialIdx] {
			curBucket += bucketSize
			if len(bucketData) > 0 {
				writeStatisticalMeasures(b, bucketData)
				bucketData = nil
			}
			b.WriteString(fmt.Sprintf("Bucket %d\n---\n", curBucket))
		}
		bucketData = append(bucketData, trial)
	}
	b.WriteString("\n=================\n")
}

func writeStatisticalMeasures(b *bytes.Buffer, data [][]int64) {
	// do for each time index: Size, mean, median, min, max, standard deviation
	size := len(data)
	b.WriteString(fmt.Sprintf("Size: %d\n", size))
	var driverTimes []int64
	var driverSum int64
	var handleEventTimes []int64
	var handlEventsSum int64
	var completionsTimes []int64
	var completionsSum int64
	var totalTimes []int64
	var totalSum int64
	var roundTripTimes []int64
	var roundTripSum int64

	for _, trial := range data {
		driverTimes = append(driverTimes, trial[driverTimeIdx])
		driverSum += trial[driverTimeIdx]
		handleEventTimes = append(handleEventTimes, trial[handleEventTimeIdx])
		handlEventsSum += trial[handleEventTimeIdx]
		completionsTimes = append(completionsTimes, trial[completionsTimeIdx])
		completionsSum += trial[completionsTimeIdx]
		totalTimes = append(totalTimes, trial[totalTimeIdx])
		totalSum += trial[totalTimeIdx]
		roundTripTimes = append(roundTripTimes, trial[roundTripTimeIdx])
		roundTripSum += trial[roundTripTimeIdx]
	}
	b.WriteString("\nDriver Time (ms)\n------------\n")
	writeMeasures(b, driverTimes, driverSum)

	b.WriteString("\nHandle Event Time (ms)\n------------\n")
	sort.SliceStable(handleEventTimes, func(i, j int) bool { return handleEventTimes[i] < handleEventTimes[j] })
	writeMeasures(b, handleEventTimes, handlEventsSum)

	b.WriteString("\nCompletions Time (ms)\n------------\n")
	sort.SliceStable(completionsTimes, func(i, j int) bool { return completionsTimes[i] < completionsTimes[j] })
	writeMeasures(b, completionsTimes, completionsSum)

	b.WriteString("\nTotal Time (ms)\n------------\n")
	sort.SliceStable(totalTimes, func(i, j int) bool { return totalTimes[i] < totalTimes[j] })
	writeMeasures(b, totalTimes, totalSum)

	b.WriteString("\nRound Trip Time (ms)\n------------\n")
	sort.SliceStable(roundTripTimes, func(i, j int) bool { return roundTripTimes[i] < roundTripTimes[j] })
	writeMeasures(b, roundTripTimes, roundTripSum)
}

func writeMeasures(b *bytes.Buffer, times []int64, timeSum int64) {
	sort.SliceStable(times, func(i, j int) bool { return times[i] < times[j] })
	mean := timeSum / int64(len(times))
	b.WriteString(fmt.Sprintf("Mean: %f\n", float64(mean)/1000000))
	b.WriteString(fmt.Sprintf("Min: %f\n", float64(times[0])/1000000))
	b.WriteString(fmt.Sprintf("Max: %f\n", float64(times[len(times)-1])/1000000))
	b.WriteString(fmt.Sprintf("Standard Deviation: %f\n", stdDev(times, mean)/1000000))
}

func stdDev(nums []int64, mean int64) float64 {
	var sd float64

	for i := 0; i < len(nums); i++ {
		sd = math.Pow(float64(nums[i]-mean), 2)
	}

	return math.Sqrt(sd / float64(len(nums)))
}
