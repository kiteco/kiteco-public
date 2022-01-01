// Package errors defines error reason codes and extracts the reason
// associated with an error.
package errors

import "fmt"

// Reason identifies the reason why an error is returned by a call
// to parse. A Reason code is also a valid error value.
type Reason int

// List of error result reasons.
const (
	// Unknown error reason.
	Unknown Reason = iota
	// TooManyLines in input, a partial result is also returned.
	TooManyLines
	// MaxExpressionsLimit hit when parsing input, a nil result is returned.
	MaxExpressionsLimit
	// InvalidEncoding is when the input to parse has invalid utf-8 encoding.
	InvalidEncoding
)

var reasonString = map[Reason]string{
	Unknown:             "unknown",
	TooManyLines:        "too many lines",
	MaxExpressionsLimit: "max expressions limit",
	InvalidEncoding:     "invalid encoding",
}

// String representation of a Reason.
func (r Reason) String() string {
	if s, ok := reasonString[r]; ok {
		return s
	}
	return fmt.Sprintf("invalid reason (%d)", r)
}

// Error returns the string representation of the Reason
// as error message.
func (r Reason) Error() string {
	return r.String()
}

// Reason returns itself as the error Reason.
func (r Reason) Reason() Reason {
	return r
}

// ErrorReason returns the reason why an error was returned
// by a Parse call. It returns Unknown if this is not one of
// the known error reasons.
//
// If the error implements "Cause() error", a community-defined
// interface popularized by github.com/pkg/errors to retrieve
// the root cause of an error, it is called first and the returned
// error is used hereafter.
//
// If the error implements "WrappedErrors() []error", a community-defined
// interface popularized by github.com/hashicorp/go-multierror to
// get the list of errors contained in a "multi-error" type of error
// value, it is called and each returned error is checked for a reason.
//
// The Reason is extracted from errors that implement "Reason() Reason".
// The first non-unknown Reason is returned, otherwise it returns
// Unknown.
func ErrorReason(err error) Reason {
	// Work on the root cause error.
	if ce, ok := err.(interface {
		Cause() error
	}); ok {
		err = ce.Cause()
	}

	// Get the list of errors if this is a multi-error type of error.
	errs := []error{err}
	if me, ok := err.(interface {
		WrappedErrors() []error
	}); ok {
		errs = me.WrappedErrors()
	}

	// Return the first non-Unknown Reason.
	for _, e := range errs {
		if re, ok := e.(interface {
			Reason() Reason
		}); ok {
			if r := re.Reason(); r != Unknown {
				return r
			}
		}
	}
	return Unknown
}
