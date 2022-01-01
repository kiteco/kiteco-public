package web

import (
	"fmt"
	"log"

	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/rollbar"
)

// VerboseLog allows to control the level of logging of web.ErrorData
var VerboseLog = true

// ErrorData is used to report error in a web context
type ErrorData struct {
	// Debug message for backend
	Debug string
	// Msg for front end
	Msg string
	// Code for http error
	Code int
}

// Prepend prepends a message to the current error debug info
func (e ErrorData) Prepend(s string) ErrorData {
	return ErrorData{
		Debug: fmt.Sprintf("%s %s", s, e.Debug),
		Msg:   e.Msg,
		Code:  e.Code,
	}
}

// RollbarError sends the debug info of this error to rollbar
func (e ErrorData) RollbarError() ErrorData {
	if VerboseLog {
		log.Println(e.Debug)
	}
	rollbar.Error(errors.Errorf("%s", e.Debug))
	return e
}

// RollbarCritical sends the debug info of this error to rollbar in critical mode
func (e ErrorData) RollbarCritical() ErrorData {
	if VerboseLog {
		log.Println(e.Debug)
	}
	rollbar.Critical(errors.Errorf("%s", e.Debug))
	return e
}

// AnyErr returns true if this error contains any kind of error (equivalent to err != nil for standard errors)
func (e ErrorData) AnyErr() bool {
	return e.Debug != "" || e.HTTPError()
}

// HTTPError returns true if this error should be reported back to user through http response
func (e ErrorData) HTTPError() bool {
	return e.Code > 0 && e.Msg != ""
}

// Error returns the debug string associated to this error
func (e ErrorData) Error() string {
	return e.Debug
}
