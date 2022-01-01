package re

import "regexp"

var (
	// FuncRegexp is a regex for recognizing functions
	FuncRegexp = regexp.MustCompile(`([a-zA-Z0-9_]+)\([^\(\)]*\)`)
	// SelRegexp is a regex for recognizing selectors
	SelRegexp = regexp.MustCompile(`(?P<x>[a-zA-Z0-9._]*)\.(?P<sel>[a-zA-Z0-9_]*)`)
)
