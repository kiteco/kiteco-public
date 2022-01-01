package pythonlocal

import (
	"bytes"
	"strings"
	"unicode"
)

// DedentDocstring renders a "dedented" docstring for returning to the client
// implement algorithm from https://www.python.org/dev/peps/pep-0257/#handling-docstring-indentation
func DedentDocstring(doc string) string {
	doc = expandtabs(doc)
	if doc == "" {
		return ""
	}

	minIndent := -1
	lines := strings.Split(doc, "\n")
	for _, line := range lines[1:] {
		trimmedLen := len(strings.TrimLeftFunc(line, unicode.IsSpace))
		if trimmedLen == 0 {
			continue
		}
		indent := len(line) - trimmedLen
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}

	lines[0] = strings.TrimSpace(lines[0])
	if minIndent > 0 {
		for i, line := range lines[1:] {
			if minIndent > len(line) {
				line = ""
			} else {
				line = strings.TrimRightFunc(line[minIndent:], unicode.IsSpace)
			}
			lines[i+1] = line
		}
	}

	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}

// implement algorithm from https://docs.python.org/3/library/stdtypes.html#str.expandtabs
func expandtabs(str string) string {
	tabsize := 8

	var b bytes.Buffer // cannot use strings.Builder due to running on old Go version in Windows

	var col int
	for _, c := range str {
		switch c {
		case '\t':
			numSpaces := col % tabsize
			if numSpaces == 0 {
				numSpaces = tabsize
			}
			for i := 0; i < numSpaces; i++ {
				b.WriteRune(' ')
			}
			col += numSpaces
			continue

		case '\n', '\r':
			col = 0
		default:
			col++
		}
		b.WriteRune(c)
	}

	return b.String()
}

func trim(line string) string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, `"`)
	line = strings.Trim(line, `'`)
	return line
}
