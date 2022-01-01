package text

import (
	"bufio"
	"bytes"
	"strings"
	"unicode"
)

// RemoveSquareBrackets removes square brackets from a string.
func RemoveSquareBrackets(s string) string {
	s = strings.Replace(s, "[", "", -1)
	s = strings.Replace(s, "]", "", -1)
	return s
}

// RemovePunctuations removes puncuations from a string.
func RemovePunctuations(s string) string {
	newStr := []byte(s)
	for i, c := range s {
		if (unicode.IsPunct(c) && !specialCase(s, i)) || IsOperator(c) {
			newStr[i] = ' '
		}
	}
	return string(newStr)
}

// RemoveBackTicks removes back ticks.
func RemoveBackTicks(s string) string {
	s = strings.Replace(s, "`s", "", -1)
	s = strings.Replace(s, "`", "", -1)
	return s
}

// RemoveTrailingSpaces removes trailing spaces of a string.
func RemoveTrailingSpaces(s string) string {
	s = strings.Trim(s, " \n")
	return s
}

// Normalize removes
// 1) trailing spaces
// 2) square brackets
// 3) backticks
// 4) punctuations from a string.
func Normalize(s string) string {
	s = RemoveSquareBrackets(s)
	s = RemoveBackTicks(s)
	s = RemovePunctuations(s)
	s = RemoveTrailingSpaces(s)
	return s
}

// specialCase checks whether the punctuation corresponds to
// a special case that should be skipped.
func specialCase(s string, i int) bool {
	// checks if the punctuation is the slash in I/O
	switch s[i] {
	case '/':
		if i > 0 && i < len(s)-1 {
			return (s[i-1] == 'I' || s[i-1] == 'i') && (s[i+1] == 'O' || s[i+1] == 'o')
		}
	case '_':
		return true
	}
	return false
}

// IgnoreComments ignore lines that start with kite's comment tag "##"
func IgnoreComments(s string) string {
	buf := bytes.NewBufferString(s)
	scanner := bufio.NewScanner(buf)
	scanner.Split(bufio.ScanLines)

	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "##") {
			continue
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// IsOperator returns true if c is an operator.
func IsOperator(c rune) bool {
	switch c {
	case '+', '-', '/', '=', '>', '<', '*':
		return true
	}
	return false
}
