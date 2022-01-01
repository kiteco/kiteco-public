package pythontype

import (
	"path"

	"github.com/kiteco/kiteco/kite-go/lang/python/pythonimports"
)

// Address is the path to a value in the import graph
type Address struct {
	// For values from the local index, user identifies the user, machine
	// identifies the machine, and filename is the path to the file in
	// which the object was defined. For values from the global graph these
	// are all empty.
	User    int64
	Machine string
	File    string
	// Path is a sequence of one or more python identifiers.
	Path           pythonimports.DottedPath
	IsExternalRoot bool // all other fields should be empty if this is true
}

// String converts an address to a string
func (a Address) String() string {
	if a.File == "" {
		return a.Path.String()
	}
	return path.Base(a.File) + ":" + a.Path.String()
}

// Nil checks if a should be considered a "nil" address.
// Note that this is distinct from a valid empty address representing ExternalRoot
func (a Address) Nil() bool {
	return !a.IsExternalRoot && a.File == "" && a.Path.Empty() && a.Machine == "" && a.User == 0
}

// ShortName converts an address to a human readable name; it returns "" for "nil" addresses
func (a Address) ShortName() string {
	if a.File == "" {
		return a.Path.Last()
	}
	return path.Base(a.File) + ":" + a.Path.Last()
}

// WithTail returns a copy of this address with one or more components appended
func (a Address) WithTail(components ...string) Address {
	return Address{User: a.User, Machine: a.Machine, File: a.File, Path: a.Path.WithTail(components...)}
}

// Equals returns true in the case that the two adresses are the same
func (a Address) Equals(b Address) bool {
	return a.User == b.User && a.Machine == b.Machine && a.File == b.File && a.Path.Hash == b.Path.Hash && a.IsExternalRoot == b.IsExternalRoot
}

// SplitAddress creates an address by splitting a string at each period
func SplitAddress(s string) Address {
	return Address{Path: pythonimports.NewDottedPath(s)}
}
