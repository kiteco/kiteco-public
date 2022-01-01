package errors

import (
	"bytes"
	"fmt"
)

// Errors represents a list of errors; any non-nil Errors value represents a non-empty list of errors.
// This invariant is maintained so that the client may simply compare an Errors value will nil to check for the absence of errors.
type Errors interface {
	error
	// Slice returns a (non-empty) slice of underlying (non-nil) errors.
	Slice() []error
	// Len is always > 0. To check for existence of an error, simply compare the Errors with nil.
	Len() int

	sliceNoCopy() []error
	append(e error) Errors
}

type errorSlice []error

func (m errorSlice) append(e error) Errors {
	return errorSlice(append(m, e))
}

func (m errorSlice) sliceNoCopy() []error {
	return []error(m)
}

func (m errorSlice) Slice() []error {
	return append([]error(nil), m...)
}

func (m errorSlice) Len() int {
	return len(m)
}

func (m errorSlice) Error() string {
	var b bytes.Buffer
	for i, err := range m {
		if i > 0 {
			fmt.Fprint(&b, "\n")
		}
		fmt.Fprint(&b, err)
	}
	return b.String()
}

// Append appends the given (possibly nil) error to the given (possibly nil) Errors.
// If the error is nil, it returns the given Errors unchanged.
func Append(errs Errors, err error) Errors {
	if err == nil {
		return errs
	}
	if errs == nil {
		return errorSlice{err}
	}
	if err, _ := err.(Errors); err != nil {
		for _, err := range err.sliceNoCopy() {
			errs = errs.append(err)
		}
		return errs
	}
	return errs.append(err)
}

// Combine combines errors e & f into a single error
func Combine(e, f error) error {
	switch e := e.(type) {
	case nil:
		return f
	case Errors:
		// copy e to avoid mutating the backing array
		return Append(errorSlice(e.Slice()), f)
	default:
		switch f := f.(type) {
		case nil:
			return e
		case Errors:
			return Append(errorSlice{e}, f)
		default:
			return errorSlice{e, f}
		}
	}
}

// Defer is a helper method for deferring error-returning functions
func Defer(err *error, f func() error) {
	*err = Combine(*err, f())
}
