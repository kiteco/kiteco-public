package event

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
)

const (
	numRetries           = 1
	iteratorInitialBatch = 10
)

// Iterator is an iterator that returns events in order from newest to oldest.
type Iterator struct {
	uid     int64
	stream  string
	fs      blockFileSystem
	manager *MetadataManager

	blocks []*Metadata
	dec    *json.Decoder
	ev     *Event
	err    error
}

func newIterator(uid int64, stream string, fs blockFileSystem, mm *MetadataManager) (*Iterator, error) {
	iter := &Iterator{
		uid:     uid,
		stream:  stream,
		fs:      fs,
		manager: mm,
	}
	dec, err := iter.next()
	if err != nil {
		return nil, err
	}
	iter.dec = dec
	return iter, nil
}

func newIteratorWithReader(uid int64, stream string, fs blockFileSystem, mm *MetadataManager, initial io.Reader) (*Iterator, error) {
	iter := &Iterator{
		uid:     uid,
		stream:  stream,
		fs:      fs,
		manager: mm,
	}
	dec, err := iter.nextWithReader(initial)
	if err != nil {
		return nil, err
	}
	iter.dec = dec
	return iter, nil
}

// Next populates i.ev with the next event in the iteration
// and returns its success after numRetries of retries.
func (i *Iterator) Next() bool {
	i.ev = &Event{}
	var retries int
	for retries <= numRetries {
		i.err = i.dec.Decode(i.ev)
		switch i.err {
		case nil:
			return true
		case io.EOF:
			retries++
			i.dec, i.err = i.next()
			if i.err != nil {
				return false
			}
			continue
		default:
			return false
		}
	}
	return false
}

// Event returns the event stored in i.ev.
func (i *Iterator) Event() *Event {
	return i.ev
}

// Err returns the current error for the iterator, unless
// the error is io.EOF.
func (i *Iterator) Err() error {
	if i.err == io.EOF {
		return nil
	}
	return i.err
}

// --

func (i *Iterator) next() (*json.Decoder, error) {
	var err error
	i.blocks, err = i.nextBlocks(i.blocks)
	if err != nil {
		return nil, err
	}
	if len(i.blocks) == 0 {
		return json.NewDecoder(new(bytes.Buffer)), nil
	}
	var readers []io.Reader
	for _, block := range i.blocks {
		readers = append(readers, newLazyBlockReader(i.fs, block))
	}

	return i.nextDecoder(readers...)
}

func (i *Iterator) nextWithReader(initial io.Reader) (*json.Decoder, error) {
	var err error
	i.blocks, err = i.nextBlocks(i.blocks)
	if err != nil {
		return nil, err
	}
	var readers []io.Reader
	readers = append(readers, initial)
	for _, block := range i.blocks {
		readers = append(readers, newLazyBlockReader(i.fs, block))
	}

	return i.nextDecoder(readers...)
}

func (i *Iterator) nextDecoder(readers ...io.Reader) (*json.Decoder, error) {
	multi := io.MultiReader(readers...)
	decomp, err := gzip.NewReader(multi)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize decompression: %s", err)
	}

	return json.NewDecoder(decomp), nil
}

func (i *Iterator) nextBlocks(prev []*Metadata) ([]*Metadata, error) {
	var err error
	var blocks []*Metadata
	if len(prev) == 0 {
		blocks, err = i.manager.Latest(iteratorInitialBatch, i.uid, i.stream)
		if err != nil {
			return nil, fmt.Errorf("cannot find latest blocks: %s", err)
		}
	} else {
		latest := prev[len(prev)-1]
		blocks, err = i.manager.Get(iteratorInitialBatch, i.uid, i.stream, latest.End)
		if err != nil {
			return nil, fmt.Errorf("cannot find blocks after %d: %s", latest.End, err)
		}
	}

	return blocks, nil
}

// --

type lazyBlockReader struct {
	fs       blockFileSystem
	metadata *Metadata
	buf      *bytes.Buffer
}

func newLazyBlockReader(fs blockFileSystem, metadata *Metadata) *lazyBlockReader {
	return &lazyBlockReader{
		fs:       fs,
		metadata: metadata,
	}
}

func (l *lazyBlockReader) Read(buf []byte) (int, error) {
	if l.buf == nil {
		block, err := l.fs.readBlock(l.metadata)
		if err != nil {
			return 0, fmt.Errorf("error reading block: %s", err)
		}
		l.buf = bytes.NewBuffer(block)
	}
	n, err := l.buf.Read(buf)
	if err == io.EOF {
		l.buf = nil
	}
	return n, err
}
