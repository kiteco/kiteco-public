package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/conversion/monetizable"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
)

// Result type
type Result struct {
	Score        float64 `json:"score"`
	Timestamp    int64   `json:"timestamp"`
	Userid       string  `json:"userid"`
	ModelVersion string  `json:"model_version"`
}

// Inputs alias
type Inputs = monetizable.Inputs

// Input type
type Input struct {
	Userid string `json:"userid"`
	Inputs
}

func main() {
	var dataPath string
	flag.StringVar(&dataPath, "data", "", "path to data directory")

	var region string
	flag.StringVar(&region, "region", "us-east-1", "AWS region of source data path")

	var destPath string
	flag.StringVar(&destPath, "dest", "", "path to destination directory")

	flag.Parse()

	buildHash := os.Getenv("BUILD_HASH")
	runTS := time.Now().Unix()

	dataURL, err := url.Parse(dataPath)
	if err != nil {
		log.Fatalf("Error parsing data path, %v", err)
	}

	keys, err := awsutil.S3ListObjects(region, dataURL.Hostname(), strings.TrimPrefix(dataURL.Path, "/"))
	if err != nil {
		log.Fatalf("Error listing data directory, %v", err)
	}
	for _, key := range keys {
		srcFilename := fmt.Sprintf("s3://%s", path.Join(dataURL.Hostname(), key))
		dstFilename := fmt.Sprintf("%s/%s", destPath, fmt.Sprintf("%s.json", strings.TrimSuffix(path.Base(key), ".gz")))
		log.Printf("Processing file %s, destination=%s", srcFilename, dstFilename)
		handleFile(srcFilename, dstFilename, buildHash, runTS)
	}
}

func handleFile(srcFilename string, dstFilename string, buildHash string, runTS int64) {
	zReader, err := fileutil.NewReader(srcFilename)
	if err != nil {
		log.Fatalf("Error reading data file %s", srcFilename)
	}
	defer zReader.Close()
	gr, err := gzip.NewReader(zReader)
	if err != nil {
		log.Fatalf("Error reading gzip data in file %s", srcFilename)
	}
	defer gr.Close()

	outf, err := fileutil.NewBufferedWriter(dstFilename)
	if err != nil {
		log.Fatalf("Error opening file %s for writing, %v", dstFilename, err)
	}
	defer outf.Close()
	writer := json.NewEncoder(outf)

	scanner := bufio.NewScanner(gr)

	for scanner.Scan() {
		var input Input
		err = json.Unmarshal(scanner.Bytes(), &input)
		if err != nil {
			log.Fatalf("Error parsing JSON from %s, %v", srcFilename, err)
		}
		score, err := monetizable.Score(input.Inputs)
		if err != nil {
			log.Fatalf("Error computing score, %v", err)
		}

		var result = Result{
			Score:        score,
			Timestamp:    runTS,
			Userid:       input.Userid,
			ModelVersion: buildHash,
		}
		if err := writer.Encode(result); err != nil {
			log.Fatalf("Error writing result data file %s, %v", dstFilename, err)
		}
	}
}
