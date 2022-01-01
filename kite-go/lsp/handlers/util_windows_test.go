package handlers

import (
	"path/filepath"
	"testing"
)

func TestFilepathFromURI(t *testing.T) {
	// Note: 3 slashes, drive letter is part of path
	testURI := "file:///c:/Users/"
	fp, err := filepathFromURI(testURI)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !filepath.IsAbs(fp) {
		t.Fatalf("Expected absolute file-path, got %s", fp)
	}
}
