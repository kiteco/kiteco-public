package normalize

import (
	"testing"

	"github.com/kiteco/kiteco/kite-golib/complete/data"
	"github.com/stretchr/testify/require"
)

type matchTC struct {
	completion                string
	beforeCursor              string
	afterCursor               string
	expectedMatchChars        int
	expectedMatchPlaceholders int
	expectedMatchIdentifiers  int
	expectedMatchKeywords     int
}

func TestMatch(t *testing.T) {
	tcs := []matchTC{
		matchTC{
			completion:               "numpy as np",
			beforeCursor:             "import nu",
			afterCursor:              "mpy as np",
			expectedMatchChars:       9,
			expectedMatchIdentifiers: 2,
			expectedMatchKeywords:    1,
		},
		matchTC{
			completion:   "numpy as np",
			beforeCursor: "import nu",
			afterCursor:  "mber",
		},
		matchTC{
			completion: "figsize",
			beforeCursor: `fig, ax = plt.subplots(
				fi`,
			afterCursor: `gsize=(4, 4)
				)`,
			expectedMatchChars:       5,
			expectedMatchIdentifiers: 1,
		},
		matchTC{
			completion: "figsize=(4, 5)",
			beforeCursor: `fig, ax = plt.subplots(
				fi`,
			afterCursor: `gsize=(4, 4)
				)`,
		},
		matchTC{
			completion: "word=\002[abc]\003, length=\002[]\003",
			beforeCursor: `count(
				wo`,
			afterCursor: `rd=MyWord, length=len(myWord)
			)`,
			expectedMatchChars:        12,
			expectedMatchPlaceholders: 2,
			expectedMatchIdentifiers:  3,
		},
		matchTC{
			completion: "word=\002[abc]\003, length=\002[]\003",
			beforeCursor: `count(
				wo`,
			afterCursor: `rd=MyWord, Length=len(myWord)
			)`,
		},
		matchTC{
			completion: "word=\002[abc]\003, length=\002[]\003",
			beforeCursor: `count(
				vocab, wo`,
			afterCursor: `rd=MyWord, length=len(myWord)
			)`,
			expectedMatchChars:        12,
			expectedMatchPlaceholders: 2,
			expectedMatchIdentifiers:  3,
		},
	}
	for _, tc := range tcs {
		snippet := data.BuildSnippet(tc.completion)
		matchMetrics, err := match(snippet, tc.beforeCursor, tc.afterCursor)
		require.NoError(t, err)
		require.Equal(t, tc.expectedMatchChars, matchMetrics.characters)
		require.Equal(t, tc.expectedMatchPlaceholders, matchMetrics.placeholders)
		require.Equal(t, tc.expectedMatchIdentifiers, matchMetrics.identifiers)
		require.Equal(t, tc.expectedMatchKeywords, matchMetrics.keywords)
	}
}
