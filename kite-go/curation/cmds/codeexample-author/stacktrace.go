package main

import (
	"html/template"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/sandbox"
)

const tracebackHeader = `Traceback (most recent call last):`

var tracebackRegex = regexp.MustCompile(`^  File "([^w]+)", line (\d+), in .*$`)

// Given output from a python program possibly containing a stack trace, return a formatted
// HTML representation of the output, including parts that are not part of the stack trace.
// The returned string is always html-escaped, even if no traceback was found in the string.
func colorizeTraceback(s string, python3 bool) (template.HTML, error) {
	// Resolve path to pygments
	pygments, err := exec.LookPath("pygmentize")
	if err != nil {
		return "", err
	}

	// Pick the pygments lexer corresponding to the version of python being used
	var lexer string
	if python3 {
		lexer = "pytb3"
	} else {
		lexer = "pytb"
	}

	// Run pygments
	stdout, _, err := sandbox.RunCommand([]byte(s), pygments, "-l", lexer, "-f", "html")
	if err != nil {
		return "", err
	}

	return template.HTML(stdout), nil
}

// Represents an error extracted from a traceback
type pythonError struct {
	Line    int    `json:"line"`    // Line number at which error occurred
	Message string `json:"message"` // Error message
	Segment string `json:"segment"` // Name of code segment (prelude / code / postlude)
}

// Find the last (most recent, topmost on stack) line at which an error was reported, or nil
// if no traceback was found, or a traceback was found but no lines matched the given path.
func findLastErrorForPath(path, stderr string) *pythonError {
	var curError pythonError
	var finalError pythonError
	var foundHeader, foundLineNo, foundCompleteError bool
	for _, line := range strings.Split(stderr, "\n") {
		if line == tracebackHeader {
			foundHeader = true
		} else if foundHeader && strings.HasPrefix(line, "  ") {
			if foundLineNo {
				continue
			}
			submatches := tracebackRegex.FindStringSubmatch(line)
			if len(submatches) != 3 {
				continue
			}
			curpath := submatches[1]
			lineno, err := strconv.Atoi(submatches[2])
			if err != nil {
				log.Printf("Failed to parse %s integer in %s. Ignoring.\n", submatches[2], line)
				continue
			}
			if curpath == path {
				curError.Line = lineno - 1 // minus one because pylint line numbers are 1-based
				foundLineNo = true
			}
		} else if foundHeader && foundLineNo && !strings.HasPrefix(line, " ") {
			curError.Message = line
			finalError = curError
			foundCompleteError = true
			break
		} else {
			foundHeader = false
			foundLineNo = false
		}
	}
	if foundCompleteError {
		return &finalError
	}
	return nil
}
