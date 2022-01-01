package tracks

import (
	"encoding/json"
	"io"
	"log"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
	analytics "gopkg.in/segmentio/analytics-go.v3"
)

// FilterFunc allows events to get filtered. Returning true indicates the
// event should be included in the stream.
type FilterFunc func(*analytics.Track) bool

// Reader reads multiple segment files from S3 in parallel
type Reader struct {
	Tracks chan *analytics.Track

	bucket string
	keys   []string
	pool   *workerpool.Pool
	filter FilterFunc

	start     time.Time
	completed int64
}

// NewReader returns a new reader
func NewReader(bucket string, keys []string, readers int) *Reader {
	return NewFilteredReader(bucket, keys, nil, readers)
}

// NewFilteredReader returns a new reader with a filter
func NewFilteredReader(bucket string, keys []string, filter FilterFunc, readers int) *Reader {
	return &Reader{
		Tracks: make(chan *analytics.Track, readers*1000),
		bucket: bucket,
		keys:   keys,
		pool:   workerpool.New(readers),
		filter: filter,
		start:  time.Now(),
	}
}

// StartAndWait starts the reader and waits until all reads are completed. It is expected
// that the client consumes from the Tracks channel after calling this method.
func (r *Reader) StartAndWait() error {
	var jobs []workerpool.Job
	for _, key := range r.keys {
		thisKey := key
		jobs = append(jobs, func() error { return r.reader(thisKey) })
	}

	r.pool.Add(jobs)
	err := r.pool.Wait()
	close(r.Tracks)
	r.pool.Stop()

	return err
}

// --

func (r *Reader) reader(key string) error {
	var count int
	defer func() {
		d := time.Since(r.start)
		c := int(atomic.LoadInt64(&r.completed))
		rem := len(r.keys) - c
		eta := time.Duration(int64(rem) * (int64(d) / int64(c)))
		if c%10 == 0 {
			log.Printf("%d out of %d in %s, ETA: %s (found %d events)", c, len(r.keys), d, eta, count)
		}
	}()

	defer atomic.AddInt64(&r.completed, 1)

	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	var svc *s3.S3
	if r.bucket == MetricsBucket {
		svc = s3.New(sess, aws.NewConfig().WithRegion("us-east-1"))
	} else {
		svc = s3.New(sess, defaults.Get().Config)
	}

	getReq := &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}

	getResp, err := svc.GetObject(getReq)
	if err != nil {
		log.Println(err)
		return err
	}

	defer getResp.Body.Close()

	decoder := json.NewDecoder(getResp.Body)
	for {
		var track analytics.Track
		err := decoder.Decode(&track)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Println(err)
			return err
		}

		if r.filter == nil || r.filter(&track) {
			r.Tracks <- &track
			count++
		}
	}
}
