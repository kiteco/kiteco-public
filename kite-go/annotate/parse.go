//go:generate go-bindata -pkg annotate -o bindata.go presentation_api/...

package annotate

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kiteco/kiteco/kite-go/lang"
)

const (
	// LineNumMacro is a string that gets replaced with the current line number in code examples
	LineNumMacro = "$LINE$"
)

var (
	// Bash presentation API sources
	bashPresentationAPI = MustAsset("presentation_api/bash/kite.sh")

	// Python presentation API sources
	pythonPresentationAPI        = MustAsset("presentation_api/python/kite.py")
	pythonPresentationEntrypoint = MustAsset("presentation_api/python/entrypoint.py")
)

// A Stencil represents the parsed version of the input source code from which an annotated
// code example is generated.
type Stencil struct {
	Original     string
	Presentation string
	Runnable     string
	Inline       bool
	// LineMap maps
	// the line number in Runnable (the rewritten-source that generates annotations)
	//  to
	// the line number in Presentation (the original source of the code being executed)
	//  i.e. using 0-index:
	// LineMap[i] -> j where i is the line number in Presentation and j is the corresponding
	// line number in Runnable (length of LineMap is lines of code in Presentation)
	LineMap []int
}

// ParseExample parses a code example
func ParseExample(code string, language lang.Language) (*Stencil, error) {
	switch language {
	case lang.Bash:
		return parseBashExample(code)
	case lang.Python:
		return parsePythonExample(code)
	default:
		return parseExampleWithNoAnnotations(code)
	}
}

// Determine whether the string s is a python identifer
func isIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	first, _ := utf8.DecodeRuneInString(s)
	if !unicode.IsLetter(first) {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '_' {
			return false
		}
	}
	return true
}

// Remove the leading "##" from a line if it exists, otherwise return empty string.
// The leader may appear after an arbitrary amount of whitespace.
func trimLeader(line, leader string) string {
	if strings.HasPrefix(strings.TrimSpace(line), leader) {
		pos := strings.Index(line, leader)
		return line[:pos] + strings.TrimSpace(line[pos+2:])
	}
	return ""
}

func parseExampleWithNoAnnotations(code string) (*Stencil, error) {
	// Process the example source
	var present []string
	var run []string
	var lineMap []int
	for _, line := range strings.Split(code, "\n") {
		// TODO: need to deal with the case where we are inside a comment
		line := substitute(line, globalSubstitutions)
		lineMap = append(lineMap, len(run))
		present = append(present, line)
		run = append(run, line)
	}

	parsed := &Stencil{
		Original:     code,
		Runnable:     strings.Join(run, "\n"),
		Presentation: strings.Join(present, "\n"),
		Inline:       true,
		LineMap:      lineMap,
	}

	return parsed, nil
}

// parsePythonExample parses a python code example written with kite annotations
func parsePythonExample(example string) (*Stencil, error) {
	code := "##from kite import kite\n" + example

	// Process the example source
	var present []string
	var run []string
	var lineMap []int
	for _, line := range strings.Split(code, "\n") {
		// TODO: need to deal with the case where we are inside a comment
		line := substitute(line, globalSubstitutions)
		if runline := trimLeader(line, "##"); runline != "" {
			run = append(run, runline)
		} else {
			lineMap = append(lineMap, len(run))
			present = append(present, line)
			run = append(run, line)
		}
	}

	parsed := &Stencil{
		Original:     example,
		Runnable:     strings.Join(run, "\n"),
		Presentation: strings.Join(present, "\n"),
		Inline:       true,
		LineMap:      lineMap,
	}

	return parsed, nil
}

var bashBlockSyntax = map[string]string{
	"if":    "fi",
	"case":  "esac",
	"while": "done",
	"until": "done",
	"for":   "done",
}

func parseBashExample(code string) (*Stencil, error) {
	var present []string
	var run []string
	var lineMap []int

	// When running Python examples for annotation, we do this kind of source-
	// rewriting inside the Python process.  We just do it here in Go for now,
	// since it's easier.
	run = strings.Split(string(bashPresentationAPI), "\n")
	lines := strings.Split(code, "\n")
	for lineNum := 0; lineNum < len(lines); lineNum++ {
		line := lines[lineNum]
		if strings.HasPrefix(line, "##") {
			run = append(run, strings.TrimSpace(line[2:]))
		} else if detectContinuedLine(line) {
			var continuedLines []string
			beginContinuedLine := lineNum
			for detectContinuedLine(lines[lineNum]) {
				continuedLines = append(continuedLines, lines[lineNum])
				lineMap = append(lineMap, len(run)+(lineNum-beginContinuedLine))
				lineNum++
			}
			// before the first continued line we have to print the line number of
			// the last continued line
			run = append(run, fmt.Sprintf("kite_line %d", len(run)+len(continuedLines)))
			present = append(present, continuedLines...)
			run = append(run, continuedLines...)
		} else if _, ending := detectBlockBegin(line); ending != "" {
			beginBlockLine := lineNum
			blockLines := []string{line}
			remainingBlockEndings := []string{ending}
			lineMap = append(lineMap, len(run))
			lineNum++
			for len(remainingBlockEndings) > 0 && lineNum < len(lines) {
				line = lines[lineNum]
				blockLines = append(blockLines, line)
				lineMap = append(lineMap, len(run)+(lineNum-beginBlockLine))
				innermostEndingIndex := len(remainingBlockEndings) - 1
				if strings.TrimSpace(line) == remainingBlockEndings[innermostEndingIndex] {
					remainingBlockEndings = remainingBlockEndings[:innermostEndingIndex]
				} else if _, nestedEnding := detectBlockBegin(line); nestedEnding != "" {
					remainingBlockEndings = append(remainingBlockEndings, nestedEnding)
				}
				lineNum++
			}
			present = append(present, blockLines...)
			// before the first line in the block we have to print the line nmuber
			// of the last line in the block
			run = append(run, fmt.Sprintf("kite_line %d", len(run)+len(blockLines)))
			run = append(run, blockLines...)
		} else {
			lineMap = append(lineMap, len(run))
			present = append(present, line)
			run = append(run, fmt.Sprintf("kite_line %d", len(run)+1))
			run = append(run, line)
		}
	}

	return &Stencil{
		Original:     code,
		Runnable:     strings.Join(run, "\n"),
		Presentation: strings.Join(present, "\n"),
		Inline:       true,
		LineMap:      lineMap,
	}, nil
}

func detectContinuedLine(line string) bool {
	return strings.HasSuffix(line, `\`) && !strings.HasSuffix(line, `\\`)
}

func detectBlockBegin(line string) (string, string) {
	for beginning, ending := range bashBlockSyntax {
		if strings.HasPrefix(strings.TrimSpace(line), beginning) {
			return beginning, ending
		}
	}
	return "", ""
}
