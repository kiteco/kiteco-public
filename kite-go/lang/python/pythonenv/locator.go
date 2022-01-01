package pythonenv

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
	"github.com/kiteco/kiteco/kite-go/lang/python/pythontype"
	"github.com/pkg/errors"
)

const (
	// The separator character used in locators
	locatorSep = ";"
	// The regular expression for value locators
	valueLocatorRegexpStr = "^[0-9]+;[^;]*;[^;]*;[^;]*$"
	// The regular expression for symbol locators
	symbolLocatorRegexpStr = "^[0-9]+;[^;]*;[^;]*;[^;]*;[^;]+$"
	locatorRegexpStr       = "^[0-9]*;[^;]*;[^;]*;[^;]*(?:;[^;]+)?$"
)

var (
	valueLocatorRegexp  = regexp.MustCompile(valueLocatorRegexpStr)
	symbolLocatorRegexp = regexp.MustCompile(symbolLocatorRegexpStr)
	locatorRegexp       = regexp.MustCompile(locatorRegexpStr)
)

// Name returns the last component of the address:
// Name("a.b.c") -> "c"
func Name(v pythontype.Value) string {
	// if v.Address() is empty, this function will return ""
	return v.Address().Path.Last()
}

// LocatorForAddress returns the locator string
// for the value defined for the given user and machine
// in the provided file and with the provided dotted path.
func LocatorForAddress(addr pythontype.Address) string {
	if addr.Nil() {
		return ""
	}
	path := addr.Path.String()
	var user, machine, filename string
	if addr.File != "" {
		user = fmt.Sprintf("%d", addr.User)
		machine = addr.Machine
		filename = encodeFilename(addr.File)
	}
	return strings.Join([]string{user, machine, filename, path}, locatorSep)
}

// Locator returns a string that contains all the info that is necessary
// to find a value in a source tree. It contains the user, machine and
// file name the value is defined in and its full address path. If the
// value is from the global import graph, this function just returns the
// full address path.
func Locator(v pythontype.Value) string {
	if v == nil {
		return ""
	}
	return LocatorForAddress(v.Address())
}

// SymbolLocator returns the locator for a symbol which is an attribute of
// a value (the value is the namespace of the symbol).
func SymbolLocator(ns pythontype.Value, attr string) string {
	nsLoc := Locator(ns)
	if nsLoc == "" {
		return ""
	}
	return nsLoc + locatorSep + attr
}

// ModuleLocator returns a string that contains all the info that is
// necessary to find a value's module in a source tree. It contains the
// user, machine and file name the value is defined in and its module's
// full address path. If the value is from the global import graph, this
// function just returns the full address path.
func ModuleLocator(v pythontype.Value) string {
	addr := v.Address()
	if addr.Nil() {
		return ""
	}
	// Note that if v is a top-level External, then LocatorForAddress will return "" here
	// instead of ";;;" (the locator for ExternalRoot). This is acceptable, since ExternalRoot isn't really a module
	return LocatorForAddress(pythontype.Address{
		User:    addr.User,
		Machine: addr.Machine,
		File:    addr.File,
		Path:    addr.Path.Predecessor(),
	})
}

// IsLocator checks if loc is a valid _local_ value or symbol locator.
// It returns false for global locators (i.e. symbol graph locators)
func IsLocator(loc string) bool {
	return IsValueLocator(loc) || IsSymbolLocator(loc)
}

// IsValueLocator checks if loc is a valid _local_ value locator.
// It returns false for global locators (i.e. symbol graph locators)
func IsValueLocator(loc string) bool {
	b := []byte(loc)
	match := valueLocatorRegexp.Match(b)
	return match
}

// IsSymbolLocator checks if loc is a valid symbol locator. Symbol locators must have a non-empty attribute name.
// It returns false for global locators (i.e. symbol graph locators)
func IsSymbolLocator(loc string) bool {
	b := []byte(loc)
	match := symbolLocatorRegexp.Match(b)
	return match
}

// ParseValueLocator parses a value locator string into an address. It
// may return an error if the locator is not valid.
func ParseValueLocator(loc string) (addr pythontype.Address, err error) {
	if !IsValueLocator(loc) {
		err = fmt.Errorf("invalid locator")
		return
	}
	addr = pythontype.Address{}
	parts := strings.Split(loc, locatorSep)
	addr.User, _ = strconv.ParseInt(parts[0], 10, 64)
	addr.Machine = parts[1]
	addr.File = decodeFilename(strings.Join(parts[2:len(parts)-1], locatorSep))
	addr.Path = pythonimports.NewDottedPath(parts[len(parts)-1])
	return
}

// ParseSymbolLocator parses a symbol locator string into the address of its
// namespace and its attribute name. It may return an error if the locator
// is not valid.
func ParseSymbolLocator(loc string) (pythontype.Address, string, error) {
	if !IsSymbolLocator(loc) {
		return pythontype.Address{}, "", fmt.Errorf("invalid symbol locator")
	}

	parts := strings.Split(loc, locatorSep)

	var user int64
	if parts[0] != "" {
		if uid, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
			user = uid
		} else {
			return pythontype.Address{}, "", errors.Errorf("invalid locator string (cannot parse user ID): %s", loc)
		}
	}

	addr := pythontype.Address{
		User:    user,
		Machine: parts[1],
		File:    decodeFilename(strings.Join(parts[2:len(parts)-2], locatorSep)),
		Path:    pythonimports.NewDottedPath(parts[len(parts)-2]),
	}
	if addr.Nil() {
		addr.IsExternalRoot = true
	}
	return addr, parts[len(parts)-1], nil
}

// ParseLocator parses the locator string loc as either a value or symbol locator.
// If it is a symbol locator, the returned string will be non-empty (otherwise it is a value locator).
// If loc is not a valid locator string, an error is returned.
func ParseLocator(loc string) (pythontype.Address, string, error) {
	if !strings.Contains(loc, locatorSep) {
		if loc == "" {
			return pythontype.Address{}, "", errors.Errorf("invalid empty locator string")
		}

		// old locator format was just a dotted path
		return pythontype.Address{
			Path: pythonimports.NewDottedPath(loc),
		}, "", nil
	}

	if !locatorRegexp.Match([]byte(loc)) {
		return pythontype.Address{}, "", errors.Errorf("invalid locator string: %s", loc)
	}

	parts := strings.Split(loc, locatorSep)

	var user int64
	if parts[0] != "" {
		if uid, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
			user = uid
		} else {
			return pythontype.Address{}, "", errors.Errorf("invalid locator string (cannot parse user ID): %s", loc)
		}
	}

	addr := pythontype.Address{
		User:    user,
		Machine: parts[1],
		File:    decodeFilename(parts[2]),
		Path:    pythonimports.NewDottedPath(parts[3]),
	}
	if addr.Nil() {
		addr.IsExternalRoot = true
	}

	var attr string
	if len(parts) == 5 {
		attr = parts[4]
	}

	return addr, attr, nil
}

func encodeFilename(f string) string {
	return strings.Replace(strings.Replace(f, ":", "::", -1), "/", ":", -1)
}

func decodeFilename(s string) string {
	parts := strings.Split(s, "::")
	for i := range parts {
		parts[i] = strings.Replace(parts[i], ":", "/", -1)
	}
	return strings.Join(parts, ":")
}
