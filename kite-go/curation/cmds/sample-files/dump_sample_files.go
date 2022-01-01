package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/kiteco/kiteco/kite-go/annotate"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/serialization"
)

func main() {
	var output string
	flag.StringVar(&output, "output", "", "path to which to write output")
	flag.Parse()

	w, err := serialization.NewEncoder(output)
	if err != nil {
		log.Fatalln(err)
	}
	defer w.Close()

	sampleFiles, err := fetchSampleFiles()
	if err != nil {
		log.Println(err)
	}

	err = w.Encode(sampleFiles)
	if err != nil {
		log.Fatalln(err)
	}
}

func fetchSampleFiles() (map[string][]byte, error) {
	s3url, err := awsutil.ValidateURI(annotate.SamplesDir)
	if err != nil {
		return nil, err
	}
	bucket, err := awsutil.GetBucket(s3url.Host)
	if err != nil {
		return nil, err
	}
	key := strings.TrimPrefix(s3url.Path, "/")

	resp, err := bucket.List(key, "", "", 1000)
	if err != nil {
		return nil, fmt.Errorf("error listing bucket: %+v", err)
	}
	sampleFiles := make(map[string][]byte)
	for _, k := range resp.Contents {
		data, err := bucket.Get(k.Key)
		if err != nil {
			return nil, fmt.Errorf("error getting key %s from bucket: %+v", k.Key, err)
		}
		k.Key = strings.TrimPrefix(k.Key, annotate.SamplesBase+"/")
		sampleFiles[k.Key] = data
	}
	return sampleFiles, nil
}
