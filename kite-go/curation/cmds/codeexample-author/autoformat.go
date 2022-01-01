package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/kiteco/kiteco/kite-go/sandbox"
)

// The sentinel used to separate code segments during formatting.
// Note that this string is designed so that it parses as a python comment.
const sentinel = "\n# ~~~ SENTINEL ~~~\n"

var (
	// The regexp used to find the sentinel after the autoformatted has potentially changed its indentation
	sentinelRegexp = regexp.MustCompile(`\n *# ~~~ SENTINEL ~~~\n`)

	// List of pep8 rules to whitelist when autoformatting
	autoformatWhitelist = []string{"E301", "E302", "E309", "E265"}
)

type formatter func(code string) (string, error)

func autoformatPythonSegmentsCustom(f formatter, segments ...string) ([]string, error) {
	// Run autoformatter
	joined := strings.Join(segments, sentinel)
	formatted, err := f(joined)
	if err != nil {
		return nil, err
	}

	// Find sentinels (note that autopep8 may have adjusted the indentation so there may be extra whitespace)
	formattedSegments := sentinelRegexp.Split(formatted, len(segments))
	if len(formattedSegments) != len(segments) {
		return nil, fmt.Errorf(
			"started with %d segments but ended with %d, output was:\n%s",
			len(segments),
			len(formattedSegments),
			formatted)
	}

	return formattedSegments, nil
}

func autoformatPythonSegments(segments ...string) ([]string, error) {
	return autoformatPythonSegmentsCustom(autoformatPythonCode, segments...)
}

func autoformatPythonCode(code string) (string, error) {
	// Strip known formatting blobs from start of lines. These may be introduced by
	// e.g. copying and pasting python code from ipython-style documentation.
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, ">>> ") || strings.HasPrefix(line, "... ") {
			lines[i] = line[4:]
		}
	}
	code = strings.Join(lines, "\n")

	// Resolve path to autopep8
	reformatter, err := exec.LookPath("autopep8")
	if err != nil {
		return "", err
	}

	// Execute autopep8
	stdout, _, err := sandbox.RunCommand([]byte(code), reformatter, "--ignore", strings.Join(autoformatWhitelist, ","), "-")
	if err != nil {
		return "", err
	}

	return stdout, nil
}
