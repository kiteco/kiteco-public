package localfiles

import (
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kiteco/kiteco/kite-golib/pipeline/sample"
)

var sitePackagesFilters = []filter{
	contains{"/site-packages/"},
	contains{"/dist-packages/"},
}

var libFilters = []filter{
	contains{"/anaconda/"},
	newRegex(`\/anaconda\d?\/`),
	contains{"/node_modules/"},
	contains{"/gyp/pylib/"},
	contains{"/venv/"},
	contains{"/bin/activate_this.py"},
	contains{"/distutils/"},
	contains{"/Python.framework/"},
	contains{"/python_stubs/"},
	contains{"/bower_components/"},
	newRegex(`\/python\/python\d\d\/`),
	newRegex(`\/lib/python[\d\.]+\/`),

	// temporaries created by setuptools
	newRegex(`/build/.?dist\..*/`),
	newRegex(`/build/scripts.*/`),
	contains{`/build/lib`},

	// these directories contain code for user's applications, usually not run directly by user
	newRegex(`/Library/Application Support/`),
	newRegex(`/Library/Cache/`),
	newRegex(`/Library/Caches/`),
	newRegex(`/Library/Android/`),
	newRegex(`/Library/Containers/`),
	newRegex(`/Library/Frameworks`),
	newRegex(`/Applications/`),
	newRegex(`/appdata/`),
}

// CategorizedFile ...
type CategorizedFile struct {
	sample.FileInfo
	IsLibrary      bool
	IsSitePackages bool
}

// CategorizeLocalFiles attempts to categorize a list of files into:
// - site-packages files: ones which are part of a user's site-packages installation
// - library files: ones which are part of a user's library, which user files may refer to, but which the user is
//     unlikely to have edited
// - user files: ones that a user has actively created/edited
func CategorizeLocalFiles(fis []sample.FileInfo) []CategorizedFile {
	var categorized []CategorizedFile
	for _, fi := range fis {
		c := CategorizedFile{
			FileInfo: fi,
		}
		for _, f := range sitePackagesFilters {
			if f.match(fi.Name) {
				c.IsSitePackages = true
				c.IsLibrary = true
				break
			}
		}
		for _, f := range libFilters {
			if f.match(fi.Name) {
				c.IsLibrary = true
				break
			}
		}
		categorized = append(categorized, c)
	}

	byDir := make(map[string][]int) // directory -> indices of file infos of files in that directory
	for i, fi := range categorized {
		dir := filepath.Dir(fi.Name)
		byDir[dir] = append(byDir[dir], i)
	}

	for _, idxs := range byDir {
		var earliest time.Time
		var latest time.Time
		for _, idx := range idxs {
			fi := categorized[idx]
			if fi.UpdatedAt.After(latest) {
				latest = fi.UpdatedAt
			}
			if earliest == (time.Time{}) || fi.UpdatedAt.Before(earliest) {
				earliest = fi.UpdatedAt
			}
		}

		// If the files were all updated at the same time, we assume it's a library directory. The assumption here is
		// that if the user was working on the files, they'd be modifying one file at a time and these changes would be
		// picked up by Kite with different timestamps.
		// If there's only one file, we can't apply this logic, so we assume it's a user file. (often library
		// directories have > 1 file, especially since __init__.py is common)
		if len(idxs) > 1 && latest.Sub(earliest) < 3*time.Second {
			for _, idx := range idxs {
				categorized[idx].IsLibrary = true
			}
		}
	}
	return categorized
}

type filter interface {
	match(path string) bool
}

type contains struct {
	substr string
}

func (c contains) match(path string) bool {
	return strings.Contains(path, c.substr)
}

type regex struct {
	pat *regexp.Regexp
}

func (r regex) match(path string) bool {
	return r.pat.MatchString(path)
}

func newRegex(pat string) regex {
	return regex{pat: regexp.MustCompile(pat)}
}
