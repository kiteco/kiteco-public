package pipeline

import "fmt"

// Sample represents a piece of data that is used as input/output for a Feed.
type Sample interface {
	SampleTag()
}

// Keyed wraps a sample and a string key
type Keyed struct {
	Key    string
	Sample Sample
}

// FlattenError returns an error if the sample stored in the Keyed sample is an error
func (k Keyed) FlattenError() Sample {
	if e, ok := k.Sample.(error); ok {
		return e.(Sample)
	}
	return k
}

// SampleTag implements pipeline.Sample
func (Keyed) SampleTag() {}

// NewError can be used by a Feed to communicate that no sample is being returned, along with a string
// representing a reason.
// A given feed should return errors with a finite set of reasons, since statistics are aggregated by reason.
func NewError(reason string) Sample {
	return sampleError{Reason: reason}
}

// WrapError works like NewError, but wraps an existing error.
func WrapError(reason string, err error) Sample {
	if s, ok := err.(sampleError); ok {
		// if we're already dealing with a sampleError, we can compose the reasons
		return sampleError{Reason: fmt.Sprintf("%s: %s", reason, s.Reason), Err: s.Err}
	}
	return sampleError{Reason: reason, Err: err}

}

// NewErrorAsError works like NewError, but returns an error interface.
func NewErrorAsError(reason string) error {
	return NewError(reason).(error)
}

// WrapErrorAsError works like WrapError, but returns an error interface.
func WrapErrorAsError(reason string, err error) error {
	return WrapError(reason, err).(error)
}

// CoerceError coerces an error to a Sample.
func CoerceError(err error) Sample {
	if err, ok := err.(sampleError); ok {
		return err
	}
	return WrapError("uncategorized", err)
}

// sampleError is used internally by the pipeline to communicate that a feed cannot return a sample for a reason
type sampleError struct {
	Reason string
	Err    error
}

// sampleError implements pipeline.Sample
func (sampleError) SampleTag() {}

// sampleError implements error
func (s sampleError) Error() string {
	if s.Err != nil {
		return fmt.Sprintf("%s: %v", s.Reason, s.Err)
	}

	return fmt.Sprintf("%v", s.Reason)
}
