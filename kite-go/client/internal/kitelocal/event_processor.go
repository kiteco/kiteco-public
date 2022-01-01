package kitelocal

import (
	"crypto/md5"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	spooky "github.com/dgryski/go-spooky"
	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/client/internal/localpath"
	"github.com/kiteco/kiteco/kite-go/diff"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/proto"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/stringindex"
)

const (
	historyDuration = time.Minute
)

type fileSource struct {
	filename string
	source   string
}

type eventData struct {
	hash       string
	text       string
	selections []*event.Selection
	ts         time.Time
}

type responseData struct {
	hash       string
	text       string
	selections []*event.Selection
	ts         time.Time
	resend     bool
}

type eventProcessor struct {
	maxFileSizeBytes func() int

	permissions component.PermissionsManager
	differ      *diff.Differ

	m         sync.Mutex
	events    map[fileSource]*eventData
	responses map[fileSource]*responseData
}

func newEventProcessor(permissions component.PermissionsManager, maxFileSizeBytes func() int) *eventProcessor {
	return &eventProcessor{
		maxFileSizeBytes: maxFileSizeBytes,
		permissions:      permissions,
		differ:           diff.NewDiffer(),
		events:           make(map[fileSource]*eventData),
		responses:        make(map[fileSource]*responseData),
	}
}

// reset and release the resources held by this event processor
func (e *eventProcessor) reset() {
	e.m.Lock()
	defer e.m.Unlock()

	for fs := range e.responses {
		delete(e.responses, fs)
	}

	for ev := range e.events {
		delete(e.events, ev)
	}
}

// update the content latest hash for the provided source, filename
func (e *eventProcessor) updateLatestResponse(source, filename, text string, sel []*event.Selection, resend bool) {
	e.m.Lock()
	defer e.m.Unlock()

	// Cleanup stale responses
	for fs, rd := range e.responses {
		if time.Since(rd.ts) > historyDuration {
			delete(e.responses, fs)
		}
	}

	fs := fileSource{filename: filename, source: source}
	e.responses[fs] = &responseData{
		hash:       fmt.Sprintf("%x", md5.Sum([]byte(text))),
		text:       text,
		selections: sel,
		ts:         time.Now(),
		resend:     resend,
	}
}

var (
	errUnsavedFile     = errors.New("unsaved file")
	errUnsupportedFile = errors.New("unsupported file")
	errSkipped         = errors.New("skipped")
	errFileTooLarge    = errors.New("file too large")
	errUnused          = errors.New("unused event")
	errDuplicate       = errors.New("duplicate")
)

// processEvent takes editor events and converts them to backend events.
// For edit events, it computes diffs and periodically resends the entire
// buffer. It also dedupes events.
func (e *eventProcessor) processEvent(ev *component.EditorEvent) (*event.Event, error) {
	start := time.Now()

	e.m.Lock()
	defer e.m.Unlock()

	// Cleanup any stale events
	for fs, ed := range e.events {
		if time.Since(ed.ts) > historyDuration {
			delete(e.events, fs)
		}
	}

	// convert selections from rune offset to byte offsets
	var selections []*event.Selection
	conv := stringindex.NewConverter(ev.Text)
	for _, sel := range ev.Selections {
		start, err := conv.OffsetToUTF8(int(sel.Start), sel.Encoding)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode selection start offset")
		}
		end, err := conv.OffsetToUTF8(int(sel.End), sel.Encoding)
		if err != nil {
			return nil, errors.Wrapf(err, "could not decode selection end offset")
		}
		selections = append(selections, &event.Selection{
			Start: proto.Int64(int64(start)),
			End:   proto.Int64(int64(end)),
		})
	}

	switch {
	case ev.Filename == "":
		return nil, errUnsavedFile
	}

	// send unix paths to backend
	remotePath, err := localpath.ToUnix(ev.Filename)
	if err != nil {
		return nil, err
	}

	// Lowercase paths on windows to ensure consistent casing.
	if runtime.GOOS == "windows" {
		remotePath = strings.ToLower(remotePath)
	}

	// copy data into the event that will be sent to the backend
	processed := &event.Event{
		Source:       proto.String(ev.Source),
		Action:       proto.String(ev.Action),
		Filename:     proto.String(remotePath),
		Text:         proto.String(ev.Text),
		TextChecksum: proto.Uint64(spooky.Hash64([]byte(ev.Text))),
		TextMD5:      proto.String(fmt.Sprintf("%x", md5.Sum([]byte(ev.Text)))),
		Selections:   selections,
		Timestamp:    proto.Int64(ev.Timestamp.UnixNano()),
	}

	ss := e.permissions.IsSupportedExtension(ev.Filename)

	// handle skip cases, no deduping or throttling
	// order here is important! unsupported language > skip
	switch {
	case !ss.EditEventSupported:
		return nil, errUnsupportedFile
	case ev.Action == "skip":
		return nil, errSkipped
	case len(ev.Text) >= e.maxFileSizeBytes():
		return nil, errFileTooLarge
	case ev.Action == "lost_focus":
		return nil, errUnused
	}

	fs := fileSource{filename: ev.Filename, source: ev.Source}
	ed := e.events[fs]
	rd := e.responses[fs]

	switch {
	case ed == nil:
		// set to an edit event to make sure text gets updated on backend
		processed.Action = proto.String("edit")
	case rd == nil:
		// no reference event to diff against, send full text
		processed.Action = proto.String("edit")
	case rd.resend:
		// need to resend full text
		processed.Action = proto.String("edit")
	case processed.GetAction() == "focus":
		// focus event, send full text
	case rd.text != processed.GetText():
		// compute the diff
		diffs := e.differ.Diff(rd.text, processed.GetText())
		for i := range diffs {
			processed.Diffs = append(processed.Diffs, &diffs[i])
		}

		// set to an edit event to make sure text gets updated on the backend
		processed.Action = proto.String("edit")
		processed.ReferenceState = proto.String(rd.hash)

	case selectionChanged(rd.selections, processed.GetSelections()):
		// make sure set to selection
		processed.Action = proto.String("selection")
		// technically this is also the reference state
		processed.ReferenceState = proto.String(rd.hash)
	default:
		return nil, errDuplicate
	}

	// Update event data to latest (but after grabbing values above)
	e.events[fs] = &eventData{
		text:       ev.Text,
		selections: selections,
		hash:       fmt.Sprintf("%x", md5.Sum([]byte(ev.Text))),
		ts:         time.Now(),
	}

	processed.ProcessingTime = proto.Int64(int64(time.Since(start)))
	processed.FirstSeen = proto.Int64(ev.Timestamp.UnixNano())

	return processed, nil
}

func selectionChanged(sel1, sel2 []*event.Selection) bool {
	if len(sel1) != len(sel2) {
		return true
	}

	sels := make(map[string]bool)
	for _, sel := range sel1 {
		key := fmt.Sprintf("%d-%d", sel.GetStart(), sel.GetEnd())
		sels[key] = true
	}
	for _, sel := range sel2 {
		key := fmt.Sprintf("%d-%d", sel.GetStart(), sel.GetEnd())
		if !sels[key] {
			return true
		}
	}
	return false
}
