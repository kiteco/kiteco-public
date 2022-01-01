package localpath

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/kiteco/kiteco/kite-golib/lexicalv0"
)

// Absolute path
type Absolute string

// Relative path
type Relative string

// ErrPathNotAbsolute ...
var ErrPathNotAbsolute = errors.New("Path not absolute")

// NewAbsolute ...
func NewAbsolute(unverifiedPath string) (Absolute, error) {
	if !filepath.IsAbs(unverifiedPath) {
		return "", ErrPathNotAbsolute
	}
	return Absolute(unverifiedPath), nil
}

// HasSupportedExtension ...
func (a Absolute) HasSupportedExtension() bool {
	return Extension(filepath.Ext(string(a))).IsSupported()
}

// Join ...
func (a Absolute) Join(rels ...Relative) Absolute {
	parts := []string{string(a)}
	for _, rel := range rels {
		parts = append(parts, string(rel))
	}
	return Absolute(filepath.Join(parts...))
}

// RelativeTo ...
func (a Absolute) RelativeTo(base Absolute) (Relative, error) {
	rel, err := filepath.Rel(string(base), string(a))
	return Relative(rel), err
}

// Dir ...
func (a Absolute) Dir() Absolute {
	return Absolute(filepath.Dir(string(a)))
}

// Readdirnames ...
func (a Absolute) Readdirnames(n int) ([]Relative, error) {
	f, err := a.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	names, err := f.Readdirnames(n)
	if err != nil {
		return nil, err
	}
	var rels []Relative
	for _, name := range names {
		rels = append(rels, Relative(name))
	}
	return rels, nil
}

// Clean ...
func (a Absolute) Clean() Absolute {
	return Absolute(filepath.Clean(string(a)))
}

// Open ...
func (a Absolute) Open() (*os.File, error) {
	return os.Open(string(a))
}

// Lstat ...
func (a Absolute) Lstat() (os.FileInfo, error) {
	return os.Lstat(string(a))
}

// Extension ...
type Extension string

var supported = getSupportedExtensions()

func getSupportedExtensions() map[Extension]struct{} {
	extensions := make(map[Extension]struct{})
	for _, language := range lexicalv0.AllLangsGroup.Langs {
		for _, ext := range language.Extensions() {
			extensions[Extension("."+ext)] = struct{}{}
		}
	}
	return extensions
}

// IsSupported ...
func (e Extension) IsSupported() bool {
	_, ok := supported[e]
	return ok
}
