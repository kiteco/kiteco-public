package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	bucket              = "kite-web-snapshots"
	defaultPollInterval = time.Hour * 4
)

// makeKey creates a key for a snapshot of a URL using the URL and current timestamp
func makeKey(keyURL string) string {
	// strip http and https from url, strip trailing /, and replace / with :
	keyURL = strings.TrimPrefix(keyURL, "http://")
	keyURL = strings.TrimPrefix(keyURL, "https://")
	keyURL = strings.TrimRight(keyURL, "/")
	keyURL = strings.Replace(keyURL, "/", ":", -1)

	// get formatted current time
	currentTime := time.Now().Format(time.RFC3339)
	return fmt.Sprintf("%s/%s.html", keyURL, currentTime)
}

// snapshotToS3 will, given a list of URLs, make a GET request and save the response body as a text
// file and upload it to S3 with a url+timestamp key
func snapshotToS3(urls []string) {
	// initialize aws service
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := s3.New(sess)

	for _, u := range urls {
		// GET the url
		resp, err := http.Get(u)
		if err != nil {
			log.Println(err)
			continue
		}
		defer resp.Body.Close()

		// create object to put
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			continue
		}
		key := makeKey(u)
		reader := bytes.NewReader(body)
		params := &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   reader,
		}

		// upload
		_, err = svc.PutObject(params)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("created snapshot for %s at %s\n", u, key)
	}
}

func main() {
	// get args from command line
	var pollInterval time.Duration
	var urls string
	flag.DurationVar(&pollInterval, "interval", defaultPollInterval, "polling interval")
	flag.StringVar(&urls, "urls", "", "list of URLs")

	flag.Parse()

	// validate URLs
	urlList := strings.Split(urls, ",")
	for _, u := range urlList {
		_, err := url.ParseRequestURI(u)
		if err != nil {
			log.Fatalf("invalid URL %s", u)
		}
	}

	// run the snapshot once per interval
	for range time.NewTicker(pollInterval).C {
		snapshotToS3(urlList)
	}
}
