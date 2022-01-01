package editorapi

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"strings"

	"github.com/kiteco/kiteco/kite-go/lang"
)

// ID identifies a value or a symbol. Although some `ID`s may be of the form “a.b.c”,
// clients should not assume this to be true for all `ID`s.
// In particular, `ID`s for classes defined in the user's own codebase will have a different structure.
// Important:
//   * `ID`s should be treated as opaque strings
//   * `ID`s should always be URL escaped when they are included in a URL path
//   * `ID`s should not be URL escaped when they are included as a query parameter
type ID struct {
	id   string
	lang lang.Language
}

const idSeparator = ";"

// NewID constructs a new editorapi.ID from
// a language specific id string and the
// language the id string references.
func NewID(l lang.Language, langSpecific string) ID {
	return ID{
		id:   langSpecific,
		lang: l,
	}
}

// ParseID parses an `ID` from a string.
func ParseID(ids string) ID {
	var l lang.Language
	switch {
	case strings.HasPrefix(ids, lang.Python.Name()):
		l = lang.Python
	case strings.HasPrefix(ids, lang.JavaScript.Name()):
		l = lang.JavaScript
	default:
		return ID{}
	}
	return ID{
		id:   strings.TrimPrefix(ids, l.Name()+idSeparator),
		lang: l,
	}
}

// MarshalJSON determines how an `ID` is marshalled to json.
func (i ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// UnmarshalJSON determines how an `ID` is unmarshalled from json.
func (i *ID) UnmarshalJSON(b []byte) error {
	var ids string
	if err := json.Unmarshal(b, &ids); err != nil {
		return err
	}

	*i = ParseID(ids)
	return nil
}

// GobEncode implements gob.GobEncoder
func (i ID) GobEncode() ([]byte, error) {
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(i.String())
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// GobDecode implements gob.GobDecoder
func (i *ID) GobDecode(buf []byte) error {
	var ids string
	err := gob.NewDecoder(bytes.NewBuffer(buf)).Decode(&ids)
	if err != nil {
		return err
	}
	*i = ParseID(ids)
	return nil
}

// String implements the Stringer interface.
func (i ID) String() string {
	if i.id == "" {
		return ""
	}
	return i.lang.Name() + idSeparator + i.id
}

// LanguageSpecific returns the language specific portion of the ID.
func (i ID) LanguageSpecific() string {
	return i.id
}
