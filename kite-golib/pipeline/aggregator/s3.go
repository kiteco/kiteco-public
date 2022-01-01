package aggregator

import (
	"compress/gzip"
	"fmt"
	"io"

	"github.com/kiteco/kiteco/kite-golib/awsutil"

	"github.com/kiteco/kiteco/kite-golib/pipeline"
	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
)

// NewEMRWriter which writes files to the specified remote directory `dir` (remote or local).
// NOTE: if the directory is local then this aggregator cannot be distributed since
// we currently do not support copying the files from the other workers to the coordinator machine.
func NewEMRWriter(opts WriterOpts, name, dir string) *Writer {
	if opts.NumGo == 0 {
		opts.NumGo++
	}

	// the aws writer already handles compression
	opts.Compress = false

	opts.FileSuffix = ".emr"

	nef := func(w io.Writer, compress bool) Emitter {
		var gz *gzip.Writer
		if compress {
			gz = gzip.NewWriter(w)
			w = gz
		}
		return &emrWriterEmitter{
			emr: awsutil.NewEMRWriter(w),
			gz:  gz,
		}
	}

	return NewWriter(opts, name, dir, nef)
}

type emrWriterEmitter struct {
	emr *awsutil.EMRWriter
	gz  *gzip.Writer
}

func (e *emrWriterEmitter) Emit(s pipeline.Sample) (int, error) {
	ks := s.(pipeline.Keyed)
	bc := len([]byte(ks.Key)) + len(ks.Sample.(sample.ByteSlice))

	if err := e.emr.Emit(ks.Key, ks.Sample.(sample.ByteSlice)); err != nil {
		return 0, fmt.Errorf("error emitting emr writer sample '%s': %v", ks.Key, err)
	}
	return bc, nil
}

func (e *emrWriterEmitter) Close() error {
	if err := e.emr.Close(); err != nil {
		return err
	}
	if e.gz != nil {
		return e.gz.Close()
	}
	return nil
}
