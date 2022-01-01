package clientlogs

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// loggedClientError implements runtime.error and Rollbar's ErrorStacker
// ErrorStacker allows to provide our own traceback. We provide the stacktrace,
// which was uploaded by the client and not the stack of the reporter, which is running on the server.
type loggedClientError struct {
	message   string
	exception []byte
}

func (e loggedClientError) Error() string {
	return fmt.Sprintf("CrashReport: %s", e.message)
}

func (e loggedClientError) Stack() []runtime.Frame {
	callStack, err := extractCallStack(string(e.exception))
	if err != nil || callStack == nil {
		return nil
	}

	return callStack
}

// see test/*-log.txt for the supported formats
func extractCallStack(stacktrace string) ([]runtime.Frame, error) {
	// extract the part between two blocks of empty lines
	lines := locateFirstStack(strings.Split(stacktrace, "\n"))
	if len(lines) == 0 {
		return nil, nil
	}

	var frames []runtime.Frame
	for len(lines) > 0 {
		var frame *runtime.Frame
		if frame, lines = takeHeadFrame(lines); frame != nil {
			frames = append(frames, *frame)
		}
	}
	return frames, nil
}

func locateFirstStack(lines []string) []string {
	lineCount := len(lines)
	for first := 0; first < lineCount; first++ {
		if lines[first] == "" {
			first++

			// skip lines starting with "goroutine"
			for first < lineCount && strings.HasPrefix(lines[first], "goroutine") {
				first++
			}

			for last := first + 1; last < lineCount; last++ {
				if lines[last] == "" {
					return lines[first:last]
				}
			}

			return lines[first:]
		}
	}

	return nil
}

func takeHeadFrame(lines []string) (*runtime.Frame, []string) {
	if len(lines) < 2 {
		return nil, nil
	}

	head := lines[0:2]
	line1 := head[0]
	line2 := head[1]

	tail := lines[2:]

	var name string

	leftPar := strings.Index(line1, "(")
	rightPar := strings.LastIndex(line1, ")")
	if leftPar <= 0 || rightPar == -1 {
		if !strings.HasPrefix(line1, "created by") {
			return nil, tail
		}
		name = line1
	} else {
		name = line1[0:leftPar]
	}

	colon := strings.LastIndex(line2, ":")
	if colon == -1 {
		return nil, tail
	}

	space := strings.Index(line2[colon:], " ") + colon
	if space <= colon {
		return nil, tail
	}

	filename := strings.Trim(line2[0:colon], " \t")
	line := line2[colon+1 : space]
	lineNo, _ := strconv.Atoi(line)

	return &runtime.Frame{
		PC:       0,
		Function: name,
		File:     filename,
		Line:     lineNo,
	}, tail
}
