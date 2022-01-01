package utils

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/go-errors/errors"
)

// SampleWriter ...
type SampleWriter struct {
	dir    string
	tmpdir string
	n      int64

	batchSize      int
	stepsPerFile   int
	samplesPerFile int

	steps int32

	samples []Sample
	files   int
	m       sync.Mutex
	wg      sync.WaitGroup
}

// Sample ...
type Sample struct {
	Context []int `json:"context"`
	Lang    int   `json:"lang"`
}

// NewSampleWriter ...
func NewSampleWriter(dir, tmpdir string, batchSize, stepsPerFile int) *SampleWriter {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
	return &SampleWriter{
		dir:            dir,
		tmpdir:         tmpdir,
		batchSize:      batchSize,
		stepsPerFile:   stepsPerFile,
		samplesPerFile: batchSize * stepsPerFile,
	}
}

// Flush ...
func (s *SampleWriter) Flush() error {
	s.m.Lock()
	defer s.m.Unlock()
	if len(s.samples) > 0 {
		s.wg.Wait()
		err := s.flushInternal(s.samples, s.n)
		if err != nil {
			return err
		}
	}
	return nil
}

// WriteSample ...
func (s *SampleWriter) WriteSample(sample Sample) error {
	s.m.Lock()
	defer s.m.Unlock()
	s.samples = append(s.samples, sample)
	if len(s.samples) >= s.samplesPerFile {
		s.wg.Wait()
		s.wg.Add(1)
		go func(samples []Sample, n int64, wg *sync.WaitGroup) {
			defer s.wg.Done()
			err := s.flushInternal(samples, n)
			if err != nil {
				log.Println("error flushing:", err)
			}
		}(s.samples, s.n, &s.wg)

		s.samples = nil
		s.files = 0
		s.n++
	}
	return nil
}

// StepsWritten ...
func (s *SampleWriter) StepsWritten() int {
	return int(atomic.LoadInt32(&s.steps))
}

func (s *SampleWriter) flushInternal(samples []Sample, n int64) error {
	for i, s := range samples {
		if len(s.Context) != len(samples[0].Context) {
			log.Fatalf("expected all samples to have length %d but got %d for sample %d", len(samples[0].Context), len(s.Context), i)
		}
	}

	tmpfile, err := ioutil.TempFile(s.tmpdir, "samplewriter")
	if err != nil {
		return err
	}

	gz := gzip.NewWriter(tmpfile)
	for _, enc := range samples {
		buf, err := json.Marshal(enc)
		if err != nil {
			return err
		}
		buf = append(buf, byte('\n'))
		_, err = gz.Write(buf)
		if err != nil {
			return err
		}
	}

	err = gz.Close()
	if err != nil {
		return err
	}

	err = tmpfile.Close()
	if err != nil {
		return err
	}

	fn := filepath.Join(s.dir, fmt.Sprintf("part-%05d.json.gz", n))
	err = os.Rename(tmpfile.Name(), fn)
	if err != nil {
		// You need to copy if source and destination are not in the same partition
		if terr, ok := err.(*os.LinkError); ok && terr.Err == syscall.EXDEV {
			// copyFile
			err = copyFile(tmpfile.Name(), fn)
		}
		if err != nil {
			return err
		}
	}

	log.Printf("%s is ready", fn)
	atomic.AddInt32(&s.steps, int32(s.stepsPerFile))
	return nil
}

func copyFile(source, target string) error {
	sourceFileStat, err := os.Stat(source)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return errors.Errorf("%s is not a regular file", source)
	}

	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	destination, err := os.Create(target)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, src)
	return err
}
