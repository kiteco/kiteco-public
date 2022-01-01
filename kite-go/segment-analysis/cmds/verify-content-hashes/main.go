package main

import (
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/kiteco/kiteco/kite-go/segment-analysis/internal/tracks"
	"github.com/kiteco/kiteco/kite-golib/awsutil"
)

var (
	regions = map[string]string{
		"us-west-1": "kite-local-content",
		"us-east-1": "kite-local-content-us-east-1",
		"eu-west-1": "kite-local-content-eu-west-1",
	}
)

func main() {
	hashSet, err := tracks.LoadContentHashSet("s3://kite-data/localfiles/2017-06-05/contenthashes.gob.gz")
	if err != nil {
		log.Fatalln(err)
		return
	}

	readers := make(map[string]*s3.S3)
	for region := range regions {
		reader, err := awsutil.NewS3(region)
		if err != nil {
			log.Fatalln(err)
			return
		}
		readers[region] = reader
	}

	// Check that each hash exists in every region
	missing := make(map[string]bool)
	for hash := range hashSet {
		for region, reader := range readers {
			bucketName := regions[region]
			headObjInput := &s3.HeadObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(hash),
			}
			_, err = reader.HeadObject(headObjInput)
			if err != nil {
				if strings.Contains(err.Error(), "Not Found") {
					missing[hash] = true
					break
				}
			}
		}
	}

	log.Printf("Hashes in hash set not on s3: %d\n", len(missing))
}
