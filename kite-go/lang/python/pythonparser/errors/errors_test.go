package errors

import (
	"fmt"
	"io"
	"testing"
)

func TestIdentity(t *testing.T) {
	for r := range reasonString {
		t.Run(r.String(), func(t *testing.T) {
			var err error = r
			got := ErrorReason(err)
			if r != got {
				t.Fatalf("want %s, got %s", r, got)
			}
			if err.Error() != r.String() {
				t.Fatalf("want message %q, got %q", r.String(), err.Error())
			}
		})
	}
}

type causeError struct {
	e error
}

func (e causeError) Cause() error  { return e.e }
func (e causeError) Error() string { return e.e.Error() }

type errList []error

func (e errList) WrappedErrors() []error { return []error(e) }
func (e errList) Error() string          { return "errors" }

func TestErrorReason(t *testing.T) {
	cases := []struct {
		in  error
		out Reason
	}{
		{nil, Unknown},
		{io.EOF, Unknown},
		{causeError{io.EOF}, Unknown},
		{errList{io.EOF}, Unknown},
		{causeError{errList{io.EOF}}, Unknown},

		{causeError{TooManyLines}, TooManyLines},
		{errList{TooManyLines}, TooManyLines},
		{causeError{errList{TooManyLines}}, TooManyLines},

		{errList{io.EOF, TooManyLines}, TooManyLines},
		{causeError{errList{io.EOF, TooManyLines}}, TooManyLines},
		{errList{io.EOF, TooManyLines, InvalidEncoding}, TooManyLines},
		{causeError{errList{TooManyLines, InvalidEncoding, io.EOF}}, TooManyLines},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%v", c.in), func(t *testing.T) {
			got := ErrorReason(c.in)
			if got != c.out {
				t.Fatalf("want %s, got %s", c.out, got)
			}
		})
	}
}
