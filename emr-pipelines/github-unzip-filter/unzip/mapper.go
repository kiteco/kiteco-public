package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	logPrefix = "[unzip-mapper] "
	logFlags  = log.LstdFlags | log.Lmicroseconds | log.Lshortfile
)

const (
	bucketPath   = "kite-github-crawl"
	maxRepoBytes = int64(8e7)  // 80 mb
	maxFileBytes = uint64(1e6) // 1 mb
)

func init() {
	log.SetPrefix(logPrefix)
	log.SetFlags(logFlags)
	log.SetOutput(os.Stderr)
}

// fetch repo from s3 bucket, unzip, and emit files.
func main() {
	start := time.Now()
	in := awsutil.NewEMRIterator(os.Stdin)
	out := awsutil.NewEMRWriter(os.Stdout)
	defer out.Close()

	// connect to s3 bucket
	auth, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		log.Fatalf("error getting authorization: %v", err)
	}

	conn := s3.New(auth, aws.USWest)
	bucket := conn.Bucket(bucketPath)

	var skippedRepos, unableToFetch int
	for in.Next() {
		nameBytes, err := base64.StdEncoding.DecodeString(in.Key())
		if err != nil {
			skippedRepos++
			log.Printf("error decoding name of repo with base64 encoded name `%s`: %v, skipping\n", in.Key(), err)
			continue
		}

		name := string(nameBytes)
		log.Printf("processing repo `%s`", name)

		pos := strings.Index(name, "/")
		if pos < 0 {
			log.Printf("\t has invalid path, skipping")
			skippedRepos++
			continue
		}
		org := name[:pos]

		// fetch repo head to check size
		head, err := bucket.Head(name, nil)
		switch {
		case err != nil:
			log.Printf("\t error fetching s3 head for %s: %v\n", name, err)
			unableToFetch++
			continue
		case head.ContentLength > maxRepoBytes:
			log.Printf("\t %s too large (%d B > %d B), skipping", name, head.ContentLength, maxRepoBytes)
			skippedRepos++
			continue
		}

		// fetch repo from s3
		repo, err := bucket.Get(name)
		if err != nil {
			log.Printf("\t error fetching contents from s3: %v\n", err)
			unableToFetch++
			continue
		}

		r := bytes.NewReader(repo)
		dr, err := zip.NewReader(r, int64(len(repo)))
		if err != nil {
			log.Printf("\t error unzipping: %v, skipping", err)
			skippedRepos++
			continue
		}

		var size uint64
		var skippedSz, skippedLang, nFiles, errOpen, errReading int64
		for _, file := range dr.File {
			if !file.Mode().IsRegular() {
				continue
			}

			nFiles++
			size += file.UncompressedSize64
			if file.UncompressedSize64 > maxFileBytes {
				skippedSz++
				continue
			}

			switch lang.FromFilename(file.Name) {
			case lang.Unknown:
				skippedLang++
				continue
			}

			f, err := file.Open()
			if err != nil {
				errOpen++
				continue
			}

			buf, err := ioutil.ReadAll(f)
			if err != nil {
				errReading++
				continue
			}

			// organzation/reponame/filepath
			key := org + "/" + file.Name
			// base 64 encode key
			key = base64.StdEncoding.EncodeToString([]byte(key))

			out.Emit(key, buf)
		}
		log.Printf("\t zipped %d b, unzipped %d b, %d files. Skipped: %d too large, %d unknown language, %d err opening, %d err reading\n",
			len(repo), size, nFiles, skippedSz, skippedLang, errOpen, errReading)
	}

	if err := in.Err(); err != nil {
		log.Fatalf("error reading from stdin: %v\n", err)
	}

	log.Printf("Done! took %v. Skipped %d repos, unable to fetch %d repos\n", time.Since(start), skippedRepos, unableToFetch)
}
