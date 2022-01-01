package source

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"sync/atomic"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

// RecordStopError bundles
// - a record together
// - a stop flag indicating if the dataset should stop processing records
// - an err to log
// Only one of the above fields should have a non zero value.
type RecordStopError struct {
	Record pipeline.Record
	Stop   bool
	Err    error
}

// ProcessFn processes the contents of a reader and sends the results along the provided channel `res`,
// this is used in conjunction with an `S3Dataset` to process arbitrary files on s3.
// NOTE:
//   - the first argument is the key for the file on s3
//   - the second argument is a reader for the contents of the file
//   - this function may be called from multiple go routines simultaneously so it must be
//     go routine safe
//   - the dataset stops processing records from a particular file if either:
//     - the `res` is closed by the ProcessFn
//     - a `RecordStopError` result with the Stop flag set to true, is sent along the channel,
//       also note that once a result is processed with Stop == True then
//       the dataset stops reading results from `res`
//     - if the `Stop` flag is true then the dataset stops processing new files and returns.
type ProcessFn func(name string, r io.Reader, res chan<- RecordStopError)

func sendErrThenStop(err error, recs chan<- RecordStopError) {
	recs <- RecordStopError{
		Err: err,
	}
	recs <- RecordStopError{
		Stop: true,
	}
}

// DatasetOpts ...
type DatasetOpts struct {
	NumGo        int
	Epochs       int
	CacheRoot    string
	Logger       io.Writer
	NoCache      bool
	PanicOnError bool
	Quit         chan struct{}
}

// DefaultDatasetOpts ...
var DefaultDatasetOpts = DatasetOpts{
	NumGo:        3,
	CacheRoot:    "/data/kite/",
	PanicOnError: true,
}

// Dataset wraps a generic dataset on s3 or local disk
type Dataset struct {
	opts  DatasetOpts
	name  string
	files []string
	f     ProcessFn

	records chan pipeline.Record
}

// NewDataset with the provided options working on the provided files and using `f` to process each file
func NewDataset(opts DatasetOpts, name string, f ProcessFn, files ...string) *Dataset {
	if len(files) == 0 {
		panic(fmt.Errorf("no files given to dataset source %s", name))
	}

	return &Dataset{
		opts:    opts,
		name:    name,
		files:   files,
		f:       f,
		records: make(chan pipeline.Record, 100*opts.NumGo),
	}
}

// SourceOut implements pipeline.Source
func (s *Dataset) SourceOut() pipeline.Record {
	r, ok := <-s.records
	if !ok {
		return pipeline.Record{}
	}
	return r
}

// Name implements pipeline.Source
func (s *Dataset) Name() string {
	return s.name
}

// ForShard implements pipeline.Source
func (s *Dataset) ForShard(shard, totalShards int) (pipeline.Source, error) {
	ss := NewDataset(s.opts, s.name, s.f, s.files...)
	ss.start(shard, totalShards)
	return ss, nil
}

func (s *Dataset) start(shard, totalShards int) {
	type fileAndIdx struct {
		File string
		Idx  int
	}

	epochs := s.opts.Epochs
	if epochs == 0 {
		epochs = 1
	}

	var fc chan fileAndIdx
	if epochs < 0 {
		fc = make(chan fileAndIdx, len(s.files))
		go func() {
			defer close(fc)
			for {
				for i, f := range s.files {
					fc <- fileAndIdx{
						File: f,
						Idx:  i,
					}
				}
			}
		}()
	} else {
		fc = make(chan fileAndIdx, len(s.files)*epochs)
		for e := 0; e < epochs; e++ {
			for i, f := range s.files {
				fc <- fileAndIdx{
					File: f,
					Idx:  i,
				}
			}
		}
		close(fc)
	}

	jobs := make([]workerpool.Job, 0, s.opts.NumGo)
	for i := 0; i < s.opts.NumGo; i++ {
		jobs = append(jobs, func() error {
			for {
				select {
				case <-s.opts.Quit:
					// NOTE: we do NOT drain the fc channel because if epochs is less than 0, this
					// will never finish (see above)
					return nil
				case f, ok := <-fc:
					if !ok {
						// file channel closed, we are done
						return nil
					}

					if (f.Idx % totalShards) != shard {
						continue
					}

					r := s.reader(f.File, f.Idx)
					if r == nil {
						// can happen if we got an error opening the file and did not panic
						continue
					}

					var stop bool
					func() {
						recs := make(chan RecordStopError)
						go s.f(f.File, r, recs)
						for {
							select {
							case <-s.opts.Quit:
								// recieved quit signal, we are done
								stop = true
								return
							case rec, ok := <-recs:
								if !ok {
									// all records consumed, done
									return
								}
								if stop = rec.Stop; stop {
									return
								}
								if rec.Err != nil {
									e := errors.Errorf("error processing record in %s (%d): %v", f.File, f.Idx, rec.Err)
									if s.opts.PanicOnError {
										panic(e)
									}
									log.Println(e)
								} else {
									s.records <- rec.Record
								}
							}
						}
					}()

					if err := r.Close(); err != nil {
						s.logf("error closing %s (%d): %v", f.File, f.Idx, err)
					}

					if stop {
						return nil
					}
				}
			}
		})
	}

	pool := workerpool.New(s.opts.NumGo)
	pool.Add(jobs)
	go func() {
		pool.Wait()
		pool.Stop()
		close(s.records)
	}()
}

func (s *Dataset) reader(path string, idx int) io.ReadCloser {
	var r io.ReadCloser
	var err error
	if !awsutil.IsS3URI(path) {
		r, err = os.Open(path)
	} else if s.opts.NoCache {
		r, err = awsutil.NewS3Reader(path)
	} else {
		r, err = awsutil.NewCachedS3ReaderWithOptions(awsutil.CachedReaderOptions{
			CacheRoot: s.opts.CacheRoot,
			Logger:    s.opts.Logger,
		}, path)
	}

	if err != nil {
		e := errors.Errorf("error reading %s (%d): %v", path, idx, err)
		if s.opts.PanicOnError {
			panic(e)
		}
		log.Println(e)
		return nil
	}
	return r
}

func (s *Dataset) logf(fstr string, args ...interface{}) {
	fstr = "%s: " + fstr
	args = append([]interface{}{s.name}, args...)
	logf(s.opts.Logger, fstr, args...)
}

// JSONProcessFn ...
func JSONProcessFn(template pipeline.Sample) ProcessFn {
	return func(name string, r io.Reader, recs chan<- RecordStopError) {
		r, gzipped, err := isGzipped(r)
		if err != nil {
			sendErrThenStop(err, recs)
			return
		}

		if gzipped {
			var err error
			r, err = gzip.NewReader(r)
			if err != nil {
				sendErrThenStop(err, recs)
				return
			}
		}

		dec := json.NewDecoder(r)

		typ := reflect.TypeOf(template)
		for {
			var valPtr interface{}
			if typ.Kind() == reflect.Ptr {
				valPtr = reflect.New(typ.Elem()).Interface()
			} else {
				valPtr = reflect.New(typ).Interface()
			}

			err := dec.Decode(&valPtr)
			if err == io.EOF {
				break
			}
			if err != nil {
				sendErrThenStop(err, recs)
				break
			}

			var val pipeline.Sample
			if typ.Kind() == reflect.Ptr {
				val = valPtr.(pipeline.Sample)
			} else {
				val = reflect.ValueOf(valPtr).Elem().Interface().(pipeline.Sample)
			}

			recs <- RecordStopError{
				Record: pipeline.Record{
					Key:   name, // TODO: can we do something smarter here?
					Value: val,
				},
			}

		}
		close(recs)
	}
}

// EMRProcessFn ...
func EMRProcessFn(maxRecords, maxRecordSize int) ProcessFn {
	var count int64
	return func(name string, r io.Reader, recs chan<- RecordStopError) {
		iter := awsutil.NewEMRIterator(r)

		var stop bool
		for iter.Next() {
			if maxRecordSize > 0 && len(iter.Value()) > maxRecordSize {
				continue
			}

			recs <- RecordStopError{
				Record: pipeline.Record{
					Key: iter.Key(),
					Value: pipeline.Keyed{
						Key:    iter.Key(),
						Sample: sample.ByteSlice(iter.Value()),
					},
				},
			}

			v := atomic.AddInt64(&count, 1)
			if maxRecords > 0 && v >= int64(maxRecords) {
				stop = true
				break
			}
		}

		if err := iter.Err(); err != nil {
			recs <- RecordStopError{
				Err: fmt.Errorf("iterator error for %s: %v", name, err),
			}
		}

		if stop {
			recs <- RecordStopError{
				Stop: true,
			}
		}

		close(recs)
	}
}

// ReadProcessFn ...
func ReadProcessFn(maxRecordSize int) ProcessFn {
	return func(name string, r io.Reader, recs chan<- RecordStopError) {
		defer close(recs)
		var buf []byte
		if maxRecordSize > 0 {
			lr := io.LimitReader(r, int64(maxRecordSize)).(*io.LimitedReader)

			var err error
			buf, err = ioutil.ReadAll(lr)
			if err != nil {
				sendErrThenStop(errors.New("error reading entry for %s: %v", name, err), recs)
				return
			}

			if lr.N <= 0 {
				// check if we hit the limit exactly, or if there are still bytes left to be consumed
				if _, err := r.Read(make([]byte, 1)); err != io.EOF {
					// there are still bytes left to be consumed, skip this entry
					return
				}
			}
		} else {
			var err error
			buf, err = ioutil.ReadAll(r)
			if err != nil {
				sendErrThenStop(errors.New("error reading entry for %s: %v", name, err), recs)
				return
			}
		}

		recs <- RecordStopError{
			Record: pipeline.Record{
				Key: name,
				Value: pipeline.Keyed{
					Key:    name,
					Sample: sample.ByteSlice(buf),
				},
			},
		}
	}
}
