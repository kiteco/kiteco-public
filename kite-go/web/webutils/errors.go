package webutils

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// StatusCodeMap is a mapping from app error codes to the appropriate http status code.
// Each app should define its own mapping, and pass it into `ErrorResponse` to handle
// errors correctly.
type StatusCodeMap map[int]int

var exposeErrorMessages bool

func init() {
	exposeErrorMessages = os.Getenv("KITE_EXPOSE_HTTP_ERRORS") != "OFF"
}

// --

// ErrorCode returns a new error object with the provided error code and message.
func ErrorCode(code int, message string) error {
	return ErrorCodef(code, message)
}

// ErrorCodef returns a new error object with the provided error code and formatted message.
func ErrorCodef(code int, message string, data ...interface{}) error {
	return errorf{
		code:    code,
		message: message,
		data:    data,
	}
}

// --

// errorf is an internal error type that wraps an application error code
// with a formatted message.
type errorf struct {
	code    int
	message string
	data    []interface{}
}

// Message formats the error message for an errorf
func (a errorf) formatMessage() string {
	return fmt.Sprintf(a.message, a.data...)
}

// Error implements the error interface
func (a errorf) Error() string {
	return fmt.Sprintf("error %d: %s", a.code, a.formatMessage())
}

// MarshalJSON implements the json.Unmarshaller interface
func (a errorf) MarshalJSON() ([]byte, error) {
	val := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{
		Code:    a.code,
		Message: a.formatMessage(),
	}

	return json.Marshal(val)
}

// HasCode checks to see whether err is an errorf with the provided error code
func HasCode(err error, code int) bool {
	errf, ok := err.(errorf)
	return ok && errf.code == code
}

// --

// ErrorResponse handles writing errors to a http.ResponseWriter. Takes in an error and a mapping of
// error codes to http status codes, and responds accordingly.
func ErrorResponse(w http.ResponseWriter, r *http.Request, err error, statusMap StatusCodeMap) {
	msg := "Internal server error"
	if exposeErrorMessages {
		msg = err.Error()
	}

	switch t := err.(type) {
	case errorf:
		status, exists := statusMap[t.code]
		if !exists {
			rollbar.RequestError(fmt.Errorf("could not find mapping for code %d", t.code), r)
			http.Error(w, msg, http.StatusInternalServerError)
			log.Println("responding with error:", err)
			return
		}
		buf, jsonErr := json.Marshal(t)
		if jsonErr != nil {
			rollbar.RequestError(jsonErr, r)
			http.Error(w, msg, http.StatusInternalServerError)
			log.Println("responding with error:", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(buf)
		log.Printf("app error %d: %s\n", t.code, t.formatMessage())
	default:
		rollbar.RequestError(t, r)
		http.Error(w, msg, http.StatusInternalServerError)
		log.Println("responding with error:", err)
	}
}

// ReportError reports a StatusInternalServerError
func ReportError(w http.ResponseWriter, s string, args ...interface{}) {
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	http.Error(w, s, http.StatusInternalServerError)
	log.Println(s)
}

// ReportNotFound reports a StatusNotFound error
func ReportNotFound(w http.ResponseWriter, s string, args ...interface{}) {
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	http.Error(w, s, http.StatusNotFound)
	log.Println(s)
}

// ReportUnauthorized reports a StatusUnauthorized error
func ReportUnauthorized(w http.ResponseWriter, s string, args ...interface{}) {
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	http.Error(w, s, http.StatusUnauthorized)
	log.Println(s)
}

// ReportBadRequest reports a StatusBadRequest error
func ReportBadRequest(w http.ResponseWriter, s string, args ...interface{}) {
	if len(args) > 0 {
		s = fmt.Sprintf(s, args...)
	}
	http.Error(w, s, http.StatusBadRequest)
	log.Println(s)
}
