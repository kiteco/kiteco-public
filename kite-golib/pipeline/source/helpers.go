package source

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func logf(w io.Writer, fstr string, args ...interface{}) {
	if w == nil {
		return
	}

	if !strings.HasSuffix(fstr, "\n") {
		fstr += "\n"
	}

	fmt.Fprintf(w, fstr, args...)
}

func isGzipped(r io.Reader) (io.Reader, bool, error) {
	// based on SO link: /questions/28309988/how-to-read-from-either-gzip-or-plain-text-reader-in-golang
	//create a bufio.Reader so we can 'peek' at the first few bytes
	bReader := bufio.NewReader(r)

	testBytes, err := bReader.Peek(64) //read a few bytes without consuming
	if err != nil {
		return nil, false, err
	}

	//Detect if the content is gzipped
	contentType := http.DetectContentType(testBytes)

	return bReader, strings.Contains(contentType, "x-gzip"), nil
}
