package errors

import (
	"fmt"

	"github.com/pkg/errors"
)

// Errorf is re-exported from fmt
var Errorf = fmt.Errorf

// New is an alias to Errorf
var New = Errorf

// ErrorfWithStack is Errorf re-exported from github.com/pkg/errors
var ErrorfWithStack = errors.Errorf

// WrapfOrNil is WithMessagef re-exported from github.com/pkg/errors
func WrapfOrNil(err error, format string, args ...interface{}) error {
	// do this check here to avoid the excessive format below
	// even though WithMessage does it
	if err == nil {
		return nil
	}
	return errors.WithMessage(err, fmt.Sprintf(format, args...))
}

// Wrapf is WrapfOrNil if err != nil, and Errorf otherwise: it never returns nil
func Wrapf(err error, format string, args ...interface{}) error {
	// reimplement until we update our vendored github.com/pkg/errors
	if err == nil {
		return Errorf(format, args...)
	}
	return WrapfOrNil(err, format, args...)
}

// WrapfWithStack is Wrapf re-exported from github.com/pkg/errors
var WrapfWithStack = errors.Wrapf

// WithStack is re-exported from github.com/pkg/errors
var WithStack = errors.WithStack

// Cause is re-exported from github.com/pkg/errors
var Cause = errors.Cause
