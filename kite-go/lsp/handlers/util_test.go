package handlers

import (
	"testing"
)

func TestFilepathFromURI_NoPath(t *testing.T) {
	testURI := "file:"
	_, err := filepathFromURI(testURI)
	if err == nil {
		t.Fatalf("should reject URI %s", testURI)
	}
}

func TestFilepathFromURI_Opaque(t *testing.T) {
	testURI := "file:foo/bar"
	_, err := filepathFromURI(testURI)
	if err != nil {
		t.Fatalf(err.Error())
	}
}
