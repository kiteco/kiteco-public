package annotate

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	lineBlob int = iota
	emitBlob
	outputBlob
)

type blob struct {
	Type    int // lineBlob, emitBlob, or outputBlob
	Line    int // meaningful for lineBlob only
	Content string
}

// Parse the raw output from a code example as a series of blobs
func parseBlobs(s string) ([]blob, error) {
	const (
		openDelim  = "[[KITE[["
		closeDelim = "]]KITE]]\n"
		linePrefix = "LINE "
		showPrefix = "SHOW "
	)

	var blobs []blob
	for len(s) > 0 {
		openPos := strings.Index(s, openDelim)
		if openPos == -1 {
			// the delimiter was not found so the rest of the string is a single output blob
			blobs = append(blobs, blob{
				Type:    outputBlob,
				Content: s,
			})
			break
		}
		if openPos > 0 {
			// the delimiter was found so the content up to that position is an output blob
			blobs = append(blobs, blob{
				Type:    outputBlob,
				Content: s[:openPos],
			})
		}

		closePos := strings.Index(s[openPos:], closeDelim)
		if closePos == -1 {
			// the close delimiter was not found even though the open delimeter was found, so
			// the rest of the string must be a single output blob
			blobs = append(blobs, blob{
				Type:    outputBlob,
				Content: s,
			})
			break
		}
		directive := s[openPos+len(openDelim) : openPos+closePos]
		s = s[openPos+closePos+len(closeDelim):]

		if strings.HasPrefix(directive, linePrefix) {
			line, err := strconv.Atoi(directive[len(linePrefix):])
			if err != nil {
				return nil, fmt.Errorf("error parsing line directive '%s': %v", directive, err)
			}
			blobs = append(blobs, blob{
				Type: lineBlob,
				Line: line,
			})
		} else if strings.HasPrefix(directive, showPrefix) {
			blobs = append(blobs, blob{
				Type:    emitBlob,
				Content: directive[len(showPrefix):],
			})
		} else {
			return nil, fmt.Errorf("unrecognized directive: %s", directive)
		}
	}

	return blobs, nil
}
