package autocorrect

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/lang/editorapi"
	"github.com/kiteco/kiteco/kite-golib/kitectx"
)

// Server for autocorrect.
type Server struct{}

// NewServer with the specified auth.
func NewServer() Server {
	return Server{}
}

// HandleAutocorrect handles suggesting corrections.
func (s Server) HandleAutocorrect(w http.ResponseWriter, r *http.Request) {
	uid := community.GetUser(r).ID
	mid := community.GetMachine(r)

	var req editorapi.AutocorrectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		breakdown.HitAndAdd("unable to decode request")
		http.Error(w, fmt.Sprintf("error unmarshalling request: %v", err), http.StatusInternalServerError)
		return
	}

	correcter, ok := getCorrecter(req.Language)
	if !ok {
		breakdown.HitAndAdd("no correcter")
		http.Error(w, fmt.Sprintf("unable to find correcter for `%s`", req.Language), http.StatusNotFound)
		return
	}

	var res Corrections
	err := kitectx.FromContext(r.Context(), func(ctx kitectx.Context) error {
		var err error
		res, err = correcter.Correct(ctx, uid, mid, req)
		return err
	})
	if err != nil {
		breakdown.HitAndAdd("correction error")
		http.Error(w, fmt.Sprintf("corrector error: %v", err), http.StatusInternalServerError)
		return
	}

	diffs := diffs(req.Buffer, res.NewBuffer)
	if len(diffs) == 0 {
		breakdown.HitAndAdd("no corrections")
	} else {
		breakdown.HitAndAdd("sent corrections")
	}

	resp := editorapi.AutocorrectResponse{
		Filename:            req.Filename,
		RequestedBufferHash: hash(req.Buffer),
		NewBuffer:           res.NewBuffer,
		Diffs:               diffs,
		Version:             correcter.Version(),
	}

	s.write(w, resp)
}

// HandleModelInfo handles information for a specified model.
func (s Server) HandleModelInfo(w http.ResponseWriter, r *http.Request) {
	var req editorapi.AutocorrectModelInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("error unmarshalling request: %v", err), http.StatusInternalServerError)
		return
	}

	correcter, ok := getCorrecter(req.Language)
	if !ok {
		http.Error(w, fmt.Sprintf("unable to find correcter for `%s`", req.Language), http.StatusNotFound)
		return
	}

	resp, err := correcter.ModelInfo(req.Version)
	if err != nil {
		http.Error(w, fmt.Sprintf("error getting model info for language %s model %d: %v", req.Language, req.Version, err), http.StatusInternalServerError)
		return
	}

	s.write(w, resp)
}

// HandleOnSaveHook handles the editor on save hook.
// Response Codes
//   * 200 -- Success
//   * 500 -- Internal error (usually JSON related)
func (s Server) HandleOnSaveHook(w http.ResponseWriter, r *http.Request) {
	var uid int64
	var mid string
	if user := community.GetUser(r); user != nil {
		uid = user.ID
		mid = community.GetMachine(r)
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("unable to unmarshal request: %v", err), http.StatusInternalServerError)
		return
	}

	send(uid, mid, req)

	w.WriteHeader(http.StatusOK)
}

// HandleOnRunHook handles the editor on console run hook.
// Response Codes
//   * 200 -- Success
//   * 500 -- Internal error (usually JSON related)
func (s Server) HandleOnRunHook(w http.ResponseWriter, r *http.Request) {
	var uid int64
	var mid string
	if user := community.GetUser(r); user != nil {
		uid = user.ID
		mid = community.GetMachine(r)
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("unable to unmarshal request: %v", err), http.StatusInternalServerError)
		return
	}

	send(uid, mid, req)

	w.WriteHeader(http.StatusOK)
}

// HandleMetrics handles metrics from the editors.
// Response Codes
//   * 200 -- Success
//   * 500 -- Internal error (usually JSON related)
func (s Server) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	uid := community.GetUser(r).ID
	mid := community.GetMachine(r)

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("unable to unmarshal request: %v", err), http.StatusInternalServerError)
		return
	}

	send(uid, mid, req)

	w.WriteHeader(http.StatusOK)
}

// HandleFeedback handles metrics from the editors.
// Response Codes
//   * 200 -- Success
//   * 500 -- Internal error (usually JSON related)
func (s Server) HandleFeedback(w http.ResponseWriter, r *http.Request) {
	uid := community.GetUser(r).ID
	mid := community.GetMachine(r)

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("unable to unmarshal request: %v", err), http.StatusInternalServerError)
		return
	}

	send(uid, mid, req)

	w.WriteHeader(http.StatusOK)
}

//
// --
//

func (s *Server) write(w http.ResponseWriter, resp interface{}) {
	buf, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func hash(buf string) string {
	h := md5.Sum([]byte(buf))
	return fmt.Sprintf("%x", h)
}
