//go:generate bash -c "go-bindata $BINDATAFLAGS -o bindata.go templates/..."

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/highlight"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontracking/internal/inspectorapi"
	"github.com/kiteco/kiteco/kite-golib/linenumber"
	"github.com/kiteco/kiteco/kite-golib/segment/analyze"
	"github.com/kiteco/kiteco/kite-golib/templateset"
)

var (
	port             = ":3031"
	inspectorServer  = "localhost:3032"
	defaultEventType = pythontracking.ServerSignatureFailureEvent
)

type app struct {
	templates *templateset.Set
}

func (a app) handleGroupedEventListings(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	eventType := pythontracking.EventType(params.Get("event_type"))
	if eventType == "" {
		eventType = defaultEventType
	}

	failure := params.Get("failure")

	url := fmt.Sprintf("http://%s/api/grouped-events?event_type=%s&failure=%s",
		inspectorServer, url.PathEscape(string(eventType)), failure)
	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		apiError(w, resp)
		return
	}

	var listings inspectorapi.GroupedEventListings
	if err = json.NewDecoder(resp.Body).Decode(&listings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = a.templates.Render(w, "listings-grouped.html", map[string]interface{}{
		"Selector": template.HTML(a.renderSelector(listings.Metadata)),
		"Listings": listings,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a app) handleEventListings(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	eventType := pythontracking.EventType(params.Get("event_type"))
	if eventType == "" {
		eventType = defaultEventType
	}

	failure := params.Get("failure")

	url := fmt.Sprintf("http://%s/api/events?event_type=%s&failure=%s",
		inspectorServer, url.PathEscape(string(eventType)), url.PathEscape(failure))
	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		apiError(w, resp)
		return
	}

	var listings inspectorapi.EventListings
	if err = json.NewDecoder(resp.Body).Decode(&listings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = a.templates.Render(w, "listings.html", map[string]interface{}{
		"Selector": template.HTML(a.renderSelector(listings.Metadata)),
		"Listings": listings,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a app) renderSelector(meta inspectorapi.ListingsMetadata) string {
	var buf bytes.Buffer
	err := a.templates.Render(&buf, "partials/selector.html", meta)
	if err != nil {
		return err.Error()
	}
	return buf.String()
}

type indexedFileListing struct {
	Filename string
	Buffer   template.HTML
}

type property struct {
	Name  string
	Value interface{}
}

func (a app) handleEvent(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	messageID, err := getMessageIDFromParams(params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Couldn't get message ID: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := http.Get(fmt.Sprintf("http://%s/api/event/info?%s",
		inspectorServer, eventQueryString(messageID)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		apiError(w, resp)
		return
	}

	var info inspectorapi.EventInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	props := []property{
		{"Message ID", info.MessageID},
		{"URI", info.URI},
		{"Timestamp", info.Timestamp},
		{"User ID", info.Event.UserID},
		{"Machine ID", info.Event.MachineID},
		{"Filename", info.Event.Filename},
		{"Index size", len(info.Event.ArtifactMeta.FileHashes)},
		{"Index build error", info.Event.ArtifactMeta.Error},
		{"Failure", info.Event.Failure()},
	}

	lineMap := linenumber.NewMap([]byte(info.Event.Buffer))
	line := lineMap.Line(int(info.Event.Offset)) + 1

	code, err := highlight.Highlight(info.Event.Buffer, info.Event.Offset)

	if err != nil {
		http.Error(w, fmt.Sprintf("error highlighting code: %v", err), http.StatusInternalServerError)
		return
	}

	err = a.templates.Render(w, "event.html", map[string]interface{}{
		"Info":         info,
		"Properties":   props,
		"CursorAnchor": highlight.CursorAnchor,
		"Line":         line,
		"Buffer":       template.HTML(code),
	})
	if err != nil {
		log.Printf("error in rendering event: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a app) handleEventDetail(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	messageID, err := getMessageIDFromParams(params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Couldn't get message ID: %v", err), http.StatusBadRequest)
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

	resp, err := http.Get(fmt.Sprintf("http://%s/api/event?%s&cursor=%d",
		inspectorServer, eventQueryString(messageID), exprCursor))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		apiError(w, resp)
		return
	}

	var detail inspectorapi.EventDetail
	if err = json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	code := highlightExprs(messageID, &detail)

	var indexedFiles []indexedFileListing
	for filename, buffer := range detail.IndexedFiles {
		highlighted := highlightPlainCode(buffer)
		indexedFiles = append(indexedFiles, indexedFileListing{
			Filename: filename,
			Buffer:   template.HTML(highlighted),
		})
	}
	sort.Slice(indexedFiles, func(i, j int) bool {
		return indexedFiles[i].Filename < indexedFiles[j].Filename
	})

	err = a.templates.Render(w, "event-detail.html", map[string]interface{}{
		"Event":  detail,
		"Buffer": template.HTML(code),
		"ExprProps": []property{
			{"Name", detail.ExprDetail.Name},
			{"Expression Type", detail.ExprDetail.ExprType},
		},
		"IndexedFiles": indexedFiles,
		"Properties":   getEventProps(&detail),
		"LineID":       lineID(detail.LineNumber),
	})
	if err != nil {
		log.Printf("error in rendering event detail: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getEventProps(detail *inspectorapi.EventDetail) []property {
	props := []property{
		{"Message ID", detail.MessageID},
		{"URI", detail.URI},
		{"Timestamp", detail.Timestamp},
		{"User ID", detail.UserID},
		{"Machine ID", detail.MachineID},
		{"Index size", len(detail.IndexedFiles)},
		{"Index build error", detail.IndexError},
		{"Failure", detail.Failure},
	}

	callee := detail.Callee
	if callee != nil {
		calleeProps := []property{
			{"Reproduced failure", failureName(callee.ReproducedFailure)},
			{"Function", callee.Function},
			{"Callee ID", callee.CalleeID},
		}
		props = append(props, calleeProps...)
	}

	return props
}

func valToProps(val inspectorapi.ValueDetail) []property {
	return []property{
		{"Repr", val.Repr},
		{"Kind", val.Kind},
		{"Type", val.Type},
		{"Address", val.Address},
		{"Global type", val.GlobalType},
		{"Canonical name", val.CanonicalName},
	}
}

func apiError(w http.ResponseWriter, resp *http.Response) {
	var bodyStr string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		bodyStr = fmt.Sprintf("Could not read response body: %v", err)
	} else {
		bodyStr = string(body)
	}
	http.Error(w, fmt.Sprintf("API error: %d\n%s", resp.StatusCode, bodyStr), http.StatusInternalServerError)
}

func eventDetailURL(URI string, messageID string) string {
	return fmt.Sprintf("/event-detail?uri=%s&message_id=%s",
		url.PathEscape(URI), url.PathEscape(messageID))
}

func eventURL(URI string, messageID string) string {
	return fmt.Sprintf("/event?uri=%s&message_id=%s#cursor",
		url.PathEscape(URI), url.PathEscape(messageID))
}

func listingsURL(eventType pythontracking.EventType, failure string) string {
	return fmt.Sprintf("/?event_type=%s&failure=%s",
		url.PathEscape(string(eventType)), url.PathEscape(failure))
}
func groupedURL(eventType pythontracking.EventType, failure string) string {
	return fmt.Sprintf("/grouped?event_type=%s&failure=%s",
		url.PathEscape(string(eventType)), url.PathEscape(failure))
}

func failureName(failure string) string {
	if failure != "" {
		return failure
	}
	return "<none>"
}

func main() {
	app := app{}

	staticfs := &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo}
	app.templates = templateset.NewSet(staticfs, "templates", template.FuncMap{
		"listingsURL":    listingsURL,
		"groupedURL":     groupedURL,
		"eventDetailURL": eventDetailURL,
		"eventURL":       eventURL,
		"failureName":    failureName,
		"valToProps":     valToProps,
	})

	r := mux.NewRouter()
	r.HandleFunc("/", app.handleEventListings)
	r.HandleFunc("/event", app.handleEvent)
	r.HandleFunc("/event-detail", app.handleEventDetail)
	r.PathPrefix("/static/").Handler(http.FileServer(staticfs))
	r.HandleFunc("/grouped", app.handleGroupedEventListings)

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

func eventQueryString(messageID analyze.MessageID) string {
	return fmt.Sprintf("uri=%s&message_id=%s", url.PathEscape(messageID.URI), url.PathEscape(messageID.ID))
}
