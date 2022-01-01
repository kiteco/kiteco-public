package ignore

import (
	"path"
	"strings"

	"github.com/kiteco/kiteco/kite-go/navigation/git"
)

type pattern interface {
	relevant(bool) bool
	ignore(pathname git.File, isDir bool) (bool, error)
}

type patternSet []pattern

func (p patternSet) ignore(pathname git.File, isDir bool) bool {
	var ignore bool
	for _, pattern := range p {
		if !pattern.relevant(ignore) {
			continue
		}
		i, err := pattern.ignore(pathname, isDir)
		// Errors can only come from path.Match which only return path.ErrBadPattern.
		// https://golang.org/src/path/match.go
		// This would occur in production if a .gitignore file has a bad pattern like "beta[a-"
		// We skip these bad patterns, which seems to match the behavior of gitignore.
		if err != nil {
			continue
		}
		ignore = i
	}
	return ignore
}

func parsePatterns(patterns []mungedPattern) patternSet {
	var set patternSet
	for _, pattern := range patterns {
		set = append(set, parsePattern(pattern))
	}
	return set
}

func parsePattern(pattern mungedPattern) pattern {
	if strings.Contains(pattern.body, "**") {
		return parseDoubleStarPattern(pattern)
	}
	return parseSimplePattern(pattern)
}

// Any pattern that doesn't include a double star "**", including these examples from
// https://www.atlassian.com/git/tutorials/saving-changes/gitignore#git-ignore-patterns
// *.log
// !important.log
// !important/*.log
// trace.*
// /debug.log
// debug?.log
// debug[0-9].log
// debug[01].log
// debug[!01].log
// debug[a-z].log
// logs
// logs/
// !logs/important.log
// logs/*day/debug.log
type simplePattern struct {
	inverted bool
	dirOnly  bool
	base     bool
	sequence []string
}

func parseSimplePattern(pattern mungedPattern) simplePattern {
	var p simplePattern
	runes := []rune(pattern.body)
	if runes[len(runes)-1] == '/' {
		p.dirOnly = true
	}
	p.base = !strings.Contains(string(runes[:len(runes)-1]), "/")
	p.inverted = pattern.inverted
	p.sequence = splitSequence(pattern.body)
	return p
}

func (p simplePattern) relevant(ignored bool) bool {
	return p.inverted == ignored
}

func (p simplePattern) ignore(pathname git.File, isDir bool) (bool, error) {
	match, err := p.match(pathname, isDir)
	if err != nil {
		return false, err
	}
	return p.inverted != match, nil
}

func (p simplePattern) match(pathname git.File, isDir bool) (bool, error) {
	if p.dirOnly && !isDir {
		return false, nil
	}
	if p.base {
		return path.Match(p.sequence[0], path.Base(string(pathname)))
	}
	pathnameSequence := splitSequence(string(pathname))
	if len(pathnameSequence) != len(p.sequence) {
		return false, nil
	}
	for i, part := range pathnameSequence {
		match, err := path.Match(p.sequence[i], part)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
	}
	return true, nil
}

func splitSequence(pathname string) []string {
	var sequence []string
	for _, part := range strings.Split(pathname, "/") {
		if part == "" {
			continue
		}
		sequence = append(sequence, part)
	}
	return sequence
}

// Any pattern that includes a double star "**" after munging.
// Note munging replaces double stars with single stars if there is
// a non-separator character on the left or right of the double stars.
// doubleStarPatterns include these examples from
// https://www.atlassian.com/git/tutorials/saving-changes/gitignore#git-ignore-patterns
// **/logs/debug.log
// logs/**/debug.log
type doubleStarPattern struct {
	inverted        bool
	dirOnly         bool
	leftSequence    []string
	rightSequence   []string
	middleSequences [][]string
	totalLength     int
}

func parseDoubleStarPattern(pattern mungedPattern) doubleStarPattern {
	sequences := strings.Split(pattern.body, "**")
	leftSequence := sequences[0]
	rightSequence := sequences[len(sequences)-1]
	rightRunes := []rune(rightSequence)

	var p doubleStarPattern
	for _, sequence := range sequences[1 : len(sequences)-1] {
		split := splitSequence(sequence)
		p.middleSequences = append(p.middleSequences, split)
		p.totalLength += len(split)
	}
	if len(rightRunes) > 0 && rightRunes[len(rightRunes)-1] == '/' {
		p.dirOnly = true
	}
	p.inverted = pattern.inverted
	p.leftSequence = splitSequence(leftSequence)
	p.rightSequence = splitSequence(rightSequence)
	p.totalLength += len(p.leftSequence) + len(p.rightSequence)
	return p
}

func (p doubleStarPattern) relevant(ignored bool) bool {
	return p.inverted == ignored
}

func (p doubleStarPattern) ignore(pathname git.File, isDir bool) (bool, error) {
	match, err := p.match(pathname, isDir)
	if err != nil {
		return false, err
	}
	return p.inverted != match, nil
}

func (p doubleStarPattern) match(pathname git.File, isDir bool) (bool, error) {
	if p.dirOnly && !isDir {
		return false, nil
	}

	pathnameSequence := splitSequence(string(pathname))
	if len(pathnameSequence) < p.totalLength {
		return false, nil
	}

	for i, part := range p.leftSequence {
		match, err := path.Match(part, pathnameSequence[i])
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
	}

	shift := len(pathnameSequence) - len(p.rightSequence)
	for i, part := range p.rightSequence {
		match, err := path.Match(part, pathnameSequence[shift+i])
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
	}

	return matchMiddleSequences(p.middleSequences, pathnameSequence[len(p.leftSequence):shift])
}

func matchMiddleSequences(sequences [][]string, parts []string) (bool, error) {
	if len(sequences) == 0 {
		return true, nil
	}
	if len(parts) < len(sequences[0]) {
		return false, nil
	}
	for i := 0; i <= len(parts)-len(sequences[0]); i++ {
		ok, err := matchSequence(sequences[0], parts[i:i+len(sequences[0])])
		if err != nil {
			return false, err
		}
		if ok {
			return matchMiddleSequences(sequences[1:], parts[i+len(sequences[0]):])
		}
	}
	return false, nil
}

func matchSequence(sequence, parts []string) (bool, error) {
	for i, part := range sequence {
		match, err := path.Match(part, parts[i])
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
	}
	return true, nil
}
