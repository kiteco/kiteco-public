package stringindex

import (
	"encoding/json"
)

// OffsetEncoding represents a way of counting Unicode string offsets.
type OffsetEncoding uint8

const (
	// UTF8 counts Unicode string offsets by the number of bytes if encoded as UTF-8
	UTF8 OffsetEncoding = iota
	// UTF16 counts Unicode string offsets by the number of 16-bit code units if encoded as UTF-16.
	// Code points <= 0xFFFF are encoded as 1 unit, while the rest are encoded as 2.
	UTF16
	// UTF32 counts Unicode string offsets by the number of 32-bit code units if encoded as UTF-32.
	// All code points are encoded as a single unit, so this is equivalent to counting the number of code points.
	UTF32
)

// Native = UTF8 is the Go-native string encoding.
const Native OffsetEncoding = UTF8

// String implements fmt.Stringer
func (e OffsetEncoding) String() string {
	switch e {
	case UTF8:
		return "utf-8"
	case UTF16:
		return "utf-16"
	case UTF32:
		return "utf-32"
	default:
		panic("unhandled OffsetEncoding")
	}
}

// UnmarshalJSON implements json.Unmarshaler
func (e *OffsetEncoding) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	*e = GetOffsetEncoding(s)
	return nil
}

// MarshalJSON implements json.Marshaler
func (e OffsetEncoding) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

// GetOffsetEncoding maps s to the appropriate Encoding.
func GetOffsetEncoding(s string) OffsetEncoding {
	switch s {
	case "utf-8":
		return UTF8
	case "utf-16":
		return UTF16
	case "utf-32":
		return UTF32
	default:
		// for backwards compatibility purposes
		return UTF32
	}
}
