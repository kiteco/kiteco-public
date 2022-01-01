//go:generate go-bindata -pkg main -o bindata.go sample-files

package main

//TODO: refactor / abstract below

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/kiteco/kiteco/kite-go/response"
)

func main() {
	var charInputDelay int
	var numRequesters int
	var requestingPeriod string
	var requestSleep int
	var outDir string
	var fileSize string
	var port string
	var endpoint string
	var host string

	// a 125ms delay equates to about 96wpm
	flag.IntVar(&charInputDelay, "charInputDelay", 125, "ms delay between completions requests sent by a requester")
	flag.IntVar(&numRequesters, "numRequesters", 50, "number of requesters to simultaneously request")
	flag.StringVar(&requestingPeriod, "requestingPeriod", "30s", "number of seconds for each requester to request. Must be valid Duration string")
	flag.IntVar(&requestSleep, "requestSleep", 0, "number of ms for the endpoint to sleep before returning")
	flag.StringVar(&outDir, "dir", "", "the dir to output the results of the test to")
	flag.StringVar(&fileSize, "size", "medium", "the python file for the requesters to use. Legal values: 'small', 'medium', 'large'")
	flag.StringVar(&port, "port", ":7060", "the port to send a request to")
	flag.StringVar(&endpoint, "endpoint", "/api/sandbox-completions", "the completions endpoint to request")
	flag.StringVar(&host, "host", "http://localhost", "the host (without the port) to send the request to")
	flag.Parse()

	if outDir == "" {
		log.Println("You need to input a dir to which the output of these request tests can be put")
		return
	}

	if fileSize != "medium" && fileSize != "large" && fileSize != "small" {
		log.Printf("The input fileSize %s is not legal", fileSize)
		return
	}

	requestingDuration, err := time.ParseDuration(requestingPeriod)
	if err != nil {
		log.Printf("Invalid Duration string: %s", requestingPeriod)
		return
	}

	fileBytes, err := Asset("sample-files/" + fileSize + "-sample.py")
	if err != nil {
		log.Printf("err loading the sample file: %v", err)
		return
	}

	latencies := latencies{
		l: make([]responseTimings, 0),
	}

	completionsURL := host + port + endpoint

	var wg sync.WaitGroup
	wg.Add(numRequesters)
	ctx, cancel := context.WithTimeout(context.Background(), requestingDuration)
	defer cancel()
	for i := 0; i < numRequesters; i++ {
		go makeRequest(ctx, requestSleep, fileBytes, completionsURL, &latencies, &wg)
	}

	wg.Wait()

	//write latencies to file
	var logBuf bytes.Buffer

	//csv header
	logBuf.WriteString(fmt.Sprintf("DriverTime(ns),HandleEventTime(ns),CompletionsTime(ns),TotalCompletionsTime(ns),RoundTrip(ns),StatusCode,TextSizeSent,NumCompletionsReturned\n"))
	for _, latency := range latencies.l {
		logBuf.WriteString(fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d\n",
			latency.DriverTime,
			latency.HandleEventTime,
			latency.CompletionsTime,
			latency.TotalCompletionsTime,
			latency.RoundTrip,
			latency.StatusCode,
			latency.TextSizeSent,
			latency.NumCompletions,
		))
	}
	logBuf.WriteString(fmt.Sprintf("#######\n"))
	logBuf.WriteString(fmt.Sprintf("Input Delay (ms): %d\n", charInputDelay))
	logBuf.WriteString(fmt.Sprintf("Requesters: %d\n", numRequesters))
	logBuf.WriteString(fmt.Sprintf("Request Sleep: %d\n", requestSleep))
	logBuf.WriteString(fmt.Sprintf("Requesting Period (s): %s\n", requestingPeriod))
	curDate := time.Now().Format(time.RFC3339)
	logDir := path.Join(outDir, "completions-test-"+curDate+".txt")

	err = ioutil.WriteFile(logDir, logBuf.Bytes(), os.ModePerm)
	if err != nil {
		log.Printf("Error writing results file: %v\n", err)
	}
}

func makeRequest(ctx context.Context, sleep int, fileBytes []byte, url string, latencies *latencies, wg *sync.WaitGroup) {
	defer wg.Done()

	fileLength := len(fileBytes)
	byteIdx := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		payload := completionsRequest{
			Text:        string(fileBytes[:byteIdx]),
			CursorBytes: int64(byteIdx),
		}

		var b bytes.Buffer
		err := json.NewEncoder(&b).Encode(payload)
		if err != nil {
			log.Println("error with encoding:", err)
			continue
		}

		func() {
			start := time.Now()
			resp, err := http.Post(url, "application/json", &b)
			if err != nil {
				log.Println("error with request:", err)
				return
			}

			if resp.StatusCode != 200 {
				log.Println("non-200, non-error: ", resp.Status, resp.ContentLength)
			}

			defer resp.Body.Close()
			// read body, record into latencies with statusCode, textlength, timings, roundTrip
			var compResp completionsResponse
			if err = json.NewDecoder(resp.Body).Decode(&compResp); err != nil {
				log.Println("error with decoding response body: ", err)
				return
			}
			roundTrip := time.Since(start)

			latencies.m.Lock()
			latencies.l = append(latencies.l, responseTimings{
				DriverTime:           compResp.Timings.DriverTime,
				HandleEventTime:      compResp.Timings.HandleEventTime,
				CompletionsTime:      compResp.Timings.CompletionsTime,
				TotalCompletionsTime: compResp.Timings.TotalTime,
				RoundTrip:            roundTrip,
				StatusCode:           resp.StatusCode,
				TextSizeSent:         len(payload.Text),
				NumCompletions:       len(compResp.Completions),
			})
			latencies.m.Unlock()

			byteIdx++
			byteIdx = byteIdx % fileLength
		}()

		time.Sleep(time.Duration(sleep) * time.Millisecond)
	}
}

type completionTimings struct {
	DriverTime      time.Duration `json:"driver_time"`
	HandleEventTime time.Duration `json:"handle_event_time"`
	CompletionsTime time.Duration `json:"completions_time"`
	TotalTime       time.Duration `json:"total_time"`
}

type completionsResponse struct {
	Completions []response.SandboxCompletion `json:"completions"`
	Timings     *completionTimings           `json:"timings"`
}

type responseTimings struct {
	DriverTime           time.Duration
	HandleEventTime      time.Duration
	CompletionsTime      time.Duration
	TotalCompletionsTime time.Duration
	RoundTrip            time.Duration
	StatusCode           int
	TextSizeSent         int
	NumCompletions       int
}

type latencies struct {
	m sync.Mutex
	l []responseTimings
}

type completionsRequest struct {
	Text        string `json:"text"`
	CursorBytes int64  `json:"cursor_bytes"`
}
