package pythonranker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilter(t *testing.T) {
	matched := map[string]struct{}{
		"spiderman": struct{}{},
	}
	candidates := []string{"kiteman", "spiderman"}

	filtered := filter(candidates, matched)
	exp := []string{"kiteman"}

	assert.Equal(t, exp, filtered)
}
