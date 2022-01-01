package helpers

import "strings"

// ReadLinesWithComments reads an input line-by-line, with support for comments, and ignoring empty lines
func ReadLinesWithComments(str string, cb func(lineStr, comment string) error) error {
	for _, lineStr := range strings.Split(str, "\n") {
		var comment string
		if commentStart := strings.Index(lineStr, "#"); commentStart > -1 {
			comment = lineStr[commentStart:]
			lineStr = lineStr[:commentStart]
		}
		lineStr = strings.TrimSpace(lineStr)

		if lineStr == "" && comment == "" {
			continue
		}

		if err := cb(lineStr, comment); err != nil {
			return err
		}
	}
	return nil
}
