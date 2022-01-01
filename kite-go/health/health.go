package health

import (
	"encoding/json"
	"net/http"
)

const (
	// Endpoint is the endpoint statusd checks
	Endpoint = "/health"
)

// Status on an endpoint
type Status int

// Status of an endpoint can be None, OK, or Unreachable.
const (
	StatusNone Status = iota
	StatusOK
	StatusUnreachable
)

// String converts Status to printable string.
func (s Status) String() string {
	switch s {
	case StatusNone:
		return "N/A"
	case StatusOK:
		return "OK"
	case StatusUnreachable:
		return "Unreachable"
	}
	return ""
}

// Response for a status check
type Response struct {
	StatusCode Status `json:"status_code"`
	Message    string `json:"message"`
}

// Handler is the default handler for health, which returns
// an OK status. Override for custom behavior.
func Handler(w http.ResponseWriter, r *http.Request) {
	status := Response{
		StatusCode: StatusOK,
	}
	buf, err := json.Marshal(&status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}
