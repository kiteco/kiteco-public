package aggregator

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/fileutil"
	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

const (
	// DoneFilename is the marker for the "DONE" file that is placed
	// in a directory once aggregation is complete
	DoneFilename = "DONE"
)

// ListDir is a simple wrapper around fileutil.ListDir
// which explicitly removes the "DONE" file placed by the
// writer from the returned list.
// TODO: where should we put this? The other option is
// to just not write content to the "DONE" file and then
// the awsutil.ListDir will do the right thing for remote directories
func ListDir(dir string) ([]string, error) {
	fs, err := fileutil.ListDir(dir)
	if err != nil {
		return nil, err
	}

	if len(fs) == 0 {
		return nil, fmt.Errorf("no files found for %s", dir)
	}

	for i, f := range fs {
		if strings.HasSuffix(f, DoneFilename) {
			return append(fs[:i], fs[i+1:]...), nil
		}
	}
	return fs, nil
}

// Emitter emits samples to a writer or encoder of some sort
type Emitter interface {
	Emit(pipeline.Sample) (int, error)
	Close() error
}

// NewEmitterFn returns a new emitter that emits to
// the specified writer, the second argument indicates if the
// emitter should compress the results
type NewEmitterFn func(io.Writer, bool) Emitter

// WriterOpts ...
type WriterOpts struct {
	NumGo      int
	Logger     io.Writer
	FilePrefix string
	FileSuffix string
	Compress   bool
	TmpDir     string

	// Deprecated, use MaxFileSizeBytes
	SamplesPerFile int

	MaxFileSizeBytes int
}

// DefaultWriterOpts ...
var DefaultWriterOpts = WriterOpts{
	NumGo:      1,
	FilePrefix: "file",
}

// Writer wraps a generic set of file writers
type Writer struct {
	opts WriterOpts
	name string
	dir  string
	nef  NewEmitterFn

	shard       int
	totalShards int

	samples chan pipeline.Sample

	pool *workerpool.Pool

	m            sync.Mutex
	files        []string
	workingFiles []*file
}

// NewWriter which writes files to the specified remote directory `dir` (remote or local).
// NOTE: if the directory is local then this aggregator cannot be distributed since
// we currently do not support copying the files from the other workers to the coordinator machine.
func NewWriter(opts WriterOpts, name, dir string, nef NewEmitterFn) *Writer {
	if opts.NumGo == 0 {
		opts.NumGo++
	}

	return &Writer{
		opts:    opts,
		name:    name,
		dir:     dir,
		nef:     nef,
		samples: make(chan pipeline.Sample, 100*opts.NumGo),
		pool:    workerpool.New(opts.NumGo),
	}
}

// Clone implements pipeline.Aggregator
func (w *Writer) Clone() pipeline.Dependent {
	return w
}

// ForShard implements pipeline.Aggregator
func (w *Writer) ForShard(shard, totalShards int) (pipeline.Aggregator, error) {
	ww := NewWriter(w.opts, w.name, w.dir, w.nef)
	ww.shard = shard
	ww.totalShards = totalShards
	ww.start()
	return ww, nil
}

func (w *Writer) start() {
	var jobs []workerpool.Job
	for i := 0; i < w.opts.NumGo; i++ {
		jobs = append(jobs, func() error {
			// if we encounter an error ever we have to
			// panic because otherwise the pipeline will hang
			// if all of the workers error out since the input
			// channel of samples will never be read from

			var f *file
			var count int
			var byteCount int
			reset := func() {
				if f != nil {
					noErr(w.completedFile(f))
				}
				var err error
				f, err = w.nextFile()
				noErr(err)
				count = 0
				byteCount = 0
			}

			reset()

			for s := range w.samples {
				bc, err := f.Emitter.Emit(s)
				noErr(err)

				byteCount += bc
				count++
				if w.opts.SamplesPerFile > 0 && count >= w.opts.SamplesPerFile {
					reset()
				}

				if w.opts.MaxFileSizeBytes > 0 && byteCount >= w.opts.MaxFileSizeBytes {
					reset()
				}

			}
			if f != nil {
				noErr(w.completedFile(f))
			}
			return nil
		})
	}

	w.pool.Add(jobs)
}

func (w *Writer) nextFile() (*file, error) {
	w.m.Lock()
	defer w.m.Unlock()

	name := fmt.Sprintf("shard-%d-of-%d-part-%d", w.shard, w.totalShards, len(w.files))
	if w.opts.FilePrefix != "" {
		name = fmt.Sprintf("%s-%s", w.opts.FilePrefix, name)
	}
	name += w.opts.FileSuffix
	if w.opts.Compress {
		name += ".gz"
	}

	path := fileutil.Join(w.dir, name)
	if !awsutil.IsS3URI(w.dir) {
		path += ".tmp"
	}

	wc, err := newBufferedWriter(w.opts.TmpDir, path)
	if err != nil {
		return nil, errors.Errorf("error creating writer '%s': %v", path, err)
	}

	f := &file{
		Path:    path,
		WC:      wc,
		Emitter: w.nef(wc, w.opts.Compress),
	}

	w.workingFiles = append(w.workingFiles, f)

	return f, nil
}

type file struct {
	Path    string
	WC      io.WriteCloser
	Emitter Emitter
}

func (f *file) Close() error {
	if err := f.Emitter.Close(); err != nil {
		return errors.Errorf("error closing emitter for '%s': %v", f.Path, err)
	}
	if err := f.WC.Close(); err != nil {
		return errors.Errorf("error closing writer for '%s': %v", f.Path, err)
	}
	return nil
}

func (w *Writer) completedFile(f *file) error {
	w.m.Lock()
	defer w.m.Unlock()

	// always close the file
	if err := f.Close(); err != nil {
		return err
	}

	if strings.HasSuffix(f.Path, ".tmp") {
		path := strings.TrimSuffix(f.Path, ".tmp")
		if err := os.Rename(f.Path, path); err != nil {
			return errors.Errorf("unable to rename %s -> %s: %v", f.Path, path, err)
		}

		// remove from working files list
		found := -1
		for i, ff := range w.workingFiles {
			if ff == f {
				found = i
				break
			}
		}

		if found > -1 {
			newWorking := append([]*file{}, w.workingFiles[:found]...)
			w.workingFiles = append(newWorking, w.workingFiles[found+1:]...)
		}
	}

	w.files = append(w.files, f.Path)
	return nil
}

// AggregateLocal implements pipeline.Aggregator
func (w *Writer) AggregateLocal(clones []pipeline.Aggregator) (pipeline.Sample, error) {
	// signal to writers that no more samples are coming
	close(w.samples)

	// wait for writers to complete
	if err := w.pool.Wait(); err != nil {
		return nil, fmt.Errorf("pool error: %v", err)
	}

	// stop writers
	w.pool.Stop()

	return sample.StringSlice(w.files), nil
}

// AggregateFromShard implements pipeline.Aggregator
func (w *Writer) AggregateFromShard(agg pipeline.Sample, shardSample pipeline.Sample, endpoint string) (pipeline.Sample, error) {
	if agg == nil {
		return shardSample, nil
	}

	files := agg.(sample.StringSlice)
	fs := shardSample.(sample.StringSlice)
	files = append(files, fs...)

	return files, nil
}

// FromJSON implements pipeline.Aggregator
func (w *Writer) FromJSON(data []byte) (pipeline.Sample, error) {
	var s sample.StringSlice
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return s, nil
}

// Finalize implements pipeline.Aggregator
func (w *Writer) Finalize() error {
	// write the done file to mark that we're finished
	return writeDoneFile(w.dir)
}

// Name implements pipeline.Aggregator
func (w *Writer) Name() string {
	return w.name
}

// In implements pipeline.Aggregator
func (w *Writer) In(s pipeline.Sample) {
	w.samples <- s
}

func writeDoneFile(dir string) error {
	outf, err := fileutil.NewBufferedWriter(fmt.Sprintf("%s/%s", dir, DoneFilename))
	if err != nil {
		return err
	}

	if _, err := outf.Write([]byte("done")); err != nil {
		outf.Close()
		return err
	}

	return outf.Close()
}

// newBufferedWriter opens a local or remote path for writing. If the path starts with
// "s3://", then this will write to a local buffer, copying to s3 on close. Otherwise,
// this will write to the local FS.
func newBufferedWriter(tmpDir, path string) (io.WriteCloser, error) {
	if awsutil.IsS3URI(path) {
		if tmpDir == "" {
			return awsutil.NewBufferedS3Writer(path)
		}
		return awsutil.NewBufferedS3WriterWithTmp(tmpDir, path)
	}
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return nil, err
	}
	return os.Create(path)
}

func noErr(err error) {
	if err != nil {
		panic(err)
	}
}
