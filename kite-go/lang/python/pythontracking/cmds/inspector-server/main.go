package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/internal/inspectorapi"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
)

var (
	port = ":3032"
)

type app struct {
	store *store
}

func newApp() (*app, error) {
	store, err := newStore()
	if err != nil {
		return nil, fmt.Errorf("could not create store: %v", err)
	}
	return &app{store: store}, nil
}

func (a *app) handleEventListings(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	eventType := pythontracking.EventType(params.Get("event_type"))
	if eventType == "" {
		http.Error(w, fmt.Sprintf("need to specify event_type parameter"), http.StatusBadRequest)
		return
	}

	failure := params.Get("failure")

	if _, ok := eventSources[eventType]; !ok {
		http.Error(w, fmt.Sprintf("unrecognized event type: %s", eventType), http.StatusBadRequest)
		return
	}

	resp, err := a.store.getEventListings(eventType, failure)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting event listings: %v", err), http.StatusInternalServerError)
		return
	}

	buf, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshalling json: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (a *app) handleEventDetail(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	messageID, err := getMessageIDFromParams(params)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting message ID: %v", err), http.StatusInternalServerError)
		return
	}

	var exprCursor int64
	cursorParam := params.Get("cursor")
	if cursorParam != "" {
		exprCursor, err = strconv.ParseInt(cursorParam, 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("couldn't read cursor param (%s): %v", cursorParam, err),
				http.StatusBadRequest)
			return
		}
	}

	resp, err := a.getEventDetail(messageID, exprCursor)
	if err != nil {
		http.Error(w, fmt.Sprintf("error retrieving event detail: %v", err), http.StatusInternalServerError)
		return
	}

	buf, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshaling json: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (a *app) handleEventInfo(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	messageID, err := getMessageIDFromParams(params)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting message ID: %v", err), http.StatusInternalServerError)
		return
	}

	event, err := a.store.getEvent(messageID)
	if err != nil {
		http.Error(w, fmt.Sprintf("error retrieving event for ID %+v: %v", messageID, err),
			http.StatusInternalServerError)
		return
	}

	resp := inspectorapi.EventInfo{
		URI:       messageID.URI,
		MessageID: messageID.ID,
		Timestamp: event.metadata.Timestamp,
		Event:     *event.track,
	}

	buf, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshaling json: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func (a *app) getEventDetail(messageID analyze.MessageID, exprCursor int64) (*inspectorapi.EventDetail, error) {
	event, err := a.store.getEventWithContext(messageID)
	if err != nil {
		return nil, fmt.Errorf("could not get event for message ID %+v: %v", messageID, err)
	}

	cursor := int64(event.track.Offset)
	line, col := getCursorLineColumn(event.track.Buffer, cursor)

	res := &inspectorapi.EventDetail{
		Type:         event.track.Type,
		MessageID:    messageID.ID,
		URI:          messageID.URI,
		Timestamp:    event.metadata.Timestamp,
		UserID:       event.track.UserID,
		MachineID:    event.track.MachineID,
		Filename:     event.track.Filename,
		Failure:      event.track.Failure(),
		Buffer:       event.track.Buffer,
		Cursor:       cursor,
		LineNumber:   line,
		ColumnNumber: col,
		IndexedFiles: event.indexedFiles,
		IndexError:   event.track.ArtifactMeta.Error,
		Exprs:        getExprListings(event.ctx),
		ExprDetail:   getExprDetail(exprCursor, event.ctx),
	}

	if event.track.Type == pythontracking.ServerSignatureFailureEvent {
		var funcType string
		var function string
		if event.calleeResult.CallExpr != nil {
			funcExpr := event.calleeResult.CallExpr.Func
			funcType = reflect.TypeOf(funcExpr).String()
			function = event.track.Buffer[funcExpr.Begin():funcExpr.End()]
		}
		var calleeID string
		if event.calleeResult.Response != nil && event.calleeResult.Response.Callee != nil {
			calleeID = event.calleeResult.Response.Callee.ID.String()
		}
		res.Callee = &inspectorapi.CalleeDetail{
			ReproducedFailure: string(event.calleeResult.Failure),
			FuncType:          funcType,
			Function:          function,
			OutsideParens:     event.calleeResult.OutsideParens,
			CalleeResponse:    event.calleeResult.Response,
			CalleeID:          calleeID,
		}
	}

	return res, nil
}

func (a *app) handleListingGroupedEvents(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	eventType := pythontracking.EventType(params.Get("event_type"))
	if eventType == "" {
		http.Error(w, fmt.Sprintf("need to specify event_type parameter"), http.StatusBadRequest)
		return
	}

	failure := params.Get("failure")

	if _, ok := eventSources[eventType]; !ok {
		http.Error(w, fmt.Sprintf("unrecognized event type: %s", eventType), http.StatusBadRequest)
		return
	}

	resp, err := a.store.getGroupedEventListings(eventType, failure)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting event listings: %v", err), http.StatusInternalServerError)
		return
	}

	buf, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("error marshalling json: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func main() {
	app, err := newApp()
	if err != nil {
		log.Fatalln(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/events", app.handleEventListings)
	r.HandleFunc("/api/event", app.handleEventDetail)
	r.HandleFunc("/api/event/info", app.handleEventInfo)
	r.HandleFunc("/api/grouped-events", app.handleListingGroupedEvents)

	log.Printf("listening on %s", port)
	log.Fatal(http.ListenAndServe(port, r))
}

func getMessageIDFromParams(params url.Values) (analyze.MessageID, error) {
	URI := params.Get("uri")
	if URI == "" {
		return analyze.MessageID{}, fmt.Errorf("need to give 'uri' query-string argument")
	}
	ID := params.Get("message_id")
	if ID == "" {
		return analyze.MessageID{}, fmt.Errorf("need to give 'message_id' query-string argument")
	}
	return analyze.MessageID{URI: URI, ID: ID}, nil
}

// getCursorLineColumn returns the 1-indexed line and column numbers.
func getCursorLineColumn(src string, cursor int64) (line int, column int) {
	offset := int(cursor)
	lineMap := linenumber.NewMap([]byte(src))
	return lineMap.Line(offset) + 1, lineMap.Column(offset) + 1
}
