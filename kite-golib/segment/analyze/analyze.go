package analyze

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kiteco/kiteco/kite-golib/awsutil"
	"github.com/kiteco/kiteco/kite-golib/segment/track"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
	"github.com/pkg/errors"
)

const (
	maxDecodeErrorsSample = 100
	maxReadAttempts       = 10
	retryDelay            = 10 * time.Second
)

// MessageType distinguishes different Segment message types
type MessageType string

const (
	// TrackMessage is used for logging Track events in Segment
	TrackMessage MessageType = "track"
	// IdentifyMessage is used for logging Identify events in Segment
	IdentifyMessage MessageType = "identify"
)

// Results holds information about the results of an analysis.
type Results struct {
	Err                error
	CompletedURIs      int64
	DecodeErrors       int64
	DecodeErrorsSample []error
	ProcessedEvents    int64
	SkippedEvents      int64
}

// MessageID uniquely identifies a message as well as the URI from which it was sourced.
type MessageID struct {
	URI string // URI for the set of messages that contains the message.
	ID  string // ID of the message in the above set. This ID is globally unique.
}

// Metadata contains information related to a single event.
type Metadata struct {
	ID                MessageID
	Timestamp         time.Time
	OriginalTimestamp time.Time // the original, localized timestamp of the log according to the client
	EventName         string
	Type              MessageType
}

// String implements fmt.Stringer
func (m MessageID) String() string {
	return fmt.Sprintf("%s::%s", m.URI, m.ID)
}

// Opts for analysis
type Opts struct {
	NumGo  int
	Logger io.Writer
	Cache  string

	// MaxDecodeErrors, if >= 0, defines the point at which the analyzer quits if that many decode errors are exceeded
	MaxDecodeErrors int64
}

// Analyze takes a slice of Segment data URIs, an event name, and a handler function.
// The handler function should have the signature:
//     func(metadata Metadata, t *TrackedStruct) or func(Metadata,*TrackedStruct) bool
// The boolean indicates if the analyzer should keep going, e.g if its false then the analyzer will
// stop sending events.
// It downloads the events from the URIs, and for every event that matches eventName, it unmarshalls
// the event into a struct of type TrackedStruct and calls handler on the unmarshalled event.
// If eventName is blank, event name filtering is not performed.
// The handler will only be called from one thread, so it does not need to be thread safe.
func Analyze(URIs []string, numThreads int, eventName string, handler interface{}) Results {
	opts := Opts{
		NumGo:  numThreads,
		Logger: os.Stderr,
	}
	return WithOpts(opts, URIs, eventName, handler)
}

// WithOpts takes a slice of Segment data URIs, an event name, and a handler function.
// The handler function should have the signature:
//     func(metadata Metadata, t *TrackedStruct) or func(Metadata,*TrackedStruct) bool
// The boolean indicates if the analyzer should keep going, e.g if its false then the analyzer will
// stop sending events.
// It downloads the events from the URIs, and for every event that matches eventName, it unmarshalls
// the event into a struct of type TrackedStruct and calls handler on the unmarshalled event.
// If eventName is blank, event name filtering is not performed.
// The handler will only be called from one thread, so it does not need to be thread safe.
func WithOpts(opts Opts, URIs []string, eventName string, handler interface{}) Results {
	inType := getHandlerInputType(handler)
	proto := reflect.Zero(inType).Interface()
	stoppable := reflect.TypeOf(handler).NumOut() == 1

	handlerFn := reflect.ValueOf(handler)

	return WithPrototypes(opts, URIs, map[string]interface{}{eventName: proto},
		func(metadata Metadata, t interface{}) bool {
			args := []reflect.Value{reflect.ValueOf(metadata)}
			if t == nil {
				args = append(args, reflect.Zero(inType))
			} else {
				args = append(args, reflect.ValueOf(t))
			}

			if stoppable {
				return handlerFn.Call(args)[0].Bool()
			}
			handlerFn.Call(args)
			return true
		})
}

// GetSingleEvent finds an event with the given message ID, unmarshalls it to the same type as
// prototype, and returns it if it is found (along with the associated metadata), or nil otherwise.
func GetSingleEvent(messageID MessageID, prototype interface{}) (interface{}, Metadata, Results) {
	var foundMetadata Metadata
	var foundEvent interface{}
	results := WithPrototypes(Opts{Logger: os.Stderr, NumGo: 1}, []string{messageID.URI},
		map[string]interface{}{"": prototype},
		func(metadata Metadata, t interface{}) bool {
			if metadata.ID.ID == messageID.ID {
				foundEvent = t
				foundMetadata = metadata
				return false
			}
			return true
		})
	return foundEvent, foundMetadata, results
}

// WithPrototypes runs analysis on events contained in the relevant URIs. For each event that matches a key of
// prototypes, it deserializes the event into an object of the same type as the corresponding value. If prototypes
// contains a key that is a blank string, the event is deserialized to an object of the same type as the corresponding
// value if the name does not match any other key.
// The values of prototypes can be pointers as well.
// The handler receives the metadata and deserialized message of the requested type (or nil if the deserialization
// failed), and should return true if the analysis should continue or false otherwise.
func WithPrototypes(opts Opts, URIs []string, prototypes map[string]interface{}, handler func(meta Metadata, t interface{}) bool) Results {
	p := analyzer{
		handler:         handler,
		prototypes:      prototypes,
		startTime:       time.Now(),
		totalURIs:       len(URIs),
		rawMessages:     make(chan *rawMessage, opts.NumGo*10),
		logger:          opts.Logger,
		cacheRoot:       opts.Cache,
		maxDecodeErrors: opts.MaxDecodeErrors,
	}

	// quit can be called max of NumGo + 2 times,
	// so sends to quit will never block.
	quit := make(chan struct{}, opts.NumGo+2)

	var jobs []workerpool.Job
	for _, URI := range URIs {
		closureURI := URI
		jobs = append(jobs, func() error {
			err := p.getRawMessages(closureURI)
			// If reading a URI failed, we want to stop the entire operation
			if err != nil {
				quit <- struct{}{}
			}
			return err
		})
	}

	pool := workerpool.New(opts.NumGo)
	go func() {
		pool.Add(jobs)
		pool.Wait()
		quit <- struct{}{}
	}()

	go func() {
		select {
		case <-quit:
			pool.Stop()          // remove any unstarted jobs
			pool.Wait()          // wait for already started jobs to finish
			close(p.rawMessages) // close raw messages so flushing go routine finishes
		}
	}()

	for msg := range p.rawMessages {
		readEvents := atomic.LoadInt64(&p.readEvents)
		processedEvents := atomic.LoadInt64(&p.processedEvents)
		if processedEvents%1000 == 0 {
			p.logf("read events: %d, processed: %d\n", readEvents, processedEvents)
		}
		if keepGoing := p.handle(msg.metadata, msg.rawEvent); !keepGoing {
			quit <- struct{}{} // client requested that we stop
			break
		}
	}

	go func() {
		for range p.rawMessages {
			// flush remaining messages from
			// workers that were already in progress
			// so that we do not leak any go routines
			// since workers will block on sends to p.rawMessages.
		}
	}()

	return Results{
		Err:                pool.Err(),
		CompletedURIs:      p.completedURIs,
		DecodeErrors:       p.decodeErrors,
		DecodeErrorsSample: p.decodeErrorsSample,
		ProcessedEvents:    p.processedEvents,
		SkippedEvents:      p.skippedEvents,
	}
}

type rawMessage struct {
	metadata Metadata
	rawEvent json.RawMessage
}

type analyzer struct {
	handler    func(meta Metadata, t interface{}) bool
	prototypes map[string]interface{}
	startTime  time.Time
	totalURIs  int

	rawMessages chan *rawMessage

	lock               sync.Mutex
	completedURIs      int64
	readEvents         int64
	processedEvents    int64
	skippedEvents      int64
	decodeErrors       int64   // Number of errors arising from decode failures
	decodeErrorsSample []error // a sampling of the decode errors

	logger    io.Writer
	cacheRoot string

	maxDecodeErrors int64
}

func (p *analyzer) logf(format string, args ...interface{}) {
	if p.logger != nil {
		fmt.Fprintf(p.logger, format, args...)
	}
}

func (p *analyzer) handle(metadata Metadata, rawEvent json.RawMessage) bool {
	// we already checked in getRawMessages() whether the event name matches a key of prototypes
	// (or there's a blank key)
	protoName := getProtoName(metadata.Type, metadata.EventName)
	proto := p.prototypes[protoName]
	if _, found := p.prototypes[protoName]; !found {
		proto = p.prototypes[""]
	}

	typ := reflect.TypeOf(proto)

	var eventPtr interface{}
	if typ.Kind() == reflect.Ptr {
		eventPtr = reflect.New(typ.Elem()).Interface()
	} else {
		eventPtr = reflect.New(typ).Interface()
	}

	var event interface{}

	if err := json.Unmarshal(rawEvent, eventPtr); err != nil {
		var stop bool

		p.lock.Lock()
		p.decodeErrors++
		if p.maxDecodeErrors >= 0 && p.decodeErrors > p.maxDecodeErrors {
			stop = true
		}
		if len(p.decodeErrorsSample) < maxDecodeErrorsSample {
			p.decodeErrorsSample = append(p.decodeErrorsSample,
				fmt.Errorf("error decoding json %s : %v", string(rawEvent), err))
		}
		p.lock.Unlock()

		if stop {
			return false
		}
	} else {
		if typ.Kind() == reflect.Ptr {
			event = eventPtr
		} else {
			event = reflect.ValueOf(eventPtr).Elem().Interface()
		}
	}

	keepGoing := p.handler(metadata, event)
	atomic.AddInt64(&p.processedEvents, 1)
	return keepGoing
}

func getProtoName(eventType MessageType, eventName string) string {
	if eventType == IdentifyMessage {
		return "identify"
	}
	return eventName
}

func (p *analyzer) getRawMessages(URI string) error {
	defer func() {
		d := time.Since(p.startTime)
		c := int(atomic.LoadInt64(&p.completedURIs))
		rem := p.totalURIs - c
		eta := time.Duration(int64(rem) * (int64(d) / int64(c)))
		if c%10 == 0 {
			p.logf("reading URIs: %d out of %d in %s, ETA: %s\n", c, p.totalURIs, d, eta)
		}
	}()

	defer atomic.AddInt64(&p.completedURIs, 1)

	readFile := func() (io.Reader, error) {
		var r io.ReadCloser
		var err error

		if p.cacheRoot != "" {
			r, err = awsutil.NewCachedS3ReaderWithOptions(awsutil.CachedReaderOptions{
				CacheRoot: p.cacheRoot,
				Logger:    p.logger,
			}, URI)
		} else {
			r, err = awsutil.NewS3Reader(URI)
		}

		if err != nil {
			return nil, err
		}
		defer r.Close()

		var buf bytes.Buffer

		if _, err := io.Copy(&buf, r); err != nil {
			return nil, err
		}

		return &buf, nil
	}

	var r io.Reader
	var err error

	for a := 0; a < maxReadAttempts; a++ {
		r, err = readFile()
		if err == nil {
			break
		}
		p.logf("open attempt #%d/%d: error opening %s: %v\n", a+1, maxReadAttempts, URI, err)
		if a < maxReadAttempts-1 {
			time.Sleep(retryDelay)
		}
	}
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(r)
	for {
		var log segmentLog
		if err := decoder.Decode(&log); err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "failure in JSON unmarshalling")
		}

		protoName := getProtoName(log.Type, log.Event)
		_, hasBlank := p.prototypes[""]
		if _, found := p.prototypes[protoName]; !found && !hasBlank {
			atomic.AddInt64(&p.skippedEvents, 1)
			continue
		}

		var rawEvent *json.RawMessage
		switch log.Type {
		case TrackMessage:
			rawEvent = getRawTrackEvent(&log)
		case IdentifyMessage:
			rawEvent = &log.Traits
		default:
			atomic.AddInt64(&p.skippedEvents, 1)
			continue
		}

		metadata := Metadata{
			ID:                MessageID{URI: URI, ID: log.MessageID},
			Timestamp:         log.Timestamp,
			OriginalTimestamp: parseOriginalTimestamp(log.OriginalTimestamp),
			EventName:         log.Event,
			Type:              log.Type,
		}
		p.rawMessages <- &rawMessage{metadata: metadata, rawEvent: *rawEvent}
		atomic.AddInt64(&p.readEvents, 1)
	}
	return nil
}

// -

type segmentLog struct {
	MessageID         string          `json:"messageId"`
	Type              MessageType     `json:"type"`
	Event             string          `json:"event"`
	Properties        json.RawMessage `json:"properties"`
	Traits            json.RawMessage `json:"traits"`
	Timestamp         time.Time       `json:"timestamp"`
	OriginalTimestamp string          `json:"originalTimestamp"`
}

func getRawTrackEvent(log *segmentLog) *json.RawMessage {
	// The message might be in one of two formats, which we both handle. Either:
	// - the relevant fields are top-level fields in Properties (old format)
	// - the relevant fields are under a single (track.EventField) field in Properties (new format)
	// TODO(damian): Once we have a comfortable history of new messages, remove handling of the old format.
	var props map[string]json.RawMessage
	if err := json.Unmarshal(log.Properties, &props); err != nil {
		// assume the message is in the old format if there's a decode failure
		return &log.Properties
	}
	if val, ok := props[track.EventField]; ok {
		// new format
		return &val
	}
	// old format
	return &log.Properties
}

// Given a function with a signature of func(metadata Metadata, t *TrackedStruct), returns the type of
// *TrackedStruct. If the function does not match the signature, panic.
func getHandlerInputType(f interface{}) reflect.Type {
	fType := reflect.TypeOf(f)
	if fType.Kind() != reflect.Func {
		panic("expected a handler function")
	}
	if fType.NumOut() > 1 {
		panic("expected handler function to return at most one argument")
	}

	if fType.NumOut() == 1 {
		if fType.Out(0).Kind() != reflect.Bool {
			panic("expected handler function to return a bool or nothing")
		}
	}

	if fType.NumIn() != 2 {
		panic("expected handler function to have two arguments")
	}
	if fType.In(0) != reflect.TypeOf(Metadata{}) {
		panic("expected Metadata as first argument of handler")
	}
	in := fType.In(1)
	if in.Kind() != reflect.Ptr {
		panic("expected second handler argument to have pointer type")
	}
	return in
}

// parse the log's originalTimestamp field, which is localized to the client's time zone. There seem to be different
// formats at play (different client libraries?). If parsing fails, return a blank time.Time.
func parseOriginalTimestamp(ts string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z07:00", ts)
	if err == nil {
		return t
	}
	t, err = time.Parse("2006-01-02T15:04:05Z0700", ts)
	if err != nil {
		return time.Time{}
	}
	return t
}
