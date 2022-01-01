package tracks

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Listing returns segment event listings from S3
type Listing struct {
	Days []DayAndKeys
}

// DayAndKeys represents all the S3 keys containing segment events for a day
type DayAndKeys struct {
	Day  time.Time
	Keys []string
}

// List will return a Listing for the provided source in the provided bucket
func List(bucket, source string) (*Listing, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	var svc *s3.S3
	if bucket == MetricsBucket {
		svc = s3.New(sess, aws.NewConfig().WithRegion("us-east-1"))
	} else {
		svc = s3.New(sess, defaults.Get().Config)
	}

	listReq := &s3.ListObjectsInput{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(fmt.Sprintf("segment-logs/%s/", source)),
		Delimiter: aws.String("/"),
	}

	listResp, err := svc.ListObjects(listReq)
	if err != nil {
		return nil, err
	}

	listing := &Listing{}
	for _, prefix := range listResp.CommonPrefixes {
		ts, err := timeFromSegmentPrefix(*prefix.Prefix)
		if err != nil {
			return nil, err
		}

		keys := DayAndKeys{Day: ts}

		listReq := &s3.ListObjectsInput{
			Bucket:    aws.String(bucket),
			Prefix:    prefix.Prefix,
			Delimiter: aws.String("/"),
		}

		listResp, err := svc.ListObjects(listReq)
		if err != nil {
			return nil, err
		}

		for _, contents := range listResp.Contents {
			keys.Keys = append(keys.Keys, *contents.Key)
		}

		listing.Days = append(listing.Days, keys)
	}

	return listing, nil
}

// --

func timeFromSegmentPrefix(prefix string) (time.Time, error) {
	prefix = strings.TrimSuffix(prefix, "/")
	parts := strings.Split(prefix, "/")
	last := parts[len(parts)-1]

	ts, err := strconv.ParseInt(last, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(ts/1000, 0), nil
}
