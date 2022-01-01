package ignore

import (
	"path"
	"testing"

	"github.com/kiteco/kiteco/kite-go/navigation/git"
	"github.com/stretchr/testify/require"
)

type patternIgnoreTC struct {
	p             pattern
	pathname      git.File
	isDir         bool
	expected      bool
	expectedError error
}

func TestPatternIgnore(t *testing.T) {
	tcs := []patternIgnoreTC{
		patternIgnoreTC{
			p: simplePattern{
				base:     true,
				inverted: true,
				sequence: []string{"alpha"},
			},
			pathname: "alpha",
			expected: false,
		},
		patternIgnoreTC{
			p: simplePattern{
				base:     true,
				sequence: []string{"*.beta"},
			},
			pathname: "gamma/delta/alpha.beta",
			expected: true,
		},
		patternIgnoreTC{
			p: simplePattern{
				base:     true,
				dirOnly:  true,
				sequence: []string{"*.beta"},
			},
			pathname: "gamma/delta/alpha.beta",
			isDir:    true,
			expected: true,
		},
		patternIgnoreTC{
			p: simplePattern{
				base:     true,
				dirOnly:  true,
				sequence: []string{"*.beta"},
			},
			pathname: "gamma/delta/alpha.beta",
			expected: false,
		},
		patternIgnoreTC{
			p: simplePattern{
				base:     true,
				sequence: []string{"*.beta"},
			},
			pathname: "gamma/delta/alphabeta",
			expected: false,
		},
		patternIgnoreTC{
			p: simplePattern{
				base:     true,
				sequence: []string{"*.beta"},
			},
			pathname: "gamma/alpha.beta/delta",
			expected: false,
		},
		patternIgnoreTC{
			p: simplePattern{
				inverted: true,
				base:     true,
				sequence: []string{"*.beta"},
			},
			pathname: "gamma/delta/alpha.beta",
			expected: false,
		},
		patternIgnoreTC{
			p: simplePattern{
				inverted: true,
				base:     true,
				sequence: []string{"*.beta"},
			},
			pathname: "gamma/delta/alphabeta",
			expected: true,
		},
		patternIgnoreTC{
			p: simplePattern{
				sequence: []string{"*", "beta", "g?mm?"},
			},
			pathname: "/alpha/beta/gamma",
			expected: true,
		},
		patternIgnoreTC{
			p: simplePattern{
				sequence: []string{"*", "beta"},
			},
			pathname: "/alpha/beta/gamma",
			expected: false,
		},
		patternIgnoreTC{
			p: simplePattern{
				sequence: []string{"*", "beta", "g?mm?"},
			},
			pathname: "/alpha/beta",
			expected: false,
		},
		patternIgnoreTC{
			p: simplePattern{
				sequence: []string{"*", "beta", "g?mm?"},
			},
			pathname: "/alpha/beta/Gamma",
			expected: false,
		},
		patternIgnoreTC{
			p: simplePattern{
				base:     true,
				sequence: []string{"bet[a-"},
			},
			pathname:      "/alpha/beta",
			expected:      false,
			expectedError: path.ErrBadPattern,
		},
		patternIgnoreTC{
			p: doubleStarPattern{
				leftSequence:  []string{"*", "beta", "g?mm?"},
				rightSequence: []string{"delta", "*", "*.phi"},
			},
			pathname: "/alpha/beta/gamma/delta/epsilon/eta.phi",
			expected: true,
		},
		patternIgnoreTC{
			p: doubleStarPattern{
				leftSequence:  []string{"*", "beta", "g?mm?"},
				rightSequence: []string{"delta", "*", "*.phi"},
			},
			pathname: "/alpha/beta/gamma/rho/sigma/tau/delta/epsilon/eta.phi",
			expected: true,
		},
		patternIgnoreTC{
			p: doubleStarPattern{
				leftSequence:  []string{"*", "beta", "g?mm?"},
				rightSequence: []string{"delta", "*", "*.phi"},
			},
			pathname: "/alpha/beta/gamma/rho/sigma/tau/epsilon/eta.phi",
			expected: false,
		},
		patternIgnoreTC{
			p: doubleStarPattern{
				leftSequence: []string{"alpha"},
				middleSequences: [][]string{
					[]string{"beta", "gamma"},
					[]string{"delta", "epsilon"},
				},
				rightSequence: []string{"phi"},
			},
			pathname: "/alpha/X/Y/beta/gamma/U/V/W/delta/epsilon/Z/phi",
			expected: true,
		},
		patternIgnoreTC{
			p: doubleStarPattern{
				leftSequence: []string{"alpha"},
				middleSequences: [][]string{
					[]string{"beta", "gamma"},
					[]string{"delta", "epsilon"},
				},
				rightSequence: []string{"phi"},
			},
			pathname: "/alpha/X/Y/beta/U/V/W/delta/epsilon/Z/phi",
			expected: false,
		},
	}

	for _, tc := range tcs {
		ignore, err := tc.p.ignore(tc.pathname, tc.isDir)
		require.Equal(t, tc.expectedError, err, tc.pathname)
		require.Equal(t, tc.expected, ignore, tc.pathname)
	}
}

type parsePatternTC struct {
	line     mungedPattern
	expected pattern
}

func TestParsePattern(t *testing.T) {
	tcs := []parsePatternTC{
		parsePatternTC{
			line: mungedPattern{body: "*.alpha"},
			expected: simplePattern{
				base:     true,
				sequence: []string{"*.alpha"},
			},
		},
		parsePatternTC{
			line: mungedPattern{body: "/alpha/beta"},
			expected: simplePattern{
				sequence: []string{"alpha", "beta"},
			},
		},
		parsePatternTC{
			line: mungedPattern{body: "/alpha/beta/"},
			expected: simplePattern{
				dirOnly:  true,
				sequence: []string{"alpha", "beta"},
			},
		},
		parsePatternTC{
			line: mungedPattern{body: "alpha/**/beta"},
			expected: doubleStarPattern{
				leftSequence:  []string{"alpha"},
				rightSequence: []string{"beta"},
				totalLength:   2,
			},
		},
		parsePatternTC{
			line: mungedPattern{body: "/alpha/beta/**"},
			expected: doubleStarPattern{
				leftSequence: []string{"alpha", "beta"},
				totalLength:  2,
			},
		},
		parsePatternTC{
			line: mungedPattern{body: "**/alpha/beta"},
			expected: doubleStarPattern{
				rightSequence: []string{"alpha", "beta"},
				totalLength:   2,
			},
		},
		parsePatternTC{
			line: mungedPattern{body: "alpha/**/beta/gamma/**/delta/epsilon/"},
			expected: doubleStarPattern{
				dirOnly:      true,
				leftSequence: []string{"alpha"},
				middleSequences: [][]string{
					[]string{"beta", "gamma"},
				},
				rightSequence: []string{"delta", "epsilon"},
				totalLength:   5,
			},
		},
		parsePatternTC{
			line: mungedPattern{body: "**/alpha/**/beta/gamma/**/delta/epsilon/phi**"},
			expected: doubleStarPattern{
				middleSequences: [][]string{
					[]string{"alpha"},
					[]string{"beta", "gamma"},
					[]string{"delta", "epsilon", "phi"},
				},
				totalLength: 6,
			},
		},
	}

	for _, tc := range tcs {
		pattern := parsePattern(tc.line)
		require.Equal(t, tc.expected, pattern)
	}
}

type patternsTestGroup struct {
	name     string
	patterns string
	tcs      []patternsTC
}

type patternsTC struct {
	pathname git.File
	isDir    bool
	expected bool
}

func (tc patternsTC) blocked(patterns patternSet) (bool, error) {
	pathname := tc.pathname
	isDir := tc.isDir
	for pathname != "." {
		if patterns.ignore(pathname, isDir) {
			return true, nil
		}
		pathname = pathname.Dir()
		isDir = true
	}
	return false, nil
}

func TestPatterns(t *testing.T) {
	groups := []patternsTestGroup{
		patternsTestGroup{
			name: "escape !, #, and space",
			patterns: `
\!alpha
\#beta
gamma\ 
`,
			tcs: []patternsTC{
				patternsTC{
					pathname: "!alpha",
					expected: true,
				},
				patternsTC{
					pathname: "#beta",
					expected: true,
				},
				patternsTC{
					pathname: "gamma ",
					expected: true,
				},
			},
		},
		patternsTestGroup{
			name:     "multiple double stars",
			patterns: "a/b/**/c/d/**/e/f/**/g/h",
			tcs: []patternsTC{
				patternsTC{
					pathname: "a/b/c/d/e/f/g/h",
					expected: true,
				},
				patternsTC{
					pathname: "a/b/x/y/z/c/d/x/y/z/e/f/x/y/z/g/h",
					expected: true,
				},
				patternsTC{
					pathname: "a/b/x/y/z/c/d/x/y/z/e/f/g/h",
					expected: true,
				},
				patternsTC{
					pathname: "a/b/x/y/z/c/d/e/x/y/z/f/x/y/z/g/h",
					expected: false,
				},
				patternsTC{
					pathname: "x/y/z/a/b/x/y/z/c/d/x/y/z/e/f/x/y/z/g/h",
					expected: false,
				},
			},
		},
		// below test cases are from
		// https://www.atlassian.com/git/tutorials/saving-changes/gitignore#git-ignore-patterns
		patternsTestGroup{
			name:     "double star directory",
			patterns: "**/logs",
			tcs: []patternsTC{
				patternsTC{
					pathname: "logs/debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "logs/monday/foo.bar",
					expected: true,
				},
				patternsTC{
					pathname: "build/logs/debug.log",
					expected: true,
				},
			},
		},
		patternsTestGroup{
			name:     "double star directory and name",
			patterns: "**/logs/debug.log",
			tcs: []patternsTC{
				patternsTC{
					pathname: "logs/debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "build/logs/debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "logs/build/debug.log",
					expected: false,
				},
			},
		},
		patternsTestGroup{
			name:     "wildcard star",
			patterns: "*.log",
			tcs: []patternsTC{
				patternsTC{
					pathname: "debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "foo.log",
					expected: true,
				},
				patternsTC{
					pathname: ".log",
					expected: true,
				},
				patternsTC{
					pathname: "logs/debug.log",
					expected: true,
				},
			},
		},
		patternsTestGroup{
			name: "exclamation point negation",
			patterns: `
*.log
!important.log
`,
			tcs: []patternsTC{
				patternsTC{
					pathname: "debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "trace.log",
					expected: true,
				},
				patternsTC{
					pathname: "important.log",
					expected: false,
				},
				patternsTC{
					pathname: "logs/important.log",
					expected: false,
				},
			},
		},
		patternsTestGroup{
			name: "reignore after exclamation point",
			patterns: `
*.log
!important/*.log
trace.*
`,
			tcs: []patternsTC{
				patternsTC{
					pathname: "debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "important/trace.log",
					expected: true,
				},
				patternsTC{
					pathname: "important/debug.log",
					expected: false,
				},
			},
		},
		patternsTestGroup{
			name:     "prepended slash",
			patterns: "/debug.log",
			tcs: []patternsTC{
				patternsTC{
					pathname: "debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "logs/debug.log",
					expected: false,
				},
			},
		},
		patternsTestGroup{
			name:     "files match any directory",
			patterns: "debug.log",
			tcs: []patternsTC{
				patternsTC{
					pathname: "debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "logs/debug.log",
					expected: true,
				},
			},
		},
		patternsTestGroup{
			name:     "question mark matches exactly one character",
			patterns: "debug?.log",
			tcs: []patternsTC{
				patternsTC{
					pathname: "debug0.log",
					expected: true,
				},
				patternsTC{
					pathname: "debugg.log",
					expected: true,
				},
				patternsTC{
					pathname: "debug10.log",
					expected: false,
				},
			},
		},
		patternsTestGroup{
			name:     "numeric character range",
			patterns: "debug[0-9].log",
			tcs: []patternsTC{
				patternsTC{
					pathname: "debug0.log",
					expected: true,
				},
				patternsTC{
					pathname: "debug1.log",
					expected: true,
				},
				patternsTC{
					pathname: "debug10.log",
					expected: false,
				},
			},
		},
		patternsTestGroup{
			name:     "character set",
			patterns: "debug[01].log",
			tcs: []patternsTC{
				patternsTC{
					pathname: "debug0.log",
					expected: true,
				},
				patternsTC{
					pathname: "debug1.log",
					expected: true,
				},
				patternsTC{
					pathname: "debug2.log",
					expected: false,
				},
				patternsTC{
					pathname: "debug01.log",
					expected: false,
				},
			},
		},
		patternsTestGroup{
			name:     "character set complement",
			patterns: "debug[!01].log",
			tcs: []patternsTC{
				patternsTC{
					pathname: "debug2.log",
					expected: true,
				},
				patternsTC{
					pathname: "debug0.log",
					expected: false,
				},
				patternsTC{
					pathname: "debug1.log",
					expected: false,
				},
				patternsTC{
					pathname: "debug01.log",
					expected: false,
				},
			},
		},
		patternsTestGroup{
			name:     "alphabetical character range",
			patterns: "debug[a-z].log",
			tcs: []patternsTC{
				patternsTC{
					pathname: "debuga.log",
					expected: true,
				},
				patternsTC{
					pathname: "debugb.log",
					expected: true,
				},
				patternsTC{
					pathname: "debug1.log",
					expected: false,
				},
			},
		},
		patternsTestGroup{
			name:     "directory anywhere",
			patterns: "logs",
			tcs: []patternsTC{
				patternsTC{
					pathname: "logs",
					isDir:    true,
					expected: true,
				},
				patternsTC{
					pathname: "logs/debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "logs/latest/foo.bar",
					expected: true,
				},
				patternsTC{
					pathname: "build/logs",
					isDir:    true,
					expected: true,
				},
				patternsTC{
					pathname: "build/logs/debug.log",
					expected: true,
				},
			},
		},
		patternsTestGroup{
			name:     "directory only",
			patterns: "logs/",
			tcs: []patternsTC{
				patternsTC{
					pathname: "logs/debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "logs/latest/foo.bar",
					expected: true,
				},
				patternsTC{
					pathname: "build/logs/foo.bar",
					expected: true,
				},
				patternsTC{
					pathname: "build/logs/latest/debug.log",
					expected: true,
				},
			},
		},
		patternsTestGroup{
			name: "cannot negate file in ignored directory",
			patterns: `
logs/
!logs/important.log
`,
			tcs: []patternsTC{
				patternsTC{
					pathname: "logs/debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "logs/important.log",
					expected: true,
				},
			},
		},
		patternsTestGroup{
			name:     "middle double star",
			patterns: "logs/**/debug.log",
			tcs: []patternsTC{
				patternsTC{
					pathname: "logs/debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "logs/monday/debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "logs/monday/pm/debug.log",
					expected: true,
				},
			},
		},
		patternsTestGroup{
			name:     "wildcard star in directory name",
			patterns: "logs/*day/debug.log",
			tcs: []patternsTC{
				patternsTC{
					pathname: "logs/monday/debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "logs/tuesday/debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "logs/latest/debug.log",
					expected: false,
				},
			},
		},
		patternsTestGroup{
			name:     "slash in middle",
			patterns: "logs/debug.log",
			tcs: []patternsTC{
				patternsTC{
					pathname: "logs/debug.log",
					expected: true,
				},
				patternsTC{
					pathname: "debug.log",
					expected: false,
				},
				patternsTC{
					pathname: "build/logs/debug.log",
					expected: false,
				},
			},
		},
	}

	for _, group := range groups {
		m := newMunger()
		munged := m.mungePatterns(group.patterns)
		patterns := parsePatterns(munged)
		for _, tc := range group.tcs {
			ignored, err := tc.blocked(patterns)
			require.NoError(t, err)
			require.Equal(t, tc.expected, ignored, group.name)
		}
	}
}
