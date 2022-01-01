package systeminfo

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"
)

// handleVersion returns the current client version
func (m *Manager) handleVersion(w http.ResponseWriter, r *http.Request) {
	type Response struct {
		Version string `json:"version"`
	}

	response := Response{
		Version: m.clientVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(w)
	if err := e.Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleGetSystemInfo returns basic system information
func (m *Manager) handleGetSystemInfo(w http.ResponseWriter, r *http.Request) {
	buf, err := json.Marshal(getSystem())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

type system struct {
	OperatingSystem string `json:"os"`
	PathSeparator   string `json:"path_separator"`
}

func getSystem() *system {
	return &system{
		OperatingSystem: runtime.GOOS,
		PathSeparator:   string(os.PathSeparator),
	}
}
