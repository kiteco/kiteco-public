package ignore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type mungePatternsTC struct {
	contents string
	expected []mungedPattern
}

func TestMungePatterns(t *testing.T) {
	tcs := []mungePatternsTC{
		mungePatternsTC{
			contents: "alpha\nbeta\ngamma",
			expected: []mungedPattern{
				mungedPattern{body: "alpha"},
				mungedPattern{body: "beta"},
				mungedPattern{body: "gamma"},
			},
		},
		mungePatternsTC{
			contents: "alpha\r\nbeta\r\ngamma",
			expected: []mungedPattern{
				mungedPattern{body: "alpha"},
				mungedPattern{body: "beta"},
				mungedPattern{body: "gamma"},
			},
		},
	}

	m := newMunger()
	for _, tc := range tcs {
		require.Equal(t, tc.expected, m.mungePatterns(tc.contents))
	}
}

type mungeLineTC struct {
	line       string
	expected   mungedPattern
	expectedOk bool
}

func TestMungeLine(t *testing.T) {
	tcs := []mungeLineTC{
		mungeLineTC{
			line:       "/alpha/beta    ",
			expected:   mungedPattern{body: "/alpha/beta"},
			expectedOk: true,
		},
		mungeLineTC{
			line:       `/alpha/beta\   `,
			expected:   mungedPattern{body: `/alpha/beta `},
			expectedOk: true,
		},
		mungeLineTC{
			line:       "/alpha/beta",
			expected:   mungedPattern{body: "/alpha/beta"},
			expectedOk: true,
		},
		mungeLineTC{
			line:       "",
			expected:   mungedPattern{body: ""},
			expectedOk: false,
		},
		mungeLineTC{
			line:       `#/alpha/beta`,
			expected:   mungedPattern{body: ""},
			expectedOk: false,
		},
		mungeLineTC{
			line:       `\#/alpha/beta`,
			expected:   mungedPattern{body: `#/alpha/beta`},
			expectedOk: true,
		},
		mungeLineTC{
			line:       "**/alpha",
			expected:   mungedPattern{body: "**/alpha"},
			expectedOk: true,
		},
		mungeLineTC{
			line:       "alpha/**/beta",
			expected:   mungedPattern{body: "alpha/**/beta"},
			expectedOk: true,
		},
		mungeLineTC{
			line:       "alpha/**",
			expected:   mungedPattern{body: "alpha/**"},
			expectedOk: true,
		},
		mungeLineTC{
			line:       "**alpha",
			expected:   mungedPattern{body: "*alpha"},
			expectedOk: true,
		},
		mungeLineTC{
			line:       "al*****pha",
			expected:   mungedPattern{body: "al*pha"},
			expectedOk: true,
		},
		mungeLineTC{
			line:       "alpha/*****/beta",
			expected:   mungedPattern{body: "alpha/**/beta"},
			expectedOk: true,
		},
		mungeLineTC{
			line:       "alpha[!01]",
			expected:   mungedPattern{body: "alpha[^01]"},
			expectedOk: true,
		},
		mungeLineTC{
			line: "!alpha",
			expected: mungedPattern{
				inverted: true,
				body:     "alpha",
			},
			expectedOk: true,
		},
	}

	m := newMunger()
	for _, tc := range tcs {
		munged, ok := m.mungeLine(tc.line)
		require.Equal(t, tc.expected, munged)
		require.Equal(t, tc.expectedOk, ok)
	}
}
