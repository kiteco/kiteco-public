package lexicalproviders

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type overlapSizeTC struct {
	template   string
	completion string
	expected   int
}

func TestOverlapSize(t *testing.T) {
	tcs := []overlapSizeTC{
		{
			template:   "a$lpha be$ta",
			completion: "beta",
			expected:   0,
		},
		{
			template:   "al$pha albe$rta",
			completion: "alberta",
			expected:   2,
		},
	}
	for _, tc := range tcs {
		processed := processTemplate(t, tc.template)
		actual := OverlapSize(processed, tc.completion)
		require.Equal(t, tc.expected, actual)
	}
}
