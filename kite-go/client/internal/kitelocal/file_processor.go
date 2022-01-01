package kitelocal

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/client/internal/kitelocal/driver"
	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-go/core"
	"github.com/kiteco/kiteco/kite-go/diff"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/lang/python"
	"github.com/kiteco/kiteco/kite-go/lang/unsupported"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/proto"
	"github.com/kiteco/kiteco/kite-go/userids"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

type stateKey struct {
	filename string
	source   string
	hash     string
}

type fileDriver struct {
	driver   core.FileDriver
	endpoint http.Handler
	hash     string
	ts       time.Time
}

type fileProcessor struct {
	userIDs userids.IDs

	m             sync.Mutex
	guardedStates map[stateKey]*fileDriver

	python *python.Services
	local  localcode.Context

	debug           bool
	testing         bool
	metricsDisabled bool
}

func newFileProcessor(python *python.Services, local localcode.Context, userIDs userids.IDs, metricsDisabled bool, debug bool) *fileProcessor {
	return &fileProcessor{
		userIDs:       userIDs,
		guardedStates: make(map[stateKey]*fileDriver),
		python:        python,
		local:         local,
		debug:         debug,
	}
}

// --

// Driver implements driver.Provider
func (f *fileProcessor) Driver(ctx kitectx.Context, filename, editor, state string) (*driver.State, bool) {
	ctx.CheckAbort()

	// Drivers use unix paths
	remotePath, err := localpath.ToUnix(filename)
	if err != nil {
		log.Println("Driver: error converting to unix path:", err)
		return nil, false
	}

	// Lowercase paths on windows to ensure consistent casing.
	if runtime.GOOS == "windows" {
		remotePath = strings.ToLower(remotePath)
	}

	fd, ok := f.fileDriver(remotePath, editor, state)
	if !ok {
		return nil, ok
	}

	return &driver.State{
		Filename:      filename,
		Editor:        editor,
		State:         fd.hash,
		FileDriver:    fd.driver,
		BufferHandler: fd.endpoint,
	}, true
}

// DriverFromContent implements driver.Provider
func (f *fileProcessor) DriverFromContent(ctx kitectx.Context, filename, editor, content string, cursor int) *driver.State {
	ctx.CheckAbort()

	// Drivers use unix paths
	remotePath, err := localpath.ToUnix(filename)
	if err != nil {
		log.Println("DriverFromContent: error converting to unix path:", err)
		return nil
	}

	// Lowercase paths on windows to ensure consistent casing.
	if runtime.GOOS == "windows" {
		remotePath = strings.ToLower(remotePath)
	}

	fd := f.fileDriverFromContent(ctx, remotePath, editor, content, cursor)
	return &driver.State{
		Filename:      filename,
		Editor:        editor,
		State:         fd.hash,
		FileDriver:    fd.driver,
		BufferHandler: fd.endpoint,
	}
}

func (f *fileProcessor) LatestDriver(ctx kitectx.Context, unixFilepath string) *python.UnifiedDriver {
	ctx.CheckAbort()

	// Drivers use unix paths
	remotePath := unixFilepath
	// Lowercase paths on windows to ensure consistent casing.
	if runtime.GOOS == "windows" {
		remotePath = strings.ToLower(remotePath)
	}

	f.m.Lock()
	defer f.m.Unlock()

	fd, ok := f.latestFileDriverLocked(remotePath, "")
	if !ok {
		return nil
	}

	unified, ok := fd.driver.(*python.UnifiedDriver)
	if !ok {
		return nil
	}
	return unified
}

// --

type eventResponse struct {
	results []interface{}
	state   string
	resend  bool
}

func (f *fileProcessor) fileDriver(filename, source, hash string) (*fileDriver, bool) {
	f.m.Lock()
	defer f.m.Unlock()
	return f.fileDriverLocked(filename, source, hash)
}

func (f *fileProcessor) fileDriverLocked(filename, source, hash string) (*fileDriver, bool) {
	sk := stateKey{filename, source, hash}
	fd, ok := f.guardedStates[sk]
	if ok {
		fd.ts = time.Now()
	}
	return fd, ok
}

func (f *fileProcessor) fileDriverFromContent(ctx kitectx.Context, filename, source, content string, cursor int) *fileDriver {
	ctx.CheckAbort()

	f.m.Lock()
	defer f.m.Unlock()

	hash := fmt.Sprintf("%x", md5.Sum([]byte(content)))
	if fd, ok := f.fileDriverLocked(filename, source, hash); ok {
		f.logf("!! using existing fd (%s, %s, %s)", filepath.Base(filename), source, hash)
		return fd
	}

	f.logf("!! new fd from content (%s, %s, %s)", filepath.Base(filename), source, hash)
	fd := f.newFileDriver(filename)

	// Initialize the driver by passing in a "dummy" event
	evt := &event.Event{
		Action:       proto.String("edit"),
		Filename:     proto.String(filename),
		Source:       proto.String(source),
		Text:         proto.String(content),
		TextMD5:      proto.String(hash),
		TextChecksum: proto.Uint64(spooky.Hash64([]byte(content))),
		Selections: []*event.Selection{
			&event.Selection{
				Start: proto.Int64(int64(cursor)),
				End:   proto.Int64(int64(cursor)),
			},
		},
	}

	fd.driver.HandleEvent(ctx, evt)
	fd.driver.CollectOutput()

	sk := stateKey{filename, source, hash}
	f.guardedStates[sk] = fd

	return fd
}

func (f *fileProcessor) reset() {
	f.m.Lock()
	defer f.m.Unlock()
	f.guardedStates = make(map[stateKey]*fileDriver)
}

// NOTE: callers are responsible for wrapping calls to this function in a timeout.
func (f *fileProcessor) handleEvent(ctx kitectx.Context, evt *event.Event) (*eventResponse, error) {
	ctx.CheckAbort()

	f.m.Lock()
	defer f.m.Unlock()
	defer f.cleanupLocked()

	sk := stateKey{
		filename: evt.GetFilename(),
		source:   evt.GetSource(),
		hash:     evt.GetTextMD5(),
	}

	// Look for an existing fileDriver for this file/source/hash.
	// If we couldn't find an existing fileDriver, we'll have to create a new one
	// Prefer creating one from full text if the full text is present, otherwise
	// use the reference state to initialize the contents of a new fileDriver

	fd, ok := f.guardedStates[sk]
	switch {
	case ok:
		f.logf("!! using existing fd (%s, %s, %s)", filepath.Base(sk.filename), sk.source, sk.hash)
		fd.ts = time.Now()

		// Zero out diffs, this driver is already in the correct state
		evt.Diffs = nil

	case evt.GetText() != "":
		// We have the full text, use it
		f.logf("!! new fd from text (%s, %s, %s)", filepath.Base(sk.filename), sk.source, sk.hash)

		fd = f.newFileDriver(sk.filename)
		f.guardedStates[sk] = fd

	case evt.GetReferenceState() != "":
		// We have a reference state, use it
		f.logf("!! new fd from ref %s (%s, %s, %s)", evt.GetReferenceState(), filepath.Base(sk.filename), sk.source, sk.hash)

		refsk := stateKey{
			filename: evt.GetFilename(),
			source:   evt.GetSource(),
			hash:     evt.GetReferenceState(),
		}

		// Find the reference state
		refd, ok := f.guardedStates[refsk]
		if !ok {
			f.logf("!! could not find ref %s (%s, %s)", evt.GetReferenceState(), filepath.Base(sk.filename), sk.source)
			break
		}

		refbuf := refd.driver.Bytes()
		fd = f.newFileDriver(sk.filename)

		// Initialize contents with the reference state
		fd.driver.SetContents(refbuf)
		f.guardedStates[sk] = fd

	default:
		// No existing match, no full text and no reference state, assume latest state is reference state
		// TODO(tarak): this is primarily for backwards compatibility with clients still sending events
		// sequentially.
		refd, ok := f.latestFileDriverLocked(sk.filename, sk.source)
		if !ok {
			f.logf("!! nothing I can do :(...")
			break
		}

		f.logf("!! assuming latest fd as ref (%s, %s, %s)", filepath.Base(sk.filename), sk.source, refd.hash)

		refbuf := refd.driver.Bytes()
		fd = f.newFileDriver(sk.filename)

		// Initialize contents with the reference state
		fd.driver.SetContents(refbuf)
		f.guardedStates[sk] = fd
	}

	if fd == nil {
		f.logf("!! no fd  (%s, %s, %s)", filepath.Base(sk.filename), sk.source, sk.hash)
		return nil, fmt.Errorf("could not construct driver: no text or reference state")
	}

	// handleEvent is already called with a timeout so we do not need to add one here
	state := fd.driver.HandleEvent(ctx, evt)
	output := fd.driver.CollectOutput()
	fd.hash = state
	// set the initial timestamp of the driver to the timestamp the event was received
	if ts := evt.GetTimestamp(); ts > 0 {
		fd.ts = time.Unix(0, ts)
	}

	results := &eventResponse{
		state:   state,
		results: output,
		resend:  fd.driver.ResendText(),
	}

	return results, nil
}

// latestFileDriverLocked assumes f.m is locked
// source is ignored if an empty value is passed
func (f *fileProcessor) latestFileDriverLocked(filename, source string) (*fileDriver, bool) {
	var latest *fileDriver
	defer func() {
		if latest != nil {
			latest.ts = time.Now()
		}
	}()

	for sk, driver := range f.guardedStates {
		if sk.filename == filename && (source == "" || sk.source == source) {
			if latest == nil {
				latest = driver
			}
			if driver.ts.After(latest.ts) {
				latest = driver
			}
		}
	}
	return latest, latest != nil
}

const (
	maxStates = 20
)

// cleanupLocked assumes f.m is locked
func (f *fileProcessor) cleanupLocked() {
	for len(f.guardedStates) > maxStates {
		var ts time.Time
		var oldestSK stateKey

		for sk, driver := range f.guardedStates {
			if ts.IsZero() || driver.ts.Before(ts) {
				ts = driver.ts
				oldestSK = sk
			}
		}

		f.logf("deleting %s", oldestSK.hash)
		delete(f.guardedStates, oldestSK)
	}
}

func (f *fileProcessor) newFileDriver(filename string) *fileDriver {
	language := lang.FromFilename(filename)

	switch {
	case f.testing:
		// sleep to make tests work on Windows, see https://github.com/kiteco/kiteco/pull/6389#issuecomment-428929161
		// timer resolution on Windows is 100ns, on Unix it's ~1ns.
		// handleEvent() calls latestFileDriverLocked() and then newFileDriver().
		// Both methods update the timestamp of the latest driver to time.Now().
		// On Windows it happened that both invocations of time.Now() returned the same value.
		// cleanupLocked() and latestFileDriverLocked() can't handle equal timestamps properly
		// and can return the incorrect driver which then breaks the tests.
		time.Sleep(150 * time.Nanosecond)
		return &fileDriver{
			driver: diff.NewBufferDriver(),
			ts:     time.Now(),
		}
	case language == lang.Python:
		driver := python.NewUnifiedDriver(
			f.userIDs,
			filename,
			f.python,
			f.local,
			true,
		)
		if f.debug {
			driver.InitSetDebug()
		}
		return &fileDriver{
			driver:   driver,
			endpoint: python.NewDriverEndpoint(driver),
			ts:       time.Now(),
		}
	default:
		return &fileDriver{
			driver: unsupported.NewDriver(
				filename),
			ts: time.Now(),
		}
	}
}

func (f *fileProcessor) logf(msg string, objs ...interface{}) {
	if f.debug {
		log.Printf(msg, objs...)
	}
}
