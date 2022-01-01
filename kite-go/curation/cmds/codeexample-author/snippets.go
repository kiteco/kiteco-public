package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/curation"
	"github.com/kiteco/kiteco/kite-go/lang"
	"github.com/kiteco/kiteco/kite-go/web/webutils"
)

var (
	curationErrorMap = webutils.StatusCodeMap{
		curation.ErrCodeDBError:         http.StatusInternalServerError,
		curation.ErrCodeSnippetNotExist: http.StatusNotFound,
		curation.ErrCodeSnippetExists:   http.StatusConflict,
		curation.ErrCodeCommentNotExist: http.StatusNotFound,
		curation.ErrCodeCommentExists:   http.StatusConflict,
		curation.ErrCodeBadSnippetID:    http.StatusBadRequest,
		curation.ErrCodeBadSnippetBody:  http.StatusBadRequest,
		curation.ErrCodeNeedEditLock:    http.StatusBadRequest,
	}
)

// --

// curatedSnippetHandlers contains HTTP handlers that wrap methods of the CuratedSnippetManager
type curatedSnippetHandlers struct {
	snippets  *curation.CuratedSnippetManager
	accesslog *accessManager
}

// newCuratedSnippetHandlers returns a new set of handlers wrapping the provided manager.
func newCuratedSnippetHandlers(snippets *curation.CuratedSnippetManager, accesslog *accessManager) *curatedSnippetHandlers {
	return &curatedSnippetHandlers{
		snippets:  snippets,
		accesslog: accesslog,
	}
}

// handleCreate wraps the manager's Create method.
func (c *curatedSnippetHandlers) handleCreate(w http.ResponseWriter, r *http.Request) {
	snippet, err := c.readSnippet(r)
	if err != nil {
		errf := webutils.ErrorCodef(curation.ErrCodeBadSnippetBody, "bad snippet: %s", err.Error())
		webutils.ErrorResponse(w, r, errf, curationErrorMap)
		return
	}

	userEmail := community.GetUser(r).Email

	lock, err := c.accesslog.acquireAccessLock(snippet.Language, snippet.Package, userEmail)
	if err != nil {
		webutils.ErrorResponse(w, r, fmt.Errorf("error checking access lock: %v", err), curationErrorMap)
		return
	}
	if lock.UserEmail != userEmail {
		errf := webutils.ErrorCodef(curation.ErrCodeNeedEditLock, "user '%s' does not have access lock (held by %s)", userEmail, lock.UserEmail)
		webutils.ErrorResponse(w, r, errf, curationErrorMap)
		return
	}

	err = c.snippets.Create(snippet)
	if err != nil {
		webutils.ErrorResponse(w, r, err, curationErrorMap)
		return
	}

	sendResponse(w, snippet)
}

// handleGet wraps the manager's GetByID method.
func (c *curatedSnippetHandlers) handleGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["snippetID"]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		errf := webutils.ErrorCodef(curation.ErrCodeBadSnippetID, "unable to parse id %s", idStr)
		webutils.ErrorResponse(w, r, errf, curationErrorMap)
		return
	}

	snippet, err := c.snippets.GetByID(id)
	if err != nil {
		webutils.ErrorResponse(w, r, err, curationErrorMap)
		return
	}

	sendResponse(w, snippet)
}

// handleUpdate wraps the manager's Update method.
func (c *curatedSnippetHandlers) handleUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["snippetID"]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		errf := webutils.ErrorCodef(curation.ErrCodeBadSnippetID, "unable to parse snippet id %s", idStr)
		webutils.ErrorResponse(w, r, errf, curationErrorMap)
		return
	}

	snippet, err := c.readSnippet(r)
	if err != nil {
		errf := webutils.ErrorCodef(curation.ErrCodeBadSnippetBody, "bad snippet: %s", err.Error())
		webutils.ErrorResponse(w, r, errf, curationErrorMap)
		return
	}

	if snippet.SnippetID != id {
		errf := webutils.ErrorCodef(curation.ErrCodeBadSnippetID, "mismatching snippet ids. path has %d, snippet has %d", id, snippet.SnippetID)
		webutils.ErrorResponse(w, r, errf, curationErrorMap)
		return
	}

	userEmail := community.GetUser(r).Email

	lock, err := c.accesslog.acquireAccessLock(snippet.Language, snippet.Package, userEmail)
	if err != nil {
		webutils.ErrorResponse(w, r, errors.New("failed to check access lock"), curationErrorMap)
		return
	}
	if lock.UserEmail != userEmail {
		errf := webutils.ErrorCodef(curation.ErrCodeNeedEditLock, "user '%s' does not have access lock (held by %s)", userEmail, lock.UserEmail)
		webutils.ErrorResponse(w, r, errf, curationErrorMap)
		return
	}

	err = c.snippets.Update(snippet)
	if err != nil {
		webutils.ErrorResponse(w, r, err, curationErrorMap)
		return
	}

	sendResponse(w, snippet)
}

// handleList wraps the manager's List method.
func (c *curatedSnippetHandlers) handleList(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pkg := vars["package"]
	language := vars["language"]

	var msg string
	switch {
	case pkg == "":
		msg = "expected package to be set in the path"
	case language == "":
		msg = "expected language to be set in the path"
	case lang.FromName(language) == lang.Unknown:
		msg = fmt.Sprintf("%s is an unrecognized language", language)
	}
	if msg != "" {
		http.Error(w, msg, http.StatusBadRequest)
	}

	snippets, err := c.snippets.List(language, pkg)
	if err != nil {
		webutils.ErrorResponse(w, r, err, curationErrorMap)
		return
	}

	sendResponse(w, &snippets)
}

func (c *curatedSnippetHandlers) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("statuses") == "" {
		webutils.ReportError(w, "TODO this should just return empty results")
		return
	}
	queryStatuses := strings.Split(r.FormValue("statuses"), ",")
	snippets, err := c.snippets.Query(curation.SnippetQuery{Statuses: queryStatuses})
	if err != nil {
		webutils.ErrorResponse(w, r, err, curationErrorMap)
		return
	}

	sendResponse(w, &snippets)
}

func (c *curatedSnippetHandlers) readSnippet(r *http.Request) (*curation.CuratedSnippet, error) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var snippet curation.CuratedSnippet
	err = json.Unmarshal(buf, &snippet)
	if err != nil {
		return nil, err
	}

	if snippet.Language == "" {
		return nil, errors.New("empty 'language' field in snippet")
	} else if snippet.Package == "" {
		return nil, errors.New("empty 'package' field in snippet")
	}

	user := community.GetUser(r)
	snippet.User = user.Email

	return &snippet, nil
}

// --

func (c *curatedSnippetHandlers) handleCreateComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["snippetID"]
	snippetID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		webutils.ErrorResponse(w, r, fmt.Errorf("unable to parse snippet ID %s", idStr), curationErrorMap)
		return
	}

	comment, err := c.readComment(r)
	if err != nil {
		webutils.ErrorResponse(w, r, err, curationErrorMap)
		return
	}

	comment.SnippetID = snippetID

	user := community.GetUser(r)
	comment.CreatedBy = user.Email

	if err := c.snippets.CreateComment(comment); err != nil {
		webutils.ErrorResponse(w, r, err, curationErrorMap)
		return
	}
	sendResponse(w, comment)
}

func (c *curatedSnippetHandlers) handleGetComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["commentID"]
	commentID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		webutils.ErrorResponse(w, r, fmt.Errorf("unable to parse comment ID %s", idStr), curationErrorMap)
		return
	}

	comment, err := c.snippets.GetByIDComment(commentID)
	if err != nil {
		webutils.ErrorResponse(w, r, err, curationErrorMap)
		return
	}

	sendResponse(w, comment)
}

func (c *curatedSnippetHandlers) handleUpdateComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["commentID"]
	commentID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, fmt.Errorf("unable to parse comment id %s", idStr).Error(), http.StatusBadRequest)
		return
	}

	comment, err := c.readComment(r)
	if err != nil {
		http.Error(w, fmt.Errorf("unable to parse comment from body").Error(), http.StatusBadRequest)
		return
	}

	if comment.ID != commentID {
		http.Error(w, fmt.Errorf("mismatching comment ids. path has %d, snippet has %d", commentID, comment.ID).Error(), http.StatusConflict)
		return
	}

	if comment.Dismissed > 1 { // dismissed
		http.Error(w, "cannot update a dismissed comment", http.StatusBadRequest)
		return
	}

	user := community.GetUser(r)

	// 1 is a constant expected by the backend for a dismiss operation
	// It's mutually agreed upon by the frontend.
	if comment.Dismissed == 1 {
		comment.DismissedBy = user.Email
	} else if comment.Dismissed == 0 {
		comment.DismissedBy = ""
		comment.ModifiedBy = user.Email
	}

	err = c.snippets.UpdateComment(comment)
	if err != nil {
		webutils.ErrorResponse(w, r, err, curationErrorMap)
		return
	}

	sendResponse(w, comment)
}

func (c *curatedSnippetHandlers) handleListComments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["snippetID"]
	snippetID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		webutils.ErrorResponse(w, r, fmt.Errorf("unable to parse snippet ID %s", idStr), curationErrorMap)
		return
	}

	comments, err := c.snippets.ListComments(snippetID)
	if err != nil {
		webutils.ErrorResponse(w, r, err, curationErrorMap)
		return
	}

	sendResponse(w, &comments)
}

func (c *curatedSnippetHandlers) readComment(r *http.Request) (*curation.Comment, error) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var comment curation.Comment
	err = json.Unmarshal(buf, &comment)
	if err != nil {
		return nil, err
	}

	return &comment, nil
}

// --

func sendResponse(w http.ResponseWriter, data interface{}) {
	buf, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(buf)
}
