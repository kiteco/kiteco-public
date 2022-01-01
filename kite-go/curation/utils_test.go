package curation

import (
	"testing"
)

func TestValidUTF(t *testing.T) {
	str := []string{"a\xc5z", "LO\x9eE\xf6D\f=\xa0(\x94\xed"}

	for _, s := range str {
		if string(ValidUTF([]byte(s))) == s {
			t.Errorf("Expected invalid utf but interpreted as valid")
		}
	}
}
