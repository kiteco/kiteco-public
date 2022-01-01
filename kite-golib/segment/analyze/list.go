package analyze

import (
	"encoding"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/kiteco/kiteco/kite-golib/segment/segmentsrc"
)

// - Dates -

// Date represents a specific date in UTC
type Date time.Time

// NewDate constructs a date from a year/month/day
func NewDate(year int, month time.Month, day int) Date {
	return Date(time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}

// ParseDate parses a date in the 2006-01-02 format
func ParseDate(s string) (Date, error) {
	var year, month, day int
	_, err := fmt.Sscanf(s, "%d-%d-%d", &year, &month, &day)
	if err != nil {
		return Date{}, err
	}
	return NewDate(year, time.Month(month), day), nil
}

// Today returns today's Date
func Today() Date {
	t := time.Now().UTC()
	return NewDate(t.Year(), t.Month(), t.Day())
}

// String implements fmt.Stringer
func (d Date) String() string {
	return time.Time(d).Format("2006-01-02")
}

// Set implements pflag.Value
func (d *Date) Set(s string) error {
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// Type implements pflag.Value
func (d *Date) Type() string {
	return "Date"
}

var _ = encoding.TextUnmarshaler((*Date)(nil))

// MarshalText implements encoding.TextMarshaler
func (d *Date) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (d *Date) UnmarshalText(text []byte) error {
	var err error
	*d, err = ParseDate(string(text))
	return err
}

// Between checks if d is between start and end, inclusive
func (d Date) Between(start Date, end Date) bool {
	return !time.Time(d).Before(time.Time(start)) && !time.Time(d).After(time.Time(end))
}

// Add adds the given years, months, days to Date
func (d Date) Add(years, months, days int) Date {
	return Date(time.Time(d).AddDate(years, months, days))
}

// - Listings -

// Listing represents a collection of S3 keys indexed by Date.
type Listing struct {
	Dates []DateAndURIs
}

// DateAndURIs represents a list of S3 URIs and a Date.
type DateAndURIs struct {
	Date Date
	URIs []string
}

// ListRange returns a listing given an S3 bucket, Segment source, and start/end dates
func ListRange(source segmentsrc.Source, start Date, end Date) (*Listing, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	svc := s3.New(sess, defaults.Get().Config.WithRegion(source.Region))

	listReq := &s3.ListObjectsInput{
		Bucket:    aws.String(source.Bucket),
		Prefix:    aws.String(fmt.Sprintf("segment-logs/%s/", source.SourceID)),
		Delimiter: aws.String("/"),
	}
	listResp, err := svc.ListObjects(listReq)
	if err != nil {
		return nil, err
	}

	var listing Listing
	for _, prefix := range listResp.CommonPrefixes {
		date, err := dateFromSegmentPrefix(*prefix.Prefix)
		if err != nil {
			return nil, err
		}
		if !date.Between(start, end) {
			continue
		}

		listReq := &s3.ListObjectsInput{
			Bucket:    aws.String(source.Bucket),
			Prefix:    prefix.Prefix,
			Delimiter: aws.String("/"),
		}
		listResp, err := svc.ListObjects(listReq)
		if err != nil {
			return nil, err
		}

		keys := DateAndURIs{Date: date}
		for _, contents := range listResp.Contents {
			keys.URIs = append(keys.URIs, fmt.Sprintf("s3://%s/%s", source.Bucket, *contents.Key))
		}
		listing.Dates = append(listing.Dates, keys)
	}

	sort.Slice(listing.Dates, func(i, j int) bool {
		return time.Time(listing.Dates[i].Date).Before(time.Time(listing.Dates[j].Date))
	})
	for _, d := range listing.Dates {
		sort.Strings(d.URIs)
	}

	return &listing, nil
}

// List returns a listing given an S3 bucket and Segment source.
func List(source segmentsrc.Source) (*Listing, error) {
	return ListRange(source, Date{}, Today())
}

// --

// When logging to S3, the directory structure Segment uses is:
// s3://<bucket name>/segment-logs/<source id>/<millsecond timestamp of received day>/<log filename>
// See https://segment.com/docs/destinations/amazon-s3/#data-format
func dateFromSegmentPrefix(prefix string) (Date, error) {
	prefix = strings.TrimSuffix(prefix, "/")
	parts := strings.Split(prefix, "/")
	last := parts[len(parts)-1]

	ts, err := strconv.ParseInt(last, 10, 64)
	if err != nil {
		return Date{}, err
	}

	t := time.Unix(ts/1000, 0).UTC()
	return NewDate(t.Year(), t.Month(), t.Day()), nil
}
