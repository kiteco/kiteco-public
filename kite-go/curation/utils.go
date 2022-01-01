package curation

import (
	"unicode/utf8"
)

// ValidUTF removes runes that are not valid UTF-8 characters,
// and returns a clean UTF-8 compatible byte array.
func ValidUTF(data []byte) []byte {
	if len(data) == 0 {
		return []byte{}
	}
	s := string(data)
	if !utf8.ValidString(s) {
		v := make([]rune, 0, len(s))
		for i, r := range s {
			if r == utf8.RuneError {
				_, size := utf8.DecodeRuneInString(s[i:])
				if size == 1 {
					continue
				}
			}
			v = append(v, r)
		}
		s = string(v)
	}
	return []byte(s)
}
