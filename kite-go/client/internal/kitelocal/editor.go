package kitelocal

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/kiteco/kiteco/kite-go/client/component"
	"github.com/kiteco/kiteco/kite-go/event"
	"github.com/kiteco/kiteco/kite-go/localcode"
	"github.com/kiteco/kiteco/kite-go/response"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
	"github.com/kiteco/kiteco/kite-golib/workerpool"
)

var fileEventTimeout = time.Second

// TestSetFileEventTimeout sets a new event timeout for tests, it returns the previous value
func TestSetFileEventTimeout(timeout time.Duration) time.Duration {
	old := fileEventTimeout
	fileEventTimeout = timeout
	return old
}

// HandleEditorEvent is the handler for /clientapi/editor/event
func (m *Manager) handleEditorEvent(w http.ResponseWriter, r *http.Request) {
	var event component.EditorEvent
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if event.Filename == "" {
		http.Error(w, errUnsupportedFile.Error(), http.StatusInternalServerError)
		return
	}

	event.Timestamp = time.Now()
	m.pool.Add([]workerpool.Job{
		func() error {
			m.processEvent(&event)
			return nil
		},
	})

	w.WriteHeader(http.StatusOK)
}

func (m *Manager) processEvent(editorEvt *component.EditorEvent) {
	defer func() {
		if r := recover(); r != nil {
			rollbar.PanicRecovery(r)
		}
	}()
	//notify components that a new event is to be processed
	m.components.PluginEvent(editorEvt)

	start := time.Now()

	// Construct an event to send to the backend
	evt, err := m.eventProcessor.processEvent(editorEvt)
	if err != nil {
		if err != errDuplicate {
			log.Println("process event error:", err)
		}
		return
	}

	//notify components that an event was processed
	m.components.ProcessedEvent(evt, editorEvt)

	m.logf("!! event: %s full_text=%t diffs=%d ref=%s hash=%s sels=%+v %s:%s\n", evt.GetAction(),
		len(evt.GetText()) > 0, len(evt.GetDiffs()), evt.GetReferenceState(), evt.GetTextMD5(),
		evt.GetSelections(),
		evt.GetSource(), filepath.Base(evt.GetFilename()))

	/*
		These are left here as a reminder of fields to remove from the event object
		evt.Id = proto.Int64(atomic.AddInt64(&u.eventID, 1))
		evt.MachineId = proto.String(u.machine)
		evt.ClientVersion = proto.String(u.clientVersion)
		evt.LastEventLatency = proto.Int64(atomic.LoadInt64(&u.eventLatency))
		evt.LastBackendLatency = proto.Int64(atomic.LoadInt64(&u.backendLatency))
		evt.LastResponseSize = proto.Int64(atomic.LoadInt64(&u.responseSize))
	*/

	if m.fileProcessor == nil {
		return
	}

	var resp *eventResponse
	// this happens async to the http request, so set a manual timeout instead of using the http connection context
	err = kitectx.Background().WithTimeout(fileEventTimeout, func(ctx kitectx.Context) (err error) {
		resp, err = m.fileProcessor.handleEvent(ctx, evt)
		return
	})
	r := m.toRespRoot(evt, resp, err)

	m.logf("!! response: hash=%s %s:%s resend=%t\n", r.Hash, r.Editor, filepath.Base(r.Filename), r.ResendText)

	// Update what we know about the current content state
	m.eventProcessor.updateLatestResponse(editorEvt.Source, editorEvt.Filename, editorEvt.Text, evt.GetSelections(), r.ResendText)

	m.logf("!! delay: %s, total: %s", start.Sub(editorEvt.Timestamp), time.Since(editorEvt.Timestamp))

	select {
	case m.Responses <- r:
	default:
		m.logf("!! dropped response")
	}
}

const indexStatusResults = 5

func (m *Manager) toRespRoot(evt *event.Event, results *eventResponse, err error) *response.Root {
	var resp response.Root

	// Indicate which editor
	if event.IsEditor(evt) {
		resp.Editor = evt.GetSource()
	}

	resp.Hash = evt.GetTextMD5()
	for _, sel := range evt.GetSelections() {
		if sel.GetStart() == sel.GetEnd() {
			resp.Cursor = sel.GetStart()
		}
	}

	resp.Type = response.Passive
	resp.Filename = evt.GetFilename()

	// If there was an error, ask the client to resend text, otherwise,
	// send back editor completions, prefetched completions and autosearch ID
	if err != nil {
		resp.ResendText = true
	} else {
		resp.State = results.state
		resp.ResendText = results.resend
		for _, result := range results.results {
			switch r := result.(type) {
			case *response.EditorCompletions:
				resp.EditorCompletions = r
			case []*response.EditorCompletions:
				resp.PrefetchedCompletionsList = r
			case *response.Autosearch:
				resp.Results = append(resp.Results, result)
			case *response.LocalIndexPresent:
				resp.LocalIndexPresent = r.Present
			case *localcode.StatusResponse:
				resp.LocalIndexStatus = r
			case *response.ExpectCompletions:
				resp.ExpectCompletions = r.ExpectCompletions
			}
		}
	}

	return &resp
}
