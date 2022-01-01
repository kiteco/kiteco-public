package ignore

import (
	"regexp"
	"strings"
)

type munger []replacer

type replacer struct {
	re          *regexp.Regexp
	replacement string
}

func newReplacer(expr, replacement string) replacer {
	return replacer{
		re:          regexp.MustCompile(expr),
		replacement: replacement,
	}
}

func (r replacer) replace(raw string) string {
	return r.re.ReplaceAllString(raw, r.replacement)
}

func newMunger() munger {
	return munger{
		// Trim trailing non-escaped spaces.
		newReplacer(`(\\ )? *`, "$1"),

		// Clean escaped spaces.
		newReplacer(`\\ `, " "),

		// Check if pattern begins with an escaped `#`.
		newReplacer(`^\\#`, "#"),

		// Check if pattern begins with an escaped `!`.
		newReplacer(`^\\!`, "!"),

		// Replace repeated stars with double stars.
		newReplacer("\\*{3,}", "**"),

		// Replace double stars with single stars if there is a non-separator character on the left.
		newReplacer("([^/]\\*)\\*", "$1"),

		// Replace double stars with single stars if there is a non-separator character on the right.
		newReplacer("\\*(\\*[^/])", "$1"),

		// Switch `!` to `^` in character class complements, to fit the path.Match syntax.
		newReplacer(`\[!(.*)\]`, "[^$1]"),
	}
}

func (m munger) mungePatterns(contents string) []mungedPattern {
	clean := strings.Replace(contents, "\r\n", "\n", -1)
	var patterns []mungedPattern
	for _, line := range strings.Split(clean, "\n") {
		munged, ok := m.mungeLine(line)
		if !ok {
			continue
		}
		patterns = append(patterns, munged)
	}
	return patterns
}

type mungedPattern struct {
	inverted bool
	body     string
}

func (m munger) mungeLine(line string) (mungedPattern, bool) {
	if strings.HasPrefix(line, "#") {
		return mungedPattern{}, false
	}
	inverted := strings.HasPrefix(line, "!")
	if inverted {
		line = line[1:]
	}
	for _, replacer := range m {
		line = replacer.replace(line)
	}
	if line == "" {
		return mungedPattern{}, false
	}
	return mungedPattern{
		inverted: inverted,
		body:     line,
	}, true
}
