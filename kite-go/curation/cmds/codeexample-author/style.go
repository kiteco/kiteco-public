package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kiteco/kiteco/kite-go/sandbox"
)

var (
	// This regex matches output lines from pylint (but it does not on its own actually perform any linting).
	linterOutputRegex = regexp.MustCompile(
		`^(?P<file>.+?):(?P<line>[0-9]+): \[(?P<code>[A-Z][0-9]+)\((?P<key>.*)\), \] (?P<msg>.*)`)

	segmentNames = []string{"prelude", "code", "postlude"}

	// A list of pylint messages that should be ignored
	pylintWhitelist = []string{
		"C0111", // missing docstring
		"C0103", // invalid constant name
		"E0611", // no name * in *  (often wrong, e.g. for numpy)
		"F0401", // unable to import * (often wrong, e.g. for matplotlib)
	}
)

// styleViolation represents an error reported by pylint
type styleViolation struct {
	Segment  string `json:"segment"` // "prelude", "code", or "postlude"
	Line     int    `json:"line"`
	RuleCode string `json:"rule_code"`
	RuleKey  string `json:"rule_key"`
	Message  string `json:"message"`
}

// checkPythonStyle runs pylint over a code block and returns a list of style violations
func checkPythonStyle(prelude, code, postlude string) ([]styleViolation, error) {
	segments := []string{prelude, code, postlude}
	joined := strings.Join(segments, "\n")

	// Resolve path to pylint
	linter, err := exec.LookPath("pylint")
	if err != nil {
		return nil, err
	}

	// Write code to a file named "src.py" in a temporary directory
	tempdir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempdir)
	srcpath := filepath.Join(tempdir, "src.py")
	ioutil.WriteFile(srcpath, []byte(joined), 0600)

	// Execute linter
	stdout, _, err := sandbox.RunCommand(
		[]byte(joined),
		linter,
		srcpath,
		"--reports=n",
		"--output-format=parseable",
		"--disable="+strings.Join(pylintWhitelist, ","))
	if err != nil {
		// pylint returns nonzero status if it finds violations, which causes an uncleanExit
		if _, ok := err.(*sandbox.UncleanExit); !ok {
			return nil, err
		}
	}

	// Parse output
	violations := parseLinterOutput(string(stdout))
	for i := range violations {
		for j, segment := range segments {
			nlines := strings.Count(segment, "\n") + 1
			if violations[i].Line < nlines {
				violations[i].Segment = segmentNames[j]
				break
			}
			violations[i].Line -= nlines
		}
		if violations[i].Segment == "" {
			violations[i].Segment = "postlude"
		}
	}

	// Return
	return violations, nil
}

// parseLinterOutput parses standard output from pylint and returns a list of style violations
func parseLinterOutput(output string) []styleViolation {
	keys := linterOutputRegex.SubexpNames()

	// Parse errors from stdout
	var violations []styleViolation
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "*") {
			// Indicates a module line
			continue
		}

		submatches := linterOutputRegex.FindStringSubmatch(line)

		if submatches == nil {
			log.Println("Failed to parse line:", line)
			continue
		}

		parts := make(map[string]string)
		for i, submatch := range submatches {
			if i != 0 {
				parts[keys[i]] = submatch
			}
		}

		srcline, err := strconv.Atoi(parts["line"])
		if err != nil {
			log.Println("Could not parse line number as integer:", parts["line"])
			continue
		}

		violations = append(violations, styleViolation{
			Line:     srcline - 1, // minus one because line numbers are 1-based
			RuleKey:  parts["key"],
			RuleCode: parts["code"],
			Message:  parts["msg"],
		})
	}

	return violations
}
