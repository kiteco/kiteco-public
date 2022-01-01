package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
)

//func uploadLogs(client *http.Client, auth *token.Token, usernode *url.URL, logDir string, machineID string) {
func uploadLogs(client component.AuthClient, logDir, machineID, installID string) error {
	// catch any panics
	defer func() {
		if ex := recover(); ex != nil {
			log.Printf("panic uploading client logs: %v", ex)
		}
	}()

	start := time.Now()

	var logs []string
	err := filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, PreviousLogsSuffix) {
			return nil
		}
		logs = append(logs, path)
		return nil
	})
	if err != nil {
		log.Printf("error uploading logs: %v\n", err)
		return err
	}

	numUploaded := 0
	for _, path := range logs {
		raw, err := ioutil.ReadFile(path)
		if err != nil {
			log.Printf("error uploading logs: error reading %s: %v", path, err)
			continue
		}

		// this is transmitted as a raw string (gzipped) so that we can
		// open the resulting log files directly in the web browser for S3
		var buf bytes.Buffer
		gzw := gzip.NewWriter(&buf)
		if _, err := gzw.Write(raw); err != nil {
			log.Printf("error uploading logs: error encoding %s: %v", path, err)
			continue
		}
		if err := gzw.Close(); err != nil {
			log.Printf("error uploading logs: error closing gz for %s: %v", path, err)
			continue
		}

		vals := url.Values{}
		vals.Set("filename", filepath.Base(path))
		vals.Set("machineid", machineID)
		vals.Set("installid", installID)
		vals.Set("platform", runtime.GOOS)
		ep, err := client.Parse("/clientlogs" + "?" + vals.Encode())
		if err != nil {
			log.Printf("error uploading logs: error parsing URL /clientlogs?%s: %v", vals.Encode(), err)
			continue
		}

		req, err := client.NewRequest("POST", ep.String(), "text/plain", &buf)
		if err != nil {
			log.Printf("error uploading logs: error creating request %s: %v", ep.String(), err)
			continue
		}

		req.Header.Set("Content-Encoding", "gzip")

		ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
		defer cancel()

		resp, err := client.Do(ctx, req)
		if err != nil {
			log.Printf("error uploading logs: error posting logs to %s: %v", ep.String(), err)
			continue
		}
		if resp.StatusCode != 200 {
			_ = resp.Body.Close()
			msg := fmt.Sprintf("error uploading logs: error posting logs %s: %s", ep.String(), resp.Status)
			log.Printf(msg)
			continue
		}
		if err := resp.Body.Close(); err != nil {
			log.Printf("error uploading logs: error closing response body %s: %v", ep.String(), err)
			continue
		}

		if err := os.Remove(path); err != nil {
			log.Printf("error uploading logs: error removing file %s: %v", path, err)
			continue
		}
		numUploaded++
	}

	log.Printf("uploaded %d logs in %v\n", numUploaded, time.Since(start))
	return nil
}
