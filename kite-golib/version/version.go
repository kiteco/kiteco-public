package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var versionMatcher = regexp.MustCompile(`\d+(\.\d+)*(-[a-zA-Z0-9_.-]+)?`)

// Info represents a generic version string, it expects that a dot "." is used as delimiter
type Info struct {
	parts  []string
	Suffix string
}

func (v Info) String() string {
	return strings.Join(v.parts, ".") + v.Suffix
}

// Major returns the 1st part of the version or 0 if not found
func (v Info) Major() int {
	return v.parsed(0)
}

// Minor returns the 2nd part of the version or 0 if not found
func (v Info) Minor() int {
	return v.parsed(1)
}

// Patch returns the 3rd part of the version or 0 if not found
func (v Info) Patch() int {
	return v.parsed(2)
}

func (v Info) parsed(i int) int {
	if len(v.parts) <= i {
		return 0
	}

	i, err := strconv.Atoi(v.parts[i])
	if err != nil {
		return 0
	}

	return i
}

// LargerThanOrEqualTo returns true if the current version is larger than or equal to the version passed as parameter
func (v Info) LargerThanOrEqualTo(b Info) bool {
	return v.largerThan(b, true)
}

// LargerThan returns true if the current version is larger than to the version passed as parameter
func (v Info) LargerThan(b Info) bool {
	return v.largerThan(b, false)
}

func (v Info) largerThan(b Info, acceptEqual bool) bool {
	max := len(v.parts)
	if len(b.parts) > max {
		max = len(b.parts)
	}

	for i := 0; i < max; i++ {
		a := v.parsed(i)
		b := b.parsed(i)

		if a > b {
			return true
		} else if a < b {
			return false
		}
	}

	// still equal, now lets check suffixes
	switch {
	case v.Suffix != "" && b.Suffix != "":
		return v.Suffix > b.Suffix
	case v.Suffix != "":
		return true
	case b.Suffix != "":
		return false
	}

	return acceptEqual
}

// Parse parses a string of the form "a.b.c-suffix" where a,b and c are positive integer literals
func Parse(version string) (Info, error) {
	matched := versionMatcher.FindString(version)
	if matched != version {
		return Info{}, fmt.Errorf("%s is not a valid version number", version)
	}

	index := strings.Index(version, "-")
	suffix := ""
	if index >= 0 {
		suffix = version[index:]
		version = version[:index]
	}

	parts := strings.Split(version, ".")

	return Info{parts: parts, Suffix: suffix}, nil
}

// MustParse parses a string of the form "a.b.c-suffix" where a,b and c are positive integer literals
// It panics when the parsing fails
func MustParse(version string) Info {
	v, err := Parse(version)

	if err != nil {
		panic(`version: unable to parse ` + version)
	}

	return v
}

// Infos represents a slice of Version structs
// that are sortable.
type Infos []Info

// Len implements part of sort.Interface.
func (v Infos) Len() int { return len(v) }

// Swap implements part of sort.Interface.
func (v Infos) Swap(i, j int) { v[i], v[j] = v[j], v[i] }

// Less implements part of sort.Interface.
func (v Infos) Less(i, j int) bool {
	return v[j].LargerThanOrEqualTo(v[i])
}
