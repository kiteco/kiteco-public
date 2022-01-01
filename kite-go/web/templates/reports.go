package templates

import (
	"fmt"
	"log"
	"net/http"
)

// ReportError reports an internal server error.
func ReportError(w http.ResponseWriter, s string, args ...interface{}) {
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	http.Error(w, s, http.StatusInternalServerError)
	log.Println(s)
}

// ReportNotFound reports a status not found error.
func ReportNotFound(w http.ResponseWriter, s string, args ...interface{}) {
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	http.Error(w, s, http.StatusNotFound)
	log.Println(s)
}

// ReportUnauthorized reports a status unauthorized error.
func ReportUnauthorized(w http.ResponseWriter, s string, args ...interface{}) {
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	http.Error(w, s, http.StatusUnauthorized)
	log.Println(s)
}

// ReportBadRequest reports a bad request error.
func ReportBadRequest(w http.ResponseWriter, s string, args ...interface{}) {
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	http.Error(w, s, http.StatusBadRequest)
	log.Println(s)
}
