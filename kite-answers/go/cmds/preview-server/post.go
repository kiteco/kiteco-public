package main

import (
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kiteco/kiteco/kite-answers/go/execution"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythonresource"
)

func handleLocal(router *mux.Router, sandbox execution.Manager,
	resourceMgr pythonresource.Manager) error {
	h := &localHandler{
		sandbox:     sandbox,
		resourceMgr: resourceMgr,
	}
	router.Path("/live/render").HandlerFunc(h.handleRender)
	router.Path("/render").HandlerFunc(h.handleRender)
	return nil
}

type localHandler struct {
	sandbox     execution.Manager
	resourceMgr pythonresource.Manager
}

func (h *localHandler) handleRender(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// direct server access
		doRender(w, h.sandbox, h.resourceMgr, MustAsset("assets/PREVIEW.md"))
		return
	}

	src, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	doRender(w, h.sandbox, h.resourceMgr, src)
}
