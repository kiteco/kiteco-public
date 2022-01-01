package fileutil

import (
	"log"
	"net/url"
	"path"
	"strings"
)

// Join is a url.URL scheme-safe join method. This allows for joining of local
// files as well as URI's.
func Join(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	first := parts[0]
	u, err := url.Parse(parts[0])
	if err != nil {
		log.Fatal(err)
	}

	parts[0] = u.Path
	u.Path = path.Join(parts...)
	parts[0] = first // reset in case we were called with a persisted slice
	return u.String()
}

// Dir is a url.URL scheme-safe Dir method.
func Dir(dir string) string {
	if i := strings.Index(dir, "//"); i > -1 {
		base := dir[:i+2]
		parts := strings.Split(dir[i+2:], "/")
		if len(parts) < 2 {
			return base
		}
		parts = parts[:len(parts)-1]
		return Join(append([]string{base}, parts...)...)
	}
	return path.Dir(dir)
}
