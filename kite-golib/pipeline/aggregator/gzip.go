package aggregator

import (
	"bytes"
	"compress/gzip"
	"io"
)

type gzipWriter struct {
	buf *bytes.Buffer
	gz  *gzip.Writer
}

func newGzipWriter(w io.Writer) *gzipWriter {
	var buffer bytes.Buffer
	w = io.MultiWriter(w, &buffer)
	return &gzipWriter{
		buf: &buffer,
		gz:  gzip.NewWriter(w),
	}
}

// WriteCompressedSize returns the compressed size of the bytes that are passed
func (w *gzipWriter) WriteCompressedSize(buf []byte) (int, error) {
	defer w.buf.Reset()
	_, err := w.gz.Write(buf)
	if err != nil {
		return 0, err
	}
	if err := w.gz.Flush(); err != nil {
		return 0, err
	}
	return w.buf.Len(), nil
}

func (w *gzipWriter) Close() error {
	w.buf.Reset()
	return w.gz.Close()
}
