package aggregator

import (
	"encoding/json"
	"io"

	"github.com/kiteco/kiteco/kite-golib/pipeline"
)

// NewJSONWriter which writes json entries to file
func NewJSONWriter(opts WriterOpts, name, dir string) *Writer {
	nef := func(w io.Writer, compress bool) Emitter {
		var gz *gzipWriter
		if compress {
			gz = newGzipWriter(w)
		}
		return &jsonEmitter{
			w:  w,
			gz: gz,
		}
	}
	opts.FileSuffix = ".json"
	return NewWriter(opts, name, dir, nef)
}

type jsonEmitter struct {
	w  io.Writer
	gz *gzipWriter
}

func (je *jsonEmitter) Emit(s pipeline.Sample) (int, error) {
	buf, err := json.Marshal(s)
	if err != nil {
		return 0, err
	}
	buf = append(buf, []byte("\n")...)

	if je.gz == nil {
		_, err := je.w.Write(buf)
		if err != nil {
			return 0, err
		}
		return len(buf), nil
	}
	return je.gz.WriteCompressedSize(buf)
}

func (je *jsonEmitter) Close() error {
	if je.gz != nil {
		return je.gz.Close()
	}
	return nil
}
