package pigeon

import "github.com/kiteco/kiteco/kite-go/lang/python/pythonparser/errors"

// expose the list of errors.
func (e errList) WrappedErrors() []error {
	return []error(e)
}

// expose the error reason.
func (e *parserError) Reason() errors.Reason {
	switch e.Inner {
	case errMaxExprCnt:
		return errors.MaxExpressionsLimit
	case errInvalidEncoding:
		return errors.InvalidEncoding
	default:
		return errors.Unknown
	}
}
