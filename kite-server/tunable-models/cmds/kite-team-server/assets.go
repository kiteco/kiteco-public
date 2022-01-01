package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

func (s *server) handleModelAsset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	asset := vars["asset"]
	lang := vars["lang"]

	var err error
	var active servableModel

	// Handle backwards compatability
	switch lang {
	case "python":
		active = servableModel{BasePath: filepath.Join(s.modelsDir, "py-large", "tfserving")}
	case "go":
		active = servableModel{BasePath: filepath.Join(s.modelsDir, "go-large", "tfserving")}
	case "javascript":
		active = servableModel{BasePath: filepath.Join(s.modelsDir, "js-large", "tfserving")}
	default:
		// if none of the above, default to the all-lang active model
		active, err = s.activeServableModel()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	fn := filepath.Join(active.BasePath, "1", "assets.extra", asset)
	f, err := os.Open(fn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	_, err = io.Copy(w, f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
