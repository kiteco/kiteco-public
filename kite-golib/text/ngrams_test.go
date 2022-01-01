package text

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNGrams(t *testing.T) {
	toks := strings.Split("how to check if a script", " ")
	expected1 := [][]string{[]string{"how"}, []string{"to"}, []string{"check"}, []string{"if"}, []string{"a"}, []string{"script"}}
	expected2 := [][]string{[]string{"how", "to"}, []string{"to", "check"}, []string{"check", "if"}, []string{"if", "a"}, []string{"a", "script"}}
	expected3 := [][]string{[]string{"how", "to", "check"}, []string{"to", "check", "if"}, []string{"check", "if", "a"}, []string{"if", "a", "script"}}
	expected := [][][]string{expected1, expected2, expected3}
	ns := []int{1, 2, 3}
	for i, n := range ns {
		actual, err := NGrams(n, toks)
		assert.Nil(t, err, "err should be nil")
		assert.Equal(t, expected[i], actual)
	}
	actual, err := NGrams(0, toks)
	assert.NotNil(t, err, "should be non nil error for n = 0")
	assert.Nil(t, actual, "should be nil ngrams for n = 0")

	actual, err = NGrams(1, nil)
	assert.NotNil(t, err, "should be non nil error for toks = nil")
	assert.Nil(t, actual, "should be nil ngrams for toks = nil")

}
