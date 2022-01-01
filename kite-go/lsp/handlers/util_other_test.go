// +build !windows

package handlers

import (
	"path/filepath"
	"testing"
)

func TestFilepathFromURI(t *testing.T) {
	testURI := "file:///home/"
	fp, err := filepathFromURI(testURI)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !filepath.IsAbs(fp) {
		t.Fatalf("Expected absolute file-path, got %s", fp)
	}
}
