package main

import (
	"encoding/base64"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/dustin/go-humanize"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

// max is 1000
const batchSize = 1000

// Steps:
// 1) Get list of organizations (users) from GH dump, each directory represents an organization (user)
// 2) Get list of all repos for an organization (user)
// 3) Convert list of repos into an EMR with the repo name as the key (base64 encoded)
func main() {
	var args struct {
		Bucket   string `arg:"positional,required,help: path containing directories for each github organization"`
		ReposEMR string `arg:"positional,required,help:path to write EMR list of all repos"`
		Max      int    `arg:"help:maximum number of repos to list if < 1 then all repos will be added (default is 0)"`
	}
	arg.MustParse(&args)

	start := time.Now()

	if !strings.HasPrefix(args.Bucket, "s3://") {
		args.Bucket = "s3://" + args.Bucket
	}

	url, err := url.Parse(args.Bucket)
	if err != nil {
		log.Fatalf("error parsing url `%s`: %v", args.Bucket, err)
	}

	if url.Scheme != "s3" {
		log.Fatalf("url `%s` has scheme `%s` expected scheme `s3`", args.Bucket, url.Scheme)
	}

	auth, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		log.Fatalf("error getting authorization: %v", err)
	}

	conn := s3.New(auth, aws.USWest)
	bucket := conn.Bucket(url.Host)

	toAdd := "all"
	if args.Max > 0 {
		toAdd = humanize.Comma(int64(args.Max))
	}

	log.Printf("adding %s repos in bucket `%s` to emr list of repos\n", toAdd, args.Bucket)

	out, err := os.Create(args.ReposEMR)
	if err != nil {
		log.Fatalf("error opening output file `%s`: %v", args.ReposEMR, err)
	}
	defer out.Close()

	writer := awsutil.NewEMRWriter(out)
	defer writer.Close()

	var marker string
	var count int
	for {
		if count%1e5 == 0 && count > 0 {
			log.Printf("added %d repos to the list\n", count)
		}

		list, err := bucket.List("", "", marker, batchSize)
		if err != nil {
			log.Fatalf("error listing bucket contents: %v", err)
		}

		var done bool
		for _, key := range list.Contents {
			encoded := base64.StdEncoding.EncodeToString([]byte(key.Key))
			writer.Emit(encoded, nil)
			count++
			if args.Max > 0 && count >= args.Max {
				done = true
				break
			}
		}

		if len(list.Contents) == 0 || done {
			break
		}
		marker = list.Contents[len(list.Contents)-1].Key
	}

	log.Printf("Done! Took %v to add %d repos to emr list\n", time.Since(start), count)
}
