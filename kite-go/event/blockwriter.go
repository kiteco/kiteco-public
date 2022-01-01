package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// blockWriter holds the metadata for an event batch as well as everything necessary
// to compress and write a batch.
type blockWriter struct {
	uid     int64
	machine string

	blockSize int
	metadata  *Metadata

	fs      blockFileSystem
	manager *MetadataManager
	stream  string
	block   *blockBuffer

	wg sync.WaitGroup
}

// newBatchWriter creates a new BatchWriter.
func newBlockWriter(uid int64, machine string, blockSize int, stream string, fs blockFileSystem, mm *MetadataManager) *blockWriter {
	return &blockWriter{
		uid:       uid,
		machine:   machine,
		blockSize: blockSize,
		fs:        fs,
		manager:   mm,
		block:     newBlockBuffer(blockSize),
		stream:    stream,
	}
}

// write adds an event to the blockWriter's current batch.
func (b *blockWriter) writeEvent(event *Event) error {
	buf, err := json.Marshal(event)
	if err != nil {
		return err
	}
	// check if theres enough space to write this event
	if len(buf) > b.block.available() {
		b.flush()
	}

	err = b.block.writeEvent(buf)
	if err != nil {
		return err
	}

	if b.metadata == nil {
		b.metadata = &Metadata{
			Stream: b.stream,
			UserID: b.uid,
			Start:  event.GetTimestamp(),
		}
	}

	b.metadata.End = event.GetTimestamp()
	b.metadata.Count++

	return nil
}

// flush writes the blockWriter's current batch to s3 and the
// metadata to the metadata db. It then clears the current batch.
func (b *blockWriter) flush() error {
	if b.block.len() == 0 {
		return nil
	}

	b.wg.Add(1)
	go func(block *blockBuffer, m *Metadata, wg *sync.WaitGroup) {
		defer wg.Done()

		// catch panics which can happen on the b.manager.Add call
		defer func() {
			if err := recover(); err != nil {
				rollbar.PanicRecovery(err)
			}
		}()

		start := time.Now()

		buf := &bytes.Buffer{}
		err := block.writeTo(buf)
		if err != nil {
			b.printf("error constructing block: %v", err)
			return
		}

		hash := spooky.Hash64(buf.Bytes())
		m.Filename = fmt.Sprintf("%d/%s/%d", m.UserID, m.Stream, hash)

		m.Size = int64(buf.Len())
		// attempt to write block until it succeeds

		var retries int
		for {
			if err = b.fs.writeBlock(buf.Bytes(), m); err != nil {
				b.printf("error writing block: %v", err)
			}
			if err == nil {
				break
			}
			b.printf("write block failed after %s, sleeping and retrying: %+v", time.Since(start), m)
			time.Sleep(5 * time.Second)
			retries++
		}

		b.manager.Add(m)
		endTime := time.Since(start)
		b.printf("flushed block in %s: %+v", endTime, m)

		flushDuration.RecordDuration(endTime)
		eventsPerBlock.Record(m.Count)
		bytesPerBlock.Record(m.Size)
		retriesPerFlush.Record(int64(retries))

	}(b.block, b.metadata, &b.wg)

	b.block = newBlockBuffer(b.blockSize)
	b.metadata = nil

	return nil
}

// Waits for goroutines to complete
func (b *blockWriter) wait() {
	b.wg.Wait()
}

// --

func (b *blockWriter) printf(msg string, vars ...interface{}) {
	log.Printf("event.blockWriter (%d, %s) %s", b.uid, b.machine, fmt.Sprintf(msg, vars...))
}
